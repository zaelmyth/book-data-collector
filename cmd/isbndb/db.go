package main

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/zaelmyth/book-data-collector/isbndb"
)

func createTables(ctx context.Context, db *sql.DB, progressDb *sql.DB) {
	_, err := progressDb.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS searched_words (word TEXT);`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		title_long TEXT,
		isbn TEXT,
		isbn13 TEXT,
		dewey_decimal TEXT,
		binding TEXT,
		publisher_id INTEGER,
		language_id INTEGER,
		date_published TEXT,
		edition TEXT,
		pages INTEGER,
		dimensions TEXT,
		overview TEXT,
		image TEXT,
		msrp TEXT,
		excerpt TEXT,
		synopsis TEXT,
		related_type TEXT
	);`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS publishers (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT);`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS languages (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT);`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS authors (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT);`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS author_book (author_id INTEGER, book_id INTEGER);`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS subjects (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT);`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS book_subject (book_id INTEGER, subject_id INTEGER);`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS reviews (id INTEGER PRIMARY KEY AUTOINCREMENT, text TEXT, book_id INTEGER);`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS other_isbns (id INTEGER PRIMARY KEY AUTOINCREMENT, isbn TEXT, binding TEXT, book_id INTEGER);`)
	if err != nil {
		log.Fatal(err)
	}
}

func saveBooks(ctx context.Context, db *sql.DB, books []isbndb.Book) {
	for _, book := range books {
		saveBook(ctx, db, book)
	}
}

func saveBook(ctx context.Context, db *sql.DB, book isbndb.Book) {
	publisherId := getIdOrInsert(ctx, db, "publishers", book.Publisher)
	languageId := getIdOrInsert(ctx, db, "languages", book.Language)

	var bookId int
	err := db.QueryRowContext(ctx, `SELECT id FROM books WHERE isbn13 = ?`, book.Isbn13).Scan(&bookId)
	if bookId != 0 {
		return
	}

	if errors.Is(err, sql.ErrNoRows) {
		bookId = insertBook(ctx, db, book, publisherId, languageId)
	} else if err != nil {
		log.Fatal(err)
	}

	for _, author := range book.Authors {
		authorId := getIdOrInsert(ctx, db, "authors", author)
		_, err := db.ExecContext(ctx, `INSERT INTO author_book (author_id, book_id) VALUES (?, ?)`, authorId, bookId)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, subject := range book.Subjects {
		subjectId := getIdOrInsert(ctx, db, "subjects", subject)
		_, err := db.ExecContext(ctx, `INSERT INTO book_subject (book_id, subject_id) VALUES (?, ?)`, bookId, subjectId)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, review := range book.Reviews {
		_, err := db.ExecContext(ctx, `INSERT INTO reviews (text, book_id) VALUES (?, ?)`, review, bookId)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, otherIsbn := range book.OtherIsbns {
		_, err := db.ExecContext(ctx, `INSERT INTO other_isbns (isbn, binding, book_id) VALUES (?, ?, ?)`, otherIsbn.Isbn, otherIsbn.Binding, bookId)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func getIdOrInsert(ctx context.Context, db *sql.DB, tableName string, name string) int {
	var id int64
	err := db.QueryRowContext(ctx, `SELECT id FROM `+tableName+` WHERE name = ?`, name).Scan(&id)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		result, err := db.ExecContext(ctx, `INSERT INTO `+tableName+` (name) VALUES (?)`, name)
		if err != nil {
			log.Fatal(err)
		}

		id, err = result.LastInsertId()
		if err != nil {
			log.Fatal(err)
		}
	case err != nil:
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
