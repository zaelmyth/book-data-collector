package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/joho/godotenv/autoload"
	"github.com/zaelmyth/book-data-collector/internal/configuration"
	"github.com/zaelmyth/book-data-collector/internal/db"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile) // add code file name and line number to error messages

	config := configuration.Get()

	fmt.Println("Creating openlibrary_id column...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	booksDb := db.GetBooksDatabase(config)
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(booksDb)

	db.CreateOpenLibraryIdColumn(ctx, booksDb)

	fmt.Println("Populating openlibrary_id column...")

	file, err := os.Open(config.File)
	if err != nil {
		log.Fatal(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(file)

	gzipFile, err := gzip.NewReader(file)
	if err != nil {
		log.Fatal(err)
	}
	defer func(fz *gzip.Reader) {
		err := fz.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(gzipFile)

	savedIsbn13s := db.GetSavedDataWithId(ctx, booksDb, "books", "isbn13")
	savedIsbn10s := db.GetSavedDataWithId(ctx, booksDb, "books", "isbn")

	scanner := bufio.NewScanner(gzipFile)
	const maxLineCharacters int = 10000000
	buffer := make([]byte, maxLineCharacters)
	scanner.Buffer(buffer, maxLineCharacters)

	linesCount := 0
	for scanner.Scan() {
		olLine := strings.Split(scanner.Text(), "\t")
		olId := strings.TrimPrefix(olLine[1], "/books/")

		type OpenLibraryEdition struct {
			Isbn13 []string `json:"isbn_13"`
			Isbn10 []string `json:"isbn_10"`
		}
		var openLibraryEdition OpenLibraryEdition
		err = json.Unmarshal([]byte(olLine[4]), &openLibraryEdition)
		if err != nil {
			log.Fatal(err)
		}

		if len(openLibraryEdition.Isbn13) > 0 {
			bookId, isSaved := savedIsbn13s[openLibraryEdition.Isbn13[0]]
			if isSaved {
				db.UpdateOpenLibraryIdColumn(ctx, booksDb, bookId, olId)
				continue
			}
		}

		if len(openLibraryEdition.Isbn10) > 0 {
			bookId, isSaved := savedIsbn10s[openLibraryEdition.Isbn10[0]]
			if isSaved {
				db.UpdateOpenLibraryIdColumn(ctx, booksDb, bookId, olId)
			}
		}

		linesCount++
		if linesCount%1000000 == 0 {
			fmt.Println(fmt.Sprintf("%v lines processed...", linesCount))
		}
	}

	err = scanner.Err()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Done!")
}
