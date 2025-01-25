package main

import (
	"context"
	"database/sql"
	"sync"

	"github.com/zaelmyth/book-data-collector/isbndb"
)

type booksSave struct {
	books            []isbndb.Book
	word             string
	isSearchComplete bool
}

type savedData struct {
	books           map[string]struct{}
	booksMutex      *sync.RWMutex
	authors         map[string]int
	authorsMutex    *sync.RWMutex
	subjects        map[string]int
	subjectsMutex   *sync.RWMutex
	publishers      map[string]int
	publishersMutex *sync.RWMutex
	languages       map[string]int
	languagesMutex  *sync.RWMutex
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

func (savedData *savedData) saveAuthor(ctx context.Context, db *sql.DB, name string) int {
	savedData.authorsMutex.Lock()
	defer savedData.authorsMutex.Unlock()

	id, isSaved := savedData.authors[name]
	if !isSaved {
		id = insert(ctx, db, "authors", name)
		savedData.authors[name] = id
	}

	return id
}

func (savedData *savedData) saveSubject(ctx context.Context, db *sql.DB, name string) int {
	savedData.subjectsMutex.Lock()
	defer savedData.subjectsMutex.Unlock()

	id, isSaved := savedData.subjects[name]
	if !isSaved {
		id = insert(ctx, db, "subjects", name)
		savedData.subjects[name] = id
	}

	return id
}

func (savedData *savedData) savePublisher(ctx context.Context, db *sql.DB, name string) int {
	savedData.publishersMutex.Lock()
	defer savedData.publishersMutex.Unlock()

	id, isSaved := savedData.publishers[name]
	if !isSaved {
		id = insert(ctx, db, "publishers", name)
		savedData.publishers[name] = id
	}

	return id
}

func (savedData *savedData) saveLanguage(ctx context.Context, db *sql.DB, name string) int {
	savedData.languagesMutex.Lock()
	defer savedData.languagesMutex.Unlock()

	id, isSaved := savedData.languages[name]
	if !isSaved {
		id = insert(ctx, db, "languages", name)
		savedData.languages[name] = id
	}

	return id
}
