package main

import (
	"fmt"

	"github.com/zaelmyth/book-data-collector/isbndb"
)

func main() {
	statsResponse := isbndb.Stats()

	fmt.Printf("statsResponse: %v\n", statsResponse)
}
