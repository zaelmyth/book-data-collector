package main

import (
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

func (savedData *savedData) addAuthor(id int, name string) {
	savedData.authorsMutex.Lock()
	defer savedData.authorsMutex.Unlock()

	savedData.authors[name] = id
}

func (savedData *savedData) getAuthorId(name string) (int, bool) {
	savedData.authorsMutex.RLock()
	defer savedData.authorsMutex.RUnlock()

	id, isSaved := savedData.authors[name]

	return id, isSaved
}

func (savedData *savedData) addSubject(id int, name string) {
	savedData.subjectsMutex.Lock()
	defer savedData.subjectsMutex.Unlock()

	savedData.subjects[name] = id
}

func (savedData *savedData) getSubjectId(name string) (int, bool) {
	savedData.subjectsMutex.RLock()
	defer savedData.subjectsMutex.RUnlock()

	id, isSaved := savedData.subjects[name]

	return id, isSaved
}

func (savedData *savedData) addPublisher(id int, name string) {
	savedData.publishersMutex.Lock()
	defer savedData.publishersMutex.Unlock()

	savedData.publishers[name] = id
}

func (savedData *savedData) getPublisherId(name string) (int, bool) {
	savedData.publishersMutex.RLock()
	defer savedData.publishersMutex.RUnlock()

	id, isSaved := savedData.publishers[name]

	return id, isSaved
}

func (savedData *savedData) addLanguage(id int, name string) {
	savedData.languagesMutex.Lock()
	defer savedData.languagesMutex.Unlock()

	savedData.languages[name] = id
}

func (savedData *savedData) getLanguageId(name string) (int, bool) {
	savedData.languagesMutex.RLock()
	defer savedData.languagesMutex.RUnlock()

	id, isSaved := savedData.languages[name]

	return id, isSaved
}
