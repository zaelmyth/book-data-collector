package db

import (
	"context"
	"database/sql"
	"log"

	"github.com/zaelmyth/book-data-collector/internal/configuration"
)

func CreateDatabases(ctx context.Context, config configuration.Config) {
	db := getDatabase(config)
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(db)

	_, err := db.ExecContext(ctx, `CREATE DATABASE IF NOT EXISTS `+config.DbNameBooks+` DEFAULT CHARACTER SET = 'utf8mb4' DEFAULT COLLATE 'utf8mb4_bin';`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE DATABASE IF NOT EXISTS `+config.DbNameProgress+` DEFAULT CHARACTER SET = 'utf8mb4' DEFAULT COLLATE 'utf8mb4_bin';`)
	if err != nil {
		log.Fatal(err)
	}
}

func GetBooksDatabase(config configuration.Config) *sql.DB {
	mysqlConnectionString := getMysqlConnectionString(config)
	// the database has to be declared in the connection instead of with a "USE" statement because of concurrency issues
	db, err := sql.Open("mysql", mysqlConnectionString+config.DbNameBooks+"?charset=utf8mb4")
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func GetProgressDatabase(config configuration.Config) *sql.DB {
	mysqlConnectionString := getMysqlConnectionString(config)
	// the database has to be declared in the connection instead of with a "USE" statement because of concurrency issues
	db, err := sql.Open("mysql", mysqlConnectionString+config.DbNameProgress+"?charset=utf8mb4")
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func CreateProgressTables(ctx context.Context, progressDb *sql.DB) {
	_, err := progressDb.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS searched_queries (query TEXT);`)
	if err != nil {
		log.Fatal(err)
	}
}

func CreateBookTables(ctx context.Context, db *sql.DB) {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTO_INCREMENT,
		title TEXT,
		title_long TEXT,
		isbn TEXT NULL,
		isbn13 VARCHAR(500) NULL,
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
		related_type TEXT,
		google_id TEXT,
		subtitle TEXT,
		average_rating FLOAT,
		rating_count INTEGER,
		main_category TEXT
# 		UNIQUE (isbn13)
	);`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS publishers (id INTEGER PRIMARY KEY AUTO_INCREMENT, name VARCHAR(500), UNIQUE (name));`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS languages (id INTEGER PRIMARY KEY AUTO_INCREMENT, name VARCHAR(500), UNIQUE (name));`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS authors (id INTEGER PRIMARY KEY AUTO_INCREMENT, name VARCHAR(500), UNIQUE (name));`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS author_book (author_id INTEGER, book_id INTEGER);`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS subjects (id INTEGER PRIMARY KEY AUTO_INCREMENT, name VARCHAR(500), UNIQUE (name));`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS book_subject (book_id INTEGER, subject_id INTEGER);`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS industry_identifiers (id INTEGER PRIMARY KEY AUTO_INCREMENT, type TEXT, identifier TEXT, book_id INTEGER);`)
	if err != nil {
		log.Fatal(err)
	}
}

func getDatabase(config configuration.Config) *sql.DB {
	mysqlConnectionString := getMysqlConnectionString(config)
	db, err := sql.Open("mysql", mysqlConnectionString)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func getMysqlConnectionString(config configuration.Config) string {
	return config.DbUsername + ":" + config.DbPassword + "@tcp(" + config.DbHost + ":" + config.DbPort + ")/"
}
