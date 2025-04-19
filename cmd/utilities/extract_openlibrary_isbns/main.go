package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile) // add code file name and line number to error messages

	inputFileFlag := flag.String("input", "", "Open Library dump file location")
	outputFileFlag := flag.String("output", "", "Output file location")
	flag.Parse()

	if *inputFileFlag == "" {
		log.Fatal("Open Library dump file location is not set")
	}

	if *outputFileFlag == "" {
		log.Fatal("Output file location is not set")
	}

	fmt.Println("Extracting ISBNs from Open Library dump file...")

	file, err := os.Open(*inputFileFlag)
	if err != nil {
		log.Fatal(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(file)

	gzipFile, err := gzip.NewReader(file)
	if err != nil {
		log.Fatal(err)
	}
	defer func(fz *gzip.Reader) {
		err := fz.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(gzipFile)

	outputFile, err := os.Create(*outputFileFlag)
	if err != nil {
		log.Fatal(err)
	}
	defer func(outputFile *os.File) {
		err := outputFile.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(outputFile)

	scanner := bufio.NewScanner(gzipFile)
	const maxLineCharacters int = 10000000
	buffer := make([]byte, maxLineCharacters)
	scanner.Buffer(buffer, maxLineCharacters)
	for scanner.Scan() {
		olLine := strings.Split(scanner.Text(), "\t")

		type OpenLibraryEdition struct {
			Isbn13 []string `json:"isbn_13"`
		}
		var openLibraryEdition OpenLibraryEdition
		err = json.Unmarshal([]byte(olLine[4]), &openLibraryEdition)
		if err != nil {
			log.Fatal(err)
		}

		for _, isbn13 := range openLibraryEdition.Isbn13 {
			_, err := outputFile.WriteString(isbn13 + "\n")
			if err != nil {
				return
			}
		}
	}

	err = scanner.Err()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Done!")
}
