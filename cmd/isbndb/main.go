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
	"strconv"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/joho/godotenv/autoload"
	"github.com/zaelmyth/book-data-collector/isbndb"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile) // add code file name and line number to error messages

	// todo: words file is in .env but isbn list file is here; apply consistency
	isbnListFileFlag := flag.String("isbn-file", "", "ISBN list file location")
	flag.Parse()

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
	bookDb, err := sql.Open("mysql", mysqlConnectionString+dbNameBooks+"?charset=utf8mb4")
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(bookDb)

	progressDb, err := sql.Open("mysql", mysqlConnectionString+dbNameProgress+"?charset=utf8mb4")
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
	saveBookData(*isbnListFileFlag, ctx, bookDb, progressDb)
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

func saveBookData(isbnListFile string, ctx context.Context, db *sql.DB, progressDb *sql.DB) {
	var fileBytes []byte
	var err error
	if isbnListFile == "" {
		fileBytes = getWordsListBytes()
	} else {
		fileBytes, err = os.ReadFile(isbnListFile)
		if err != nil {
			log.Fatal(err)
		}
	}
	countReader := bytes.NewReader(fileBytes)
	totalLines := countLines(countReader)

	mainSearchQueries := make(chan isbndb.BookSearchByQueryRequest, 10)
	defer close(mainSearchQueries)
	pageSearchQueries := make(chan isbndb.BookSearchByQueryRequest, 100)
	defer close(pageSearchQueries)
	isbnQueries := make(chan []string, 10)
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

	timeout := os.Getenv("ISBNDB_TIMEOUT")
	if timeout == "" {
		timeout = "1"
	}
	timeoutInt, err := strconv.Atoi(timeout)
	if err != nil {
		log.Fatal(err)
	}

	callsPerSecond := os.Getenv("ISBNDB_CALLS_PER_SECOND")
	if callsPerSecond == "" {
		callsPerSecond = "0"
	}
	callsPerSecondInt, err := strconv.Atoi(callsPerSecond)
	if err != nil {
		log.Fatal(err)
	}
	if callsPerSecondInt == 0 {
		callsPerSecondInt = isbndb.GetSubscriptionParams().MaxCallsPerSecond
	}

	go tickGoroutine(&wg, mainSearchQueries, pageSearchQueries, isbnQueries, booksToSave, ctx, progressDb, timeoutInt, callsPerSecondInt)

	savedData := savedData{
		books:           getSavedIsbns(ctx, db),
		booksMutex:      &sync.Mutex{},
		authors:         getSavedData(ctx, db, "authors"),
		authorsMutex:    &sync.Mutex{},
		subjects:        getSavedData(ctx, db, "subjects"),
		subjectsMutex:   &sync.Mutex{},
		publishers:      getSavedData(ctx, db, "publishers"),
		publishersMutex: &sync.Mutex{},
		languages:       getSavedData(ctx, db, "languages"),
		languagesMutex:  &sync.Mutex{},
	}

	for range dbConcurrentWriteGoroutinesInt {
		go saveGoroutine(&wg, booksToSave, ctx, db, progressDb, savedData)
	}

	reader := bytes.NewReader(fileBytes)
	scanner := bufio.NewScanner(reader)

	if isbnListFile == "" {
		handleSearchByKeyword(scanner, ctx, progressDb, mainSearchQueries, totalLines)
	} else {
		handleSearchByIsbn(scanner, isbnQueries, totalLines)
	}

	fmt.Println("Waiting for remaining data to be saved to the database...")
	wg.Wait()
	fmt.Println("Done!")
}

func handleSearchByKeyword(scanner *bufio.Scanner, ctx context.Context, progressDb *sql.DB, mainSearchQueries chan isbndb.BookSearchByQueryRequest, totalWords int) {
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

	err := scanner.Err()
	if err != nil {
		log.Fatal(err)
	}
}

