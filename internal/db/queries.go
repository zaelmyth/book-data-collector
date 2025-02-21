package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/zaelmyth/book-data-collector/google"
	"github.com/zaelmyth/book-data-collector/isbndb"
)

func GetSavedIsbns(ctx context.Context, db *sql.DB) map[string]struct{} {
	rows, err := db.QueryContext(ctx, `SELECT isbn13 FROM books`)
	if err != nil {
		log.Fatal(err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(rows)

	savedIsbns := make(map[string]struct{})
	var isbn string

	for rows.Next() {
		err := rows.Scan(&isbn)
		if err != nil {
			log.Fatal(err)
		}

		savedIsbns[isbn] = struct{}{}
	}

	return savedIsbns
}

func GetSavedData(ctx context.Context, db *sql.DB, dataType string) map[string]int {
	validDataTypes := []string{"authors", "subjects", "publishers", "languages"}
	if !slices.Contains(validDataTypes, dataType) {
		log.Fatal("Invalid data type")
	}

	rows, err := db.QueryContext(ctx, `SELECT id, name FROM `+dataType)
	if err != nil {
		log.Fatal(err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(rows)

	savedData := make(map[string]int)
	var id int
	var name string

	for rows.Next() {
		err := rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}

		savedData[name] = id
	}

	return savedData
}

func GetSavedQueries(ctx context.Context, db *sql.DB) map[string]struct{} {
	rows, err := db.QueryContext(ctx, `SELECT query FROM searched_queries`)
	if err != nil {
		log.Fatal(err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(rows)

	savedQueries := make(map[string]struct{})
	var query string

	for rows.Next() {
		err := rows.Scan(&query)
		if err != nil {
			log.Fatal(err)
		}

		savedQueries[query] = struct{}{}
	}

	return savedQueries
}

func SaveBook(ctx context.Context, db *sql.DB, book isbndb.Book, savedData SavedData) {
	book.Isbn13 = fmt.Sprintf("%.*s", 500, strings.TrimSpace(book.Isbn13))
	book.Synopsis = fmt.Sprintf("%.*s", 10000, book.Synopsis)

	if savedData.IsBookSaved(book.Isbn13) {
		return
	}

	savedData.AddBookToMemory(book.Isbn13) // add it early in case it comes up in another concurrent search

	publisher := fmt.Sprintf("%.*s", 500, strings.TrimSpace(book.Publisher))
	publisherId := savedData.SavePublisher(ctx, db, publisher)

	language := fmt.Sprintf("%.*s", 500, strings.TrimSpace(book.Language))
	languageId := savedData.SaveLanguage(ctx, db, language)

	bookId := insertBook(ctx, db, book, publisherId, languageId)

	for _, author := range book.Authors {
		author := fmt.Sprintf("%.*s", 500, strings.TrimSpace(author))
		authorId := savedData.SaveAuthor(ctx, db, author)

		_, err := db.ExecContext(ctx, `INSERT INTO author_book (author_id, book_id) VALUES (?, ?)`, authorId, bookId)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, subject := range book.Subjects {
		subject := fmt.Sprintf("%.*s", 500, strings.TrimSpace(subject))
		subjectId := savedData.SaveSubject(ctx, db, subject)

		_, err := db.ExecContext(ctx, `INSERT INTO book_subject (book_id, subject_id) VALUES (?, ?)`, bookId, subjectId)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func SaveVolume(ctx context.Context, db *sql.DB, volume google.Volume, savedData SavedData) {
	if savedData.IsBookSaved(volume.Id) {
		return
	}

	savedData.AddBookToMemory(volume.Id) // add it early in case it comes up in another concurrent search

	publisher := fmt.Sprintf("%.*s", 500, strings.TrimSpace(volume.VolumeInfo.Publisher))
	publisherId := savedData.SavePublisher(ctx, db, publisher)

	language := fmt.Sprintf("%.*s", 500, strings.TrimSpace(volume.VolumeInfo.Language))
	languageId := savedData.SaveLanguage(ctx, db, language)

	bookId := insertVolume(ctx, db, volume, publisherId, languageId)

	for _, author := range volume.VolumeInfo.Authors {
		author := fmt.Sprintf("%.*s", 500, strings.TrimSpace(author))
		authorId := savedData.SaveAuthor(ctx, db, author)

		_, err := db.ExecContext(ctx, `INSERT INTO author_book (author_id, book_id) VALUES (?, ?)`, authorId, bookId)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, subject := range volume.VolumeInfo.Categories {
		subject := fmt.Sprintf("%.*s", 500, strings.TrimSpace(subject))
		subjectId := savedData.SaveSubject(ctx, db, subject)

		_, err := db.ExecContext(ctx, `INSERT INTO book_subject (book_id, subject_id) VALUES (?, ?)`, bookId, subjectId)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, industryIdentifier := range volume.VolumeInfo.IndustryIdentifiers {
		_, err := db.ExecContext(ctx, `INSERT INTO industry_identifiers (type, identifier, book_id) VALUES (?, ?, ?)`, industryIdentifier.Type, industryIdentifier.Identifier, bookId)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func insertData(ctx context.Context, db *sql.DB, tableName string, name string) int {
	validateTableNames := []string{"authors", "subjects", "publishers", "languages"}
	if !slices.Contains(validateTableNames, tableName) {
		log.Fatal("Invalid table name")
	}

	result, err := db.ExecContext(ctx, `INSERT INTO `+tableName+` (name) VALUES (?)`, name)
	if err != nil {
		log.Fatal(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}

	return int(id)
}

func insertBook(ctx context.Context, db *sql.DB, book isbndb.Book, publisherId int, languageId int) int {
	result, err := db.ExecContext(ctx, `INSERT INTO books
		(
			title, 
			title_long,
			isbn,
			isbn13,
			dewey_decimal,
			binding,
			publisher_id,
			language_id,
			date_published,
			edition,
			pages,
			dimensions,
			overview,
			image,
			msrp,
			excerpt,
			synopsis,
			related_type
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)`,
		book.Title,
		book.TitleLong,
		book.Isbn,
		book.Isbn13,
		book.DeweyDecimal,
		book.Binding,
		publisherId,
		languageId,
		book.DatePublished,
		book.Edition,
		book.Pages,
		book.Dimensions,
		book.Overview,
		book.Image,
		book.Msrp,
		book.Excerpt,
		book.Synopsis,
		book.Related.Type,
	)

	if err != nil {
		log.Fatal(err)
	}

	bookId, err := result.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}

	return int(bookId)
}

func insertVolume(ctx context.Context, db *sql.DB, volume google.Volume, publisherId int, languageId int) int {
	var isbn10, isbn13, dimensions *string

	for _, volumeInfo := range volume.VolumeInfo.IndustryIdentifiers {
		if volumeInfo.Type == "ISBN_10" {
			isbn10 = &volumeInfo.Identifier
		} else if volumeInfo.Type == "ISBN_13" {
			isbn13 = &volumeInfo.Identifier
		}
	}

	if volume.VolumeInfo.Dimensions.Height > 0 || volume.VolumeInfo.Dimensions.Width > 0 || volume.VolumeInfo.Dimensions.Thickness > 0 {
		dimensionsString := fmt.Sprintf("%v x %v x %v", volume.VolumeInfo.Dimensions.Height, volume.VolumeInfo.Dimensions.Width, volume.VolumeInfo.Dimensions.Thickness)
		dimensions = &dimensionsString
	}

	result, err := db.ExecContext(ctx, `INSERT INTO books
		(
		 	google_id,
			title, 
			subtitle,
			publisher_id,
			date_published,
			synopsis,
			isbn,
			isbn13,
		 	pages,
			average_rating,
			rating_count,
		 	language_id,
			main_category,
			dimensions
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)`,
		volume.Id,
		volume.VolumeInfo.Title,
		volume.VolumeInfo.Subtitle,
		publisherId,
		volume.VolumeInfo.PublishedDate,
		fmt.Sprintf("%.*s", 10000, volume.VolumeInfo.Description),
		isbn10,
		isbn13,
		volume.VolumeInfo.PageCount,
		volume.VolumeInfo.AverageRating,
		volume.VolumeInfo.RatingsCount,
		languageId,
		volume.VolumeInfo.MainCategory,
		dimensions,
	)

	if err != nil {
		log.Fatal(err)
	}

	volumeId, err := result.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}

	return int(volumeId)
}

func insertQuery(ctx context.Context, db *sql.DB, query string) {
	_, err := db.ExecContext(ctx, `INSERT INTO searched_queries (query) VALUES (?)`, query)
	if err != nil {
		log.Fatal(err)
	}
}
