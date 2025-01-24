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
	"strconv"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/joho/godotenv/autoload"
	"github.com/zaelmyth/book-data-collector/isbndb"
)

const timeoutMinutes = 2

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile) //add code file name and line number to error messages

	mysqlConnectionString := getMysqlConnectionString()
	db, err := sql.Open("mysql", mysqlConnectionString)
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(db)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbNameBooks := os.Getenv("DB_NAME_BOOKS")
	dbNameProgress := os.Getenv("DB_NAME_PROGRESS")
	if dbNameBooks == "" {
		dbNameBooks = "book_data_isbndb"
	}
	if dbNameProgress == "" {
		dbNameProgress = "progress_isbndb"
	}

	createDatabases(ctx, db, dbNameBooks, dbNameProgress)

	// the database has to be declared in the connection instead of with a "USE" statement because of concurrency issues
	bookDb, err := sql.Open("mysql", mysqlConnectionString+dbNameBooks)
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(bookDb)

	progressDb, err := sql.Open("mysql", mysqlConnectionString+dbNameProgress)
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(progressDb)

	createProgressTables(ctx, progressDb)
	createBookTables(ctx, bookDb)
	saveBookData(ctx, bookDb, progressDb)
}

func getMysqlConnectionString() string {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUsername := os.Getenv("DB_USERNAME")
	dbPassword := os.Getenv("DB_PASSWORD")

	if dbHost == "" || dbPort == "" || dbUsername == "" || dbPassword == "" {
		log.Fatal("Database variables are not set")
	}

	return dbUsername + ":" + dbPassword + "@tcp(" + dbHost + ":" + dbPort + ")/"
}

type booksSave struct {
	books            []isbndb.Book
	word             string
	isSearchComplete bool
}

type savedData struct {
	books      map[string]struct{}
	booksMutex *sync.RWMutex
}

func (savedData *savedData) addBook(isbn string) {
	savedData.booksMutex.Lock()
	defer savedData.booksMutex.Unlock()

	savedData.books[isbn] = struct{}{}
}

func (savedData *savedData) isBookSaved(isbn string) bool {
	savedData.booksMutex.RLock()
	defer savedData.booksMutex.RUnlock()

	_, isSaved := savedData.books[isbn]

	return isSaved
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

	dbConcurrentWriteGoroutines := os.Getenv("DB_CONCURRENT_WRITE_GOROUTINES")
	if dbConcurrentWriteGoroutines == "" {
		dbConcurrentWriteGoroutines = "1"
	}
	dbConcurrentWriteGoroutinesInt, err := strconv.Atoi(dbConcurrentWriteGoroutines)
	if err != nil {
		log.Fatal(err)
	}

	go tickGoroutine(&wg, mainSearchQueries, pageSearchQueries, booksToSave, ctx, progressDb)

	savedData := savedData{
		books:      getSavedIsbns(ctx, db),
		booksMutex: &sync.RWMutex{},
	}

	for range dbConcurrentWriteGoroutinesInt {
		go saveGoroutine(&wg, booksToSave, ctx, db, progressDb, savedData)
	}

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
		fmt.Print("\033[H\033[2J") // clear console
		fmt.Printf("Collecting... %v / %v | %v%%\n", progressCount, totalWords, progress)
	}

	err = scanner.Err()
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
	timeoutLimiter := make(chan struct{}, 100)

	ticker := time.Tick(time.Second)
	for range ticker {
		if len(timeoutLimiter) > 0 || len(booksToSave) == cap(booksToSave) {
			continue // give it time to save some books in DB so we don't occupy too much memory unnecessarily
		}

		maxCallsPerSecond := isbndb.GetSubscriptionParams().MaxCallsPerSecond
		for range maxCallsPerSecond {
			go searchAndSave(wg, timeoutLimiter, mainSearchQueries, pageSearchQueries, booksToSave, ctx, progressDb)
		}
	}
}

func saveGoroutine(
	wg *sync.WaitGroup,
	booksToSave chan booksSave,
	ctx context.Context,
	db *sql.DB,
	progressDb *sql.DB,
	savedData savedData,
) {
	for booksSave := range booksToSave {
		saveBooks(ctx, db, booksSave.books, savedData)

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
	timeoutLimiter chan struct{},
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

	bookSearchResults, responseStatusCode := isbndb.SearchBooksByQuery(searchQuery)
	if responseStatusCode == http.StatusGatewayTimeout {
		timeoutLimiter <- struct{}{}
		time.Sleep(timeoutMinutes * time.Minute)

		bookSearchResults, responseStatusCode = isbndb.SearchBooksByQuery(searchQuery)
		if responseStatusCode == http.StatusGatewayTimeout {
			log.Fatal("Request timeout")
		}

		for len(timeoutLimiter) > 0 {
			<-timeoutLimiter
		}
	}

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
