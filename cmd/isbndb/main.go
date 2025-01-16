package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	_ "github.com/glebarez/go-sqlite"
	_ "github.com/joho/godotenv/autoload"
	"github.com/zaelmyth/book-data-collector/isbndb"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile) //add code file name and line number to error messages

	keepProgressFlag := flag.Bool("keep-progress", false, "Boolean flag to keep the progress database. Default is false.")
	flag.Parse()

	db, err := sql.Open("sqlite", "book-data-isbndb.sqlite?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(db)

	progressDb, err := sql.Open("sqlite", "progress-isbndb.sqlite?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}

		if !*keepProgressFlag {
			err = os.Remove("progress-isbndb.sqlite")
			if err != nil {
				log.Fatal(err)
			}
		}
	}(progressDb)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	createProgressTables(ctx, progressDb)
	createBookTables(ctx, db)
	saveBookData(ctx, db, progressDb)
}

type booksSave struct {
	books            []isbndb.Book
	word             string
	isSearchComplete bool
}

func saveBookData(ctx context.Context, db *sql.DB, progressDb *sql.DB) {
	wordsListBytes := getWordsListBytes()
	countReader := bytes.NewReader(wordsListBytes)
	totalWords := countWords(countReader)

	mainSearchQueries := make(chan isbndb.BookSearchByQueryRequest, 10)
	defer close(mainSearchQueries)
	pageSearchQueries := make(chan isbndb.BookSearchByQueryRequest, 100)
	defer close(pageSearchQueries)
	booksToSave := make(chan booksSave, 10)
	defer close(booksToSave)

	var wg sync.WaitGroup

	go tickGoroutine(&wg, mainSearchQueries, pageSearchQueries, booksToSave, ctx, progressDb)
	go saveGoroutine(&wg, booksToSave, ctx, db, progressDb)

	reader := bytes.NewReader(wordsListBytes)
	scanner := bufio.NewScanner(reader)
	progressCount := 0
	for scanner.Scan() {
		word := scanner.Text()

		isWordSaved := isWordSaved(ctx, progressDb, word)
		if !isWordSaved {
			mainSearchQueries <- isbndb.BookSearchByQueryRequest{
				Query:    word,
				Page:     1,
				PageSize: isbndb.MaxPageSize,
				Column:   "title",
			}
		}

		progressCount++
		progress := int(float64(progressCount) / float64(totalWords) * 100)
		//fmt.Print("\033[H\033[2J") // clear console
		fmt.Printf("Collecting... %v / %v | %v%%\n", progressCount, totalWords, progress)
	}

	err := scanner.Err()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Waiting for remaining data to be saved to the database...")
	wg.Wait()
	fmt.Println("Done!")
}

func getWordsListBytes() []byte {
	wordsListUrl := os.Getenv("WORDS_LIST_URL")
	if wordsListUrl == "" {
		log.Fatal("WORDS_LIST_URL is not set")
	}

	response, err := http.Get(wordsListUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(response.Body)

	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, response.Body)
	if err != nil {
		log.Fatal(err)
	}

	return buffer.Bytes()
}

func countWords(reader io.Reader) int {
	totalWords := 0
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		totalWords++
	}

	return totalWords
}

func tickGoroutine(
	wg *sync.WaitGroup,
	mainSearchQueries chan isbndb.BookSearchByQueryRequest,
	pageSearchQueries chan isbndb.BookSearchByQueryRequest,
	booksToSave chan booksSave,
	ctx context.Context,
	progressDb *sql.DB,
) {
	ticker := time.Tick(time.Second)
	for range ticker {
		if len(booksToSave) == cap(booksToSave) {
			continue // give it time to save some books in DB so we don't occupy too much memory unnecessarily
		}

		maxCallsPerSecond := isbndb.GetSubscriptionParams().MaxCallsPerSecond
		for range maxCallsPerSecond {
			go searchAndSave(wg, mainSearchQueries, pageSearchQueries, booksToSave, ctx, progressDb)
		}
	}
}

func saveGoroutine(wg *sync.WaitGroup, booksToSave chan booksSave, ctx context.Context, db *sql.DB, progressDb *sql.DB) {
	for booksSave := range booksToSave {
		fmt.Printf("Saving: %v\n", booksSave.word)
		saveBooks(ctx, db, booksSave.books)

		if booksSave.isSearchComplete {
			_, err := progressDb.ExecContext(ctx, `INSERT INTO searched_words (word) VALUES (?)`, booksSave.word)
			if err != nil {
				log.Fatal(err)
			}
		}

		fmt.Printf("Saved: %v\n", booksSave.word)
		wg.Done()
	}
}

func isWordSaved(ctx context.Context, progressDb *sql.DB, word string) bool {
	var searchedWord string
	err := progressDb.QueryRowContext(ctx, `SELECT word FROM searched_words WHERE word = ?`, word).Scan(&searchedWord)

	if errors.Is(err, sql.ErrNoRows) {
		return false
	} else if err != nil {
		log.Fatal(err)
	}

	return true
}

func searchAndSave(
	wg *sync.WaitGroup,
	mainSearchQueries chan isbndb.BookSearchByQueryRequest,
	pageSearchQueries chan isbndb.BookSearchByQueryRequest,
	booksToSave chan booksSave,
	ctx context.Context,
	progressDb *sql.DB,
) {
	wg.Add(1)

	var searchQuery isbndb.BookSearchByQueryRequest
	var ok bool

	select {
	case searchQuery, ok = <-pageSearchQueries:
	default:
	}

	if !ok {
		searchQuery = <-mainSearchQueries
	}

	fmt.Printf("Searching: %v %v\n", searchQuery.Query, searchQuery.Page)
	bookSearchResults, _ := isbndb.SearchBooksByQuery(searchQuery)
	fmt.Printf("Finished: %v %v\n", searchQuery.Query, searchQuery.Page)

	if len(bookSearchResults.Books) == 0 {
		_, err := progressDb.ExecContext(ctx, `INSERT INTO searched_words (word) VALUES (?)`, searchQuery.Query)
		if err != nil {
			log.Fatal(err)
		}

		wg.Done()
		return
	}

	maxPage := int(math.Ceil(float64(bookSearchResults.Total) / float64(isbndb.MaxPageSize)))
	isSearchComplete := true
	if maxPage > searchQuery.Page {
		searchQuery.Page = searchQuery.Page + 1
		pageSearchQueries <- searchQuery
		isSearchComplete = false
	}

	booksToSave <- booksSave{
		books:            bookSearchResults.Books,
		word:             searchQuery.Query,
		isSearchComplete: isSearchComplete,
	}
}
