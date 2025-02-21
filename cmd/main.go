package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/joho/godotenv/autoload"
	"github.com/zaelmyth/book-data-collector/google"
	"github.com/zaelmyth/book-data-collector/internal/configuration"
	"github.com/zaelmyth/book-data-collector/internal/db"
	"github.com/zaelmyth/book-data-collector/isbndb"
)

const searchRetryLimit = 3 // todo: make configurable

type searchQuery struct {
	query string
	page  int
	isbns []string
}

type booksSave struct {
	books            []isbndb.Book
	volumes          []google.Volume
	word             string
	isSearchComplete bool
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile) // add code file name and line number to error messages

	config := configuration.Get()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db.CreateDatabases(ctx, config)

	booksDb := db.GetBooksDatabase(config)
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(booksDb)

	progressDb := db.GetProgressDatabase(config)
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(progressDb)

	db.CreateProgressTables(ctx, progressDb)
	db.CreateBookTables(ctx, booksDb)

	saveBookData(config, ctx, booksDb, progressDb)
}

func saveBookData(config configuration.Config, ctx context.Context, booksDb *sql.DB, progressDb *sql.DB) {
	file, err := os.ReadFile(config.File)
	if err != nil {
		log.Fatal(err)
	}

	queries := make(chan searchQuery, 10)
	defer close(queries)

	priorityQueries := make(chan searchQuery, 100)
	defer close(priorityQueries)

	booksToSave := make(chan booksSave, 10)
	defer close(booksToSave)

	var wg sync.WaitGroup

	go searchGoroutine(&wg, config, queries, priorityQueries, booksToSave, ctx, progressDb)

	bookColumnId := "isbn13"
	if config.Provider == "google" {
		bookColumnId = "google_id"
	}

	savedData := db.SavedData{
		Books:           db.GetSavedData(ctx, booksDb, "books", bookColumnId),
		BooksMutex:      &sync.RWMutex{},
		Authors:         db.GetSavedDataWithId(ctx, booksDb, "authors", "name"),
		AuthorsMutex:    &sync.Mutex{},
		Subjects:        db.GetSavedDataWithId(ctx, booksDb, "subjects", "name"),
		SubjectsMutex:   &sync.Mutex{},
		Publishers:      db.GetSavedDataWithId(ctx, booksDb, "publishers", "name"),
		PublishersMutex: &sync.Mutex{},
		Languages:       db.GetSavedDataWithId(ctx, booksDb, "languages", "name"),
		LanguagesMutex:  &sync.Mutex{},
		Queries:         db.GetSavedData(ctx, progressDb, "searched_queries", "query"),
		QueriesMutex:    &sync.RWMutex{},
	}
	for range config.DbConcurrentWriteGoroutines {
		go saveGoroutine(&wg, booksToSave, ctx, booksDb, progressDb, savedData)
	}

	linesCount := getLinesCount(file)

	reader := bytes.NewReader(file)
	scanner := bufio.NewScanner(reader)
	progressCount := 0
	var isbns []string
	for scanner.Scan() {
		query := scanner.Text()

		querySaved := savedData.IsQuerySaved(query)
		if !querySaved {
			if config.Provider == "isbndb" && config.SearchBy == "isbn" {
				// todo: invalid isbns get retried every time so we should probably save what isbns we have already tried
				isbns = append(isbns, query)
				if len(isbns) == 1000 {
					queries <- searchQuery{
						page:  1,
						isbns: isbns,
					}
					isbns = nil
					wg.Add(1)
				}
			} else {
				queries <- searchQuery{
					query: query,
					page:  1,
				}
				wg.Add(1)
			}
		}

		if config.Provider == "isbndb" && config.SearchBy == "isbn" && len(isbns) > 0 {
			queries <- searchQuery{
				page:  1,
				isbns: isbns,
			}
			wg.Add(1)
		}

		progressCount++
		progress := int(float64(progressCount) / float64(linesCount) * 100)
		fmt.Print("\033[H\033[2J") // clear console
		fmt.Printf("Collecting... %v / %v | %v%%\n", progressCount, linesCount, progress)
	}

	err = scanner.Err()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Waiting for remaining data to be saved to the database...")
	wg.Wait()
	fmt.Println("Done!")
}

func searchGoroutine(
	wg *sync.WaitGroup,
	config configuration.Config,
	queries chan searchQuery,
	priorityQueries chan searchQuery,
	booksToSave chan booksSave,
	ctx context.Context,
	progressDb *sql.DB,
) {
	timeoutLimiter := make(chan struct{}, 100) //todo: refactor limiter to a mutex
	for {
		if len(timeoutLimiter) == 0 && len(booksToSave) < cap(booksToSave) {
			for range config.CallsPerSecond {
				go search(wg, config, timeoutLimiter, queries, priorityQueries, booksToSave, ctx, progressDb)
			}
		}

		time.Sleep(time.Second)
	}
}

