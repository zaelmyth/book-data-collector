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

	return call("get", "/author/"+name, url.Values{
		"page":     {fmt.Sprint(page)},
		"pageSize": {fmt.Sprint(pageSize)},
	}, Author{})
}

func SearchAuthors(query string, page uint, pageSize uint) AuthorQueryResults {
	if pageSize > isbndbMaxPageSize {
		log.Fatal("Page size cannot be bigger than 1000")
	}

	return call("get", "/authors/"+query, url.Values{
		"page":     {fmt.Sprint(page)},
		"pageSize": {fmt.Sprint(pageSize)},
	}, AuthorQueryResults{})
}

func BookDetails(isbn string, withPrices bool) Book {
	if withPrices && os.Getenv("ISBNDB_SUBSCRIPTION_TYPE") != "pro" {
		log.Fatal("Book details with prices option is only available with the pro subscription")
	}

	withPricesQuery := "0"
	if withPrices {
		withPricesQuery = "1"
	}

	return call("get", "/book/"+isbn, url.Values{
		"with_prices": {withPricesQuery},
	}, struct {
		Book Book
	}{}).Book
}

func SearchBooksByIsbn(isbns []string) BookSearchResultsByIsbn {
	if len(isbns) > isbndbMaxPageSize {
		log.Fatal("Page size cannot be bigger than 1000")
	}

	return call("post", "/books", url.Values{
		"isbns": isbns,
	}, BookSearchResultsByIsbn{})
}

func GetStats() Stats {
	return call("get", "/stats", url.Values{}, Stats{})
}
