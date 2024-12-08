package main

import (
	"fmt"
	"github.com/zaelmyth/book-data-collector/isbndb"
)

func main() {
	response := isbndb.SearchBooksByIsbn([]string{"9780542406614", "9781566199094"})

	fmt.Printf("response: %v\n", response)
}