func saveGoroutine(
	wg *sync.WaitGroup,
	booksToSave chan booksSave,
	ctx context.Context,
	booksDb *sql.DB,
	progressDb *sql.DB,
	savedData db.SavedData,
) {
	for booksSave := range booksToSave {
		for _, book := range booksSave.books {
			db.SaveBook(ctx, booksDb, book, savedData)
		}
		for _, volume := range booksSave.volumes {
			db.SaveVolume(ctx, booksDb, volume, savedData)
		}

		if booksSave.isSearchComplete {
			savedData.SaveQuery(ctx, progressDb, booksSave.word)
		}

		wg.Done()
	}
}

func search(
	wg *sync.WaitGroup,
	config configuration.Config,
	timeoutLimiter chan struct{},
	queries chan searchQuery,
	priorityQueries chan searchQuery,
	booksToSave chan booksSave,
	ctx context.Context,
	progressDb *sql.DB,
) {
	// todo: refactor this

	query, ok := getNextQuery(priorityQueries, queries)
	if !ok {
		return
	}

	for range searchRetryLimit {
		if config.Provider == "google" {
			if config.SearchBy == "title" {
				config.SearchBy = "intitle"
			}
			results, statusCode := google.Search(config.SearchBy+":"+query.query, google.SearchParameters{
				Filter:     "full",
				StartIndex: (query.page - 1) * google.MaxPageSize,
				MaxResults: google.MaxPageSize,
				PrintType:  "books",
				Projection: "full",
			})

			if shouldTimeout(statusCode) {
				handleTimeout(timeoutLimiter, config)
				continue
			}

			if len(results.Items) == 0 {
				handleNoResults(wg, progressDb, ctx, query)
				return
			}

			isComplete := false
			if config.SearchBy != "isbn" {
				isComplete = isSearchComplete(wg, results.TotalItems, google.MaxPageSize, query, priorityQueries)
			}

			booksToSave <- booksSave{
				volumes:          results.Items,
				word:             query.query,
				isSearchComplete: isComplete,
			}

			return
		}

		if config.SearchBy == "title" || config.SearchBy == "subject" {
			if config.SearchBy == "subject" {
				config.SearchBy = "subjects"
			}
			results, statusCode := isbndb.SearchBooksByQuery(isbndb.BookSearchByQueryRequest{
				Query:    query.query,
				Page:     query.page,
				PageSize: isbndb.MaxPageSize,
				Column:   config.SearchBy,
			})

			if shouldTimeout(statusCode) {
				handleTimeout(timeoutLimiter, config)
				continue
			}

			if len(results.Books) == 0 {
				handleNoResults(wg, progressDb, ctx, query)
				return
			}

			isSearchComplete := isSearchComplete(wg, results.Total, isbndb.MaxPageSize, query, priorityQueries)

			booksToSave <- booksSave{
				books:            results.Books,
				word:             query.query,
				isSearchComplete: isSearchComplete,
			}

			return
		}

		results, statusCode := isbndb.SearchBooksByIsbn(query.isbns)

		if shouldTimeout(statusCode) {
			handleTimeout(timeoutLimiter, config)
			continue
		}

		if len(results.Data) == 0 {
			handleNoResults(wg, progressDb, ctx, query)
			return
		}

		booksToSave <- booksSave{
			books:            results.Data,
			word:             query.query,
			isSearchComplete: false,
		}

		return
	}

	log.Fatal("Timeout! All retries failed!")
}

func getLinesCount(file []byte) int {
	reader := bytes.NewReader(file)
	scanner := bufio.NewScanner(reader)

	totalLines := 0
	for scanner.Scan() {
		totalLines++
	}

	return totalLines
}

func getNextQuery(priorityQueries chan searchQuery, queries chan searchQuery) (searchQuery, bool) {
	var query searchQuery
	var ok bool

	select {
	case query, ok = <-priorityQueries:
	default:
	}

	if !ok {
		select {
		case query, ok = <-queries:
		default:
		}
	}

	return query, ok
}

func shouldTimeout(statusCode int) bool {
	return statusCode == http.StatusGatewayTimeout || statusCode == http.StatusTooManyRequests
}

func handleTimeout(timeoutLimiter chan struct{}, config configuration.Config) {
	timeoutLimiter <- struct{}{}
	time.Sleep(time.Duration(config.TimeoutSeconds) * time.Second)
	for len(timeoutLimiter) > 0 {
		<-timeoutLimiter
	}
}

func handleNoResults(wg *sync.WaitGroup, progressDb *sql.DB, ctx context.Context, query searchQuery) {
	_, err := progressDb.ExecContext(ctx, `INSERT INTO searched_queries (query) VALUES (?)`, query.query)
	if err != nil {
		log.Fatal(err)
	}

	wg.Done()
}

func isSearchComplete(wg *sync.WaitGroup, resultsCount int, maxPageSize int, query searchQuery, priorityQueries chan searchQuery) bool {
	maxPage := int(math.Ceil(float64(resultsCount) / float64(maxPageSize)))
	isSearchComplete := query.page == maxPage

	if !isSearchComplete {
		query.page++
		priorityQueries <- query
		wg.Add(1)
	}

	return isSearchComplete
}
