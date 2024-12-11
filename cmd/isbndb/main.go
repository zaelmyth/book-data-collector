package main

import (
	"fmt"

	_ "github.com/joho/godotenv/autoload"
	"github.com/zaelmyth/book-data-collector/isbndb"
)

func main() {
	//response := isbndb.SearchBooksByIsbn([]string{"9780542406614", "9781566199094"})
	response := isbndb.SearchSubjects("test", 1, 10)

	fmt.Printf("response: %v\n", response)
}
