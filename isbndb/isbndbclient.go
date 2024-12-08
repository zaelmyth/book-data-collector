package isbndb

import (
	"encoding/json"
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
	response := call("post", "/books", url.Values{
		"isbns": isbns,
	})

	return toStruct(response, BookSearchResultsByIsbn{})
}

func GetStats() Stats {
	response := call("get", "/stats", url.Values{})

	return toStruct(response, Stats{})
}

type AuthorQueryResults struct {
	Total   int
	Authors []string
}

type BookSearchResultsByIsbn struct {
	Total     int
	Requested int
	Data      []Book
}

type BookSearchResultsByQuery struct {
	Total int
	Books []Book
}

type Book struct {
	Title                string
	TitleLong            string
	Isbn                 string
	Isbn13               string
	DeweyDecimal         string
	Binding              string
	Publisher            string
	Language             string
	DatePublished        string
	Edition              Edition
	Pages                int
	Dimensions           string
	DimensionsStructured struct {
		Length Measurement
		Width  Measurement
		Height Measurement
		Weight Measurement
	}
	Overview string
	Image    string
	Msrp     Msrp
	Excerpt  string
	Synopsis string
	Authors  []string
	Subjects []string
	Reviews  []string
	Prices   []Merchant
	Related  struct {
		Type string
	}
	OtherIsbns []struct {
		Isbn    string
		Binding string
	}
}

type Author struct {
	Author string
	Books  []Book
}

type Publisher struct {
	Name  string
	Books []struct {
		isbn string
	}
}

type Subject struct {
	Subject string
	Parent  string
}

type Merchant struct {
	Condition          string
	Merchant           string
	MerchantLogo       string
	MerchantLogoOffset struct {
		X string
		Y string
	}
	Shipping string
	Price    string
	Total    string
	Link     string
}

type Stats struct {
	Books      int
	Authors    int
	Publishers int
	Subjects   int
}

type Edition string

// UnmarshalJSON the edition api response can be string or number so we have to convert it
func (f *Edition) UnmarshalJSON(data []byte) error {
	var edition interface{}

	err := json.Unmarshal(data, &edition)
	if err != nil {
		return err
	}

	*f = Edition(fmt.Sprint(edition))

	return nil
}

type Msrp string

// UnmarshalJSON the msrp api response can be string or number so we have to convert it
func (f *Msrp) UnmarshalJSON(data []byte) error {
	var msrp interface{}

	err := json.Unmarshal(data, &msrp)
	if err != nil {
		return err
	}

	*f = Msrp(fmt.Sprint(msrp))

	return nil
}

type Measurement struct {
	Unit  string
	Value float64
}
