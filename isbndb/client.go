package isbndb

import (
	"fmt"
	"log"
	"net/url"
	"os"
)

const isbndbMaxPageSize = 1000

func AuthorDetails(name string, page uint, pageSize uint) Author {
	if pageSize > isbndbMaxPageSize {
		log.Fatal("Page size cannot be bigger than 1000")
	}

	response := call("get", "/author/"+name, url.Values{
		"page":     {fmt.Sprint(page)},
		"pageSize": {fmt.Sprint(pageSize)},
	})

	return toStruct(response, Author{})
}

func SearchAuthors(query string, page uint, pageSize uint) AuthorQueryResults {
	if pageSize > isbndbMaxPageSize {
		log.Fatal("Page size cannot be bigger than 1000")
	}

	response := call("get", "/authors/"+query, url.Values{
		"page":     {fmt.Sprint(page)},
		"pageSize": {fmt.Sprint(pageSize)},
	})

	return toStruct(response, AuthorQueryResults{})
}

func BookDetails(isbn string, withPrices bool) Book {
	if withPrices && os.Getenv("ISBNDB_SUBSCRIPTION_TYPE") != "pro" {
		log.Fatal("Book details with prices option is only available with the pro subscription")
	}

	withPricesQuery := "0"
	if withPrices {
		withPricesQuery = "1"
	}

	response := call("get", "/book/"+isbn, url.Values{
		"with_prices": {withPricesQuery},
	})

	toStruct := toStruct(response, struct {
		Book Book
	}{})

	return toStruct.Book
}

func SearchBooksByIsbn(isbns []string) BookSearchResultsByIsbn {
	if len(isbns) > isbndbMaxPageSize {
		log.Fatal("Page size cannot be bigger than 1000")
	}

	response := call("post", "/books", url.Values{
		"isbns": isbns,
	})

	return toStruct(response, BookSearchResultsByIsbn{})
}

func GetStats() Stats {
	response := call("get", "/stats", url.Values{})

	return toStruct(response, Stats{})
}
