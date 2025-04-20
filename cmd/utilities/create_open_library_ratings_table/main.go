package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/joho/godotenv/autoload"
	"github.com/zaelmyth/book-data-collector/internal/configuration"
	"github.com/zaelmyth/book-data-collector/internal/db"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile) // add code file name and line number to error messages

	config := configuration.Get()

	fmt.Println("Creating open_library_ratings table...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	booksDb := db.GetBooksDatabase(config)
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(booksDb)

	db.CreateOpenLibraryRatingsTable(ctx, booksDb)

	fmt.Println("Populating open_library_ratings table...")

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

	savedBooks := db.GetSavedDataWithId(ctx, booksDb, "books", "open_library_id")

	scanner := bufio.NewScanner(gzipFile)
	const maxLineCharacters int = 10000000
	buffer := make([]byte, maxLineCharacters)
	scanner.Buffer(buffer, maxLineCharacters)

	linesCount := 0
	for scanner.Scan() {
		olLine := strings.Split(scanner.Text(), "\t")
		olId := strings.TrimPrefix(olLine[1], "/books/")

		if olId == "" {
			continue
		}

		bookId, isSaved := savedBooks[olId]
		if isSaved {
			rating, err := strconv.ParseFloat(olLine[2], 64)
			if err != nil {
				log.Fatal(err)
			}
			db.SaveOpenLibraryRatings(ctx, booksDb, rating, olLine[3], bookId)
		}

		linesCount++
		if linesCount%100000 == 0 {
			fmt.Println(fmt.Sprintf("%v lines processed...", linesCount))
		}
	}

	err = scanner.Err()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Done!")
}
