package isbndb

import (
	"log"
	"net/url"
	"os"
	"slices"
	"strconv"
)

// the api documentation says that the max page size is 1000, but it actually is limited by the response size
// 2000 results per page is safely within the limit
const isbndbMaxPageSize = 2000 // todo: too high for basic subscription
const isbndbMaxReturnSize = 10000

func AuthorDetails(name string, page int, pageSize int, language string) Author {
	validatePagination(page, pageSize)

	return call("get", "/author/"+name, url.Values{
		"page":     {strconv.Itoa(page)},
		"pageSize": {strconv.Itoa(pageSize)},
		"language": {language},
	}, Author{})
}

func SearchAuthors(query string, page int, pageSize int) AuthorQueryResults {
	validatePagination(page, pageSize)

	return call("get", "/authors/"+query, url.Values{
		"page":     {strconv.Itoa(page)},
		"pageSize": {strconv.Itoa(pageSize)},
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

func SearchBooksByIsbn(isbns []string) BookSearchByIsbnResults {
	if len(isbns) > isbndbMaxPageSize {
		log.Fatal("Number of ISBNs cannot be bigger than 1000")
	}

	return call("post", "/books", url.Values{
		"isbns": isbns,
	}, BookSearchByIsbnResults{})
}

func SearchBooksByQuery(request BookSearchByQueryRequest) BookSearchByQueryResults {
	validatePagination(request.Page, request.PageSize)
	if !slices.Contains([]string{"", "title", "author", "date_published", "subjects"}, request.Column) {
		log.Fatal("Invalid column")
	}

	shouldMatchAll := "0"
	if request.ShouldMatchAll {
		shouldMatchAll = "1"
	}

	data := url.Values{
		"page":           {strconv.Itoa(request.Page)},
		"pageSize":       {strconv.Itoa(request.PageSize)},
		"column":         {request.Column},
		"language":       {request.Language},
		"shouldMatchAll": {shouldMatchAll},
	}

	if request.Year != 0 {
		data.Add("year", strconv.Itoa(request.Year))
	}

	if request.Edition != 0 {
		data.Add("edition", strconv.Itoa(request.Edition))
	}

	return call("get", "/books/"+request.Query, data, BookSearchByQueryResults{})
}

func PublisherDetails(name string, page int, pageSize int, language string) Publisher {
	validatePagination(page, pageSize)

	return call("get", "/publisher/"+name, url.Values{
		"page":     {strconv.Itoa(page)},
		"pageSize": {strconv.Itoa(pageSize)},
		"language": {language},
	}, Publisher{})
}

func SearchPublishers(query string, page int, pageSize int) PublisherQueryResults {
	validatePagination(page, pageSize)

	return call("get", "/publishers/"+query, url.Values{
		"page":     {strconv.Itoa(page)},
		"pageSize": {strconv.Itoa(pageSize)},
	}, PublisherQueryResults{})
}

func SearchByIndex(request SearchRequest) SearchResultsNames {
	validatePagination(request.Page, request.PageSize)
	if !slices.Contains([]string{"authors", "subjects", "publishers"}, request.Index) {
		log.Fatal("Invalid index")
	}

	data := url.Values{
		"page":     {strconv.Itoa(request.Page)},
		"pageSize": {strconv.Itoa(request.PageSize)},
		"text":     {request.Text},
	}

	if request.Isbn != "" {
		data.Add("isbn", request.Isbn)
	}

	if request.Isbn13 != "" {
		data.Add("isbn13", request.Isbn13)
	}

	if request.Author != "" {
		data.Add("author", request.Author)
	}

	if request.Subject != "" {
		data.Add("subject", request.Subject)
	}

	if request.Publisher != "" {
		data.Add("publisher", request.Publisher)
	}

	return call("get", "/search/"+request.Index, data, SearchResultsNames{})
}

func SearchByIndexBooks(request SearchRequest) SearchResultsBooks {
	validatePagination(request.Page, request.PageSize)
	if request.Index != "books" {
		log.Fatal("Invalid index")
	}

	data := url.Values{
		"page":     {strconv.Itoa(request.Page)},
		"pageSize": {strconv.Itoa(request.PageSize)},
		"text":     {request.Text},
	}

	if request.Isbn != "" {
		data.Add("isbn", request.Isbn)
	}

	if request.Isbn13 != "" {
		data.Add("isbn13", request.Isbn13)
	}

	if request.Author != "" {
		data.Add("author", request.Author)
	}

	if request.Subject != "" {
		data.Add("subject", request.Subject)
	}

	if request.Publisher != "" {
		data.Add("publisher", request.Publisher)
	}

	return call("get", "/search/"+request.Index, data, SearchResultsBooks{})
}

func GetStats() Stats {
	return call("get", "/stats", url.Values{}, Stats{})
}

func SubjectDetails(name string) Subject {
	return call("get", "/subject/"+name, url.Values{}, Subject{})
}

func SearchSubjects(query string, page int, pageSize int) SubjectQueryResults {
	validatePagination(page, pageSize)

	return call("get", "/subjects/"+query, url.Values{
		"page":     {strconv.Itoa(page)},
		"pageSize": {strconv.Itoa(pageSize)},
	}, SubjectQueryResults{})
}

func validatePagination(page int, pageSize int) {
	if page < 1 {
		log.Fatal("Page cannot be less than 1")
	}

	if pageSize < 1 {
		log.Fatal("Page size cannot be less than 1")
	}

	if pageSize > isbndbMaxPageSize {
		log.Fatal("Page size cannot be bigger than 1000")
	}

	if page*pageSize > isbndbMaxReturnSize {
		log.Fatal("Total number of results cannot be bigger than 10000")
	}
}
