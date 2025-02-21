package db

import (
	"context"
	"database/sql"
	"sync"
)

type SavedData struct {
	Books           map[string]struct{}
	BooksMutex      *sync.RWMutex
	Authors         map[string]int
	AuthorsMutex    *sync.Mutex
	Subjects        map[string]int
	SubjectsMutex   *sync.Mutex
	Publishers      map[string]int
	PublishersMutex *sync.Mutex
	Languages       map[string]int
	LanguagesMutex  *sync.Mutex
	Queries         map[string]struct{}
	QueriesMutex    *sync.RWMutex
}

func (savedData *SavedData) IsBookSaved(id string) bool {
	savedData.BooksMutex.RLock()
	defer savedData.BooksMutex.RUnlock()

	_, isSaved := savedData.Books[id]

	return isSaved
}

func (savedData *SavedData) AddBookToMemory(id string) bool {
	savedData.BooksMutex.Lock()
	defer savedData.BooksMutex.Unlock()

	savedData.Books[id] = struct{}{}

	return true
}

func (savedData *SavedData) SaveAuthor(ctx context.Context, db *sql.DB, name string) int {
	savedData.AuthorsMutex.Lock()
	defer savedData.AuthorsMutex.Unlock()

	id, isSaved := savedData.Authors[name]
	if !isSaved {
		id = insertData(ctx, db, "authors", name)
		savedData.Authors[name] = id
	}

	return id
}

func (savedData *SavedData) SaveSubject(ctx context.Context, db *sql.DB, name string) int {
	savedData.SubjectsMutex.Lock()
	defer savedData.SubjectsMutex.Unlock()

	id, isSaved := savedData.Subjects[name]
	if !isSaved {
		id = insertData(ctx, db, "subjects", name)
		savedData.Subjects[name] = id
	}

	return id
}

func (savedData *SavedData) SavePublisher(ctx context.Context, db *sql.DB, name string) int {
	savedData.PublishersMutex.Lock()
	defer savedData.PublishersMutex.Unlock()

	id, isSaved := savedData.Publishers[name]
	if !isSaved {
		id = insertData(ctx, db, "publishers", name)
		savedData.Publishers[name] = id
	}

	return id
}

func (savedData *SavedData) SaveLanguage(ctx context.Context, db *sql.DB, name string) int {
	savedData.LanguagesMutex.Lock()
	defer savedData.LanguagesMutex.Unlock()

	id, isSaved := savedData.Languages[name]
	if !isSaved {
		id = insertData(ctx, db, "languages", name)
		savedData.Languages[name] = id
	}

	return id
}

func (savedData *SavedData) IsQuerySaved(query string) bool {
	savedData.QueriesMutex.RLock()
	defer savedData.QueriesMutex.RUnlock()

	_, isSaved := savedData.Queries[query]

	return isSaved
}

func (savedData *SavedData) SaveQuery(ctx context.Context, db *sql.DB, query string) {
	savedData.QueriesMutex.Lock()
	defer savedData.QueriesMutex.Unlock()

	_, isSaved := savedData.Queries[query]
	if !isSaved {
		insertQuery(ctx, db, query)
		savedData.Queries[query] = struct{}{}
	}
}
