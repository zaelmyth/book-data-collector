package isbndb

import (
	"encoding/json"
	"fmt"
)

/* Types returned by the client */

type Author struct {
	Author string
	Books  []Book
}

type AuthorQueryResults struct {
	Total   int
	Authors []string
}

type Book struct {
	Title                string
	TitleLong            string `json:"title_long"`
	Isbn                 string
	Isbn13               string
	DeweyDecimal         []string `json:"dewey_decimal"`
	Binding              string
	Publisher            string
	Language             string
	DatePublished        string `json:"date_published"`
	Edition              Edition
	Pages                int
	Dimensions           Dimensions
	DimensionsStructured struct {
		Length Measurement
		Width  Measurement
		Height Measurement
		Weight Measurement
	} `json:"dimensions_structured"`
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
	} `json:"other_isbns"`
}

type BookSearchByIsbnResults struct {
	Total     int
	Requested int
	Data      []Book
}

type BookSearchByQueryResults struct {
	Total int
	Books []Book
}

type Publisher struct {
	Publisher string
	Books     []Book
}

type PublisherQueryResults struct {
	Total      int
	Publishers []string
}

type ResponseStatusCode int

type SearchResultsNames struct {
	Total int
	Data  []string
}

type SearchResultsBooks struct {
	Total int
	Data  []Book
}

type Stats struct {
	Books      int
	Authors    int
	Publishers int
	Subjects   int
}

type Subject struct {
	Subject string
	Books   []Book
}

type SubjectQueryResults struct {
	Total    int
	Subjects []string
}

/* Types used by the types above */

type Edition string

// UnmarshalJSON is overridden because the api response for edition can be a string or a number so we have to convert it
func (f *Edition) UnmarshalJSON(data []byte) error {
	var edition interface{}

	err := json.Unmarshal(data, &edition)
	if err != nil {
		return err
	}

	*f = Edition(fmt.Sprint(edition))

	return nil
}

type Dimensions string

// UnmarshalJSON is overridden because the api response for dimensions can be a string or an array so we have to convert it
func (f *Dimensions) UnmarshalJSON(data []byte) error {
	var dimensions interface{}

	err := json.Unmarshal(data, &dimensions)
	if err != nil {
		return err
	}

	*f = Dimensions(fmt.Sprint(dimensions))

	return nil
}

type Msrp string

// UnmarshalJSON is overridden because the api response for msrp can be a string or a number so we have to convert it
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

/* Request types */

type BookSearchByQueryRequest struct {
	Query          string
	Page           int
	PageSize       int
	Column         string
	Year           int
	Edition        int
	Language       string
	ShouldMatchAll bool
}

// SearchRequest contains all parameters from the api documentation but from testing it looks like only index, page,
// pageSize and text are used
type SearchRequest struct {
	Index     string
	Page      int
	PageSize  int
	Isbn      string
	Isbn13    string
	Author    string
	Text      string
	Subject   string
	Publisher string
}
