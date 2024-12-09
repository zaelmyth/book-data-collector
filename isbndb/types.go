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

type BookSearchResultsByIsbn struct {
	Total     int
	Requested int
	Data      []Book
}

type BookSearchResultsByQuery struct {
	Total int
	Books []Book
}

type Publisher struct {
	Name  string
	Books []struct {
		isbn string
	}
}

type Stats struct {
	Books      int
	Authors    int
	Publishers int
	Subjects   int
}

type Subject struct {
	Subject string
	Parent  string
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