func handleSearchByIsbn(scanner *bufio.Scanner, isbnQueries chan []string, totalIsbns int) {
	var isbns []string
	progressCount := 0
	for scanner.Scan() {
		isbn := scanner.Text()

		// todo: check if isbn is not already saved
		isbns = append(isbns, isbn)
		if len(isbns) == 1000 {
			isbnQueries <- isbns
			isbns = nil
		}

		progressCount++
		progress := int(float64(progressCount) / float64(totalIsbns) * 100)
		fmt.Print("\033[H\033[2J") // clear console
		fmt.Printf("Collecting... %v / %v | %v%%\n", progressCount, totalIsbns, progress)
	}

	if len(isbns) > 0 {
		isbnQueries <- isbns
	}

	err := scanner.Err()
	if err != nil {
		log.Fatal(err)
	}
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

func countLines(reader io.Reader) int {
	totalLines := 0
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		totalLines++
	}

	return totalLines
}

func tickGoroutine(
	wg *sync.WaitGroup,
	mainSearchQueries chan isbndb.BookSearchByQueryRequest,
	pageSearchQueries chan isbndb.BookSearchByQueryRequest,
	isbnQueries chan []string,
	booksToSave chan booksSave,
	ctx context.Context,
	progressDb *sql.DB,
	timeout int,
	callsPerSecond int,
) {
	timeoutLimiter := make(chan struct{}, 100)

	ticker := time.Tick(time.Second)
	for range ticker {
		if len(timeoutLimiter) > 0 || len(booksToSave) == cap(booksToSave) {
			continue // give it time to save some books in DB so we don't occupy too much memory unnecessarily
		}

		for range callsPerSecond {
			if len(isbnQueries) > 0 {
				go searchAndSaveByIsbn(wg, timeoutLimiter, isbnQueries, booksToSave, timeout)
			} else {
				go searchAndSaveByKeyword(wg, timeoutLimiter, mainSearchQueries, pageSearchQueries, booksToSave, ctx, progressDb, timeout)
			}
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

func searchAndSaveByKeyword(
	wg *sync.WaitGroup,
	timeoutLimiter chan struct{},
	mainSearchQueries chan isbndb.BookSearchByQueryRequest,
	pageSearchQueries chan isbndb.BookSearchByQueryRequest,
	booksToSave chan booksSave,
	ctx context.Context,
	progressDb *sql.DB,
	timeout int,
) {
	var searchQuery isbndb.BookSearchByQueryRequest
	var ok bool

	select {
	case searchQuery, ok = <-pageSearchQueries:
	default:
	}

	if !ok {
		searchQuery = <-mainSearchQueries
	}

	wg.Add(1)

	bookSearchResults, responseStatusCode := isbndb.SearchBooksByQuery(searchQuery)
	if responseStatusCode == http.StatusGatewayTimeout {
		timeoutLimiter <- struct{}{}
		time.Sleep(time.Duration(timeout) * time.Minute)

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

func searchAndSaveByIsbn(
	wg *sync.WaitGroup,
	timeoutLimiter chan struct{},
	isbnQueries chan []string,
	booksToSave chan booksSave,
	timeout int,
) {
	isbns := <-isbnQueries

	wg.Add(1)

	bookSearchResults, responseStatusCode := isbndb.SearchBooksByIsbn(isbns)
	if responseStatusCode == http.StatusGatewayTimeout {
		timeoutLimiter <- struct{}{}
		time.Sleep(time.Duration(timeout) * time.Minute)

		bookSearchResults, responseStatusCode = isbndb.SearchBooksByIsbn(isbns)
		if responseStatusCode == http.StatusGatewayTimeout {
			log.Fatal("Request timeout")
		}

		for len(timeoutLimiter) > 0 {
			<-timeoutLimiter
		}
	}

	booksToSave <- booksSave{
		books:            bookSearchResults.Data,
		word:             "",
		isSearchComplete: false,
	}
}
