package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"

	_ "github.com/glebarez/go-sqlite"
	_ "github.com/joho/godotenv/autoload"
	"github.com/zaelmyth/book-data-collector/isbndb"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile) //add code file name and line number to error messages

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
	defer func() {
		err := os.Remove("progress-isbndb.sqlite")
		if err != nil {
			log.Fatal(err)
		}
	}()
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(progressDb)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	createTables(ctx, db, progressDb)
	saveBookData(ctx, db, progressDb)
}

type booksSave struct {
	books            []isbndb.Book
	word             string
	isSearchComplete bool
}

func saveBookData(ctx context.Context, db *sql.DB, progressDb *sql.DB) {
	wordsListBytes := getWordsListBytes()
	reader := bytes.NewReader(wordsListBytes)
	countReader := bytes.NewReader(wordsListBytes)

	totalWords := countWords(countReader)
	ticker := time.Tick(time.Second)
	maxCallsPerSecond := getMaxCallsPerSecond()
	limiter := make(chan struct{}, maxCallsPerSecond)
	booksToSave := make(chan booksSave, 100)
	var wg sync.WaitGroup

	go tickGoroutine(ticker, limiter)
	go saveGoroutine(&wg, booksToSave, ctx, db, progressDb)

	scanner := bufio.NewScanner(reader)
	progressCount := 0
	for scanner.Scan() {
		word := scanner.Text()

		isWordSaved := isWordSaved(ctx, progressDb, word)
		if !isWordSaved {
			// added the limiter in the main goroutine instead of search so that it doesn't iterate the whole list of
			// words too fast and occupy memory unnecessarily
			<-limiter
			go searchAndSave(&wg, limiter, booksToSave, ctx, db, progressDb, word, 1)
		}

		progressCount++
		progress := int(float64(progressCount) / float64(totalWords) * 100)
		fmt.Print("\033[H\033[2J") // clear console
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

func getMaxCallsPerSecond() int { // todo: this should be in isbndb package and merged with the other sub info
	validSubscriptionTypes := []string{"basic", "premium", "pro"}
	subscriptionType := os.Getenv("ISBNDB_SUBSCRIPTION_TYPE")

	if !slices.Contains(validSubscriptionTypes, subscriptionType) {
		log.Fatal("Unknown subscription type")
	}

	if subscriptionType == "basic" {
		return 1
	}

	if subscriptionType == "premium" {
		return 3
	}

	return 5
}

func tickGoroutine(ticker <-chan time.Time, limiter chan struct{}) {
	for {
		emptyChannel(limiter) // empty channel to make sure there are no leftovers from last second
		maxCallsPerSecond := getMaxCallsPerSecond()
		for range maxCallsPerSecond {
			limiter <- struct{}{}
		}

		<-ticker
	}
}

func saveGoroutine(wg *sync.WaitGroup, booksToSave chan booksSave, ctx context.Context, db *sql.DB, progressDb *sql.DB) {
	for {
		booksSave := <-booksToSave
		saveBooks(ctx, db, booksSave.books)

		if booksSave.isSearchComplete {
			_, err := progressDb.ExecContext(ctx, `INSERT INTO searched_words (word) VALUES (?)`, booksSave.word)
			if err != nil {
				log.Fatal(err)
			}
		}

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
	limiter chan struct{},
	booksToSave chan booksSave,
	ctx context.Context,
	db *sql.DB,
	progressDb *sql.DB,
	word string,
	page int,
) {
	wg.Add(1)

	pageSize := 2000 // todo: too high for basic subscription
	bookSearchResults := isbndb.SearchBooksByQuery(isbndb.BookSearchByQueryRequest{
		Query:    word,
		Page:     page,
		PageSize: pageSize,
		Column:   "title",
	})

	if len(bookSearchResults.Books) == 0 {
		_, err := progressDb.ExecContext(ctx, `INSERT INTO searched_words (word) VALUES (?)`, word)
		if err != nil {
			log.Fatal(err)
		}

		wg.Done()
		return
	}

	maxPage := int(math.Ceil(float64(bookSearchResults.Total) / float64(pageSize)))
	isSearchComplete := true
	if maxPage > page {
		<-limiter
		go searchAndSave(wg, limiter, booksToSave, ctx, db, progressDb, word, page+1)
		isSearchComplete = false
	}

	booksToSave <- booksSave{
		books:            bookSearchResults.Books,
		word:             word,
		isSearchComplete: isSearchComplete,
	}
}

func emptyChannel(channel chan struct{}) {
	for {
		select {
		case <-channel:
		default:
			return
		}
	}
}
