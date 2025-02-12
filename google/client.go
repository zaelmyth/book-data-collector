package google

import (
	"log"
	"net/url"
	"slices"
	"strconv"

	client "github.com/zaelmyth/book-data-collector/internal"
)

const apiUrl = "https://www.googleapis.com/books/v1"

// Search There are special keywords you can specify in the search terms to search in particular fields:
// https://developers.google.com/books/docs/v1/using#PerformingSearch
func Search(query string, parameters SearchParameters) (SearchResults, int) {
	if query == "" {
		log.Fatal("Empty query")
	}

	validateSearchParameters(parameters)

	if parameters.MaxResults == 0 {
		parameters.MaxResults = 40
	}

	requestData := url.Values{
		"q":          {query},
		"startIndex": {strconv.Itoa(parameters.StartIndex)},
		"maxResults": {strconv.Itoa(parameters.MaxResults)},
	}

	if parameters.Download != "" {
		requestData.Add("download", parameters.Download)
	}

	if parameters.Filter != "" {
		requestData.Add("filter", parameters.Filter)
	}

	if parameters.PrintType != "" {
		requestData.Add("printType", parameters.PrintType)
	}

	if parameters.Projection != "" {
		requestData.Add("projection", parameters.Projection)
	}

	if parameters.OrderBy != "" {
		requestData.Add("orderBy", parameters.OrderBy)
	}

	if parameters.Language != "" {
		requestData.Add("langRestrict", parameters.Language)
	}

	return call("get", "/volumes", requestData, SearchResults{})
}

func VolumeDetails(id string) (Volume, int) {
	return call("get", "/volumes/"+id, url.Values{}, Volume{})
}

func call[T any](method string, url string, data url.Values, responseStruct T) (T, int) {
	return client.Call(method, apiUrl+url, data, map[string]string{}, responseStruct)
}

func validateSearchParameters(parameters SearchParameters) {
	if parameters.Download != "" && parameters.Download != "epub" {
		log.Fatal("Invalid download parameter")
	}

	validFilterValues := []string{"", "partial", "full", "free-ebooks", "paid-ebooks", "ebooks"}
	if !slices.Contains(validFilterValues, parameters.Filter) {
		log.Fatal("Invalid filter parameter")
	}

	if parameters.StartIndex < 0 {
		log.Fatal("Invalid start index parameter")
	}

	if parameters.MaxResults > 40 {
		log.Fatal("Invalid max results parameter")
	}

	validPrintTypeValues := []string{"", "all", "books", "magazines"}
	if !slices.Contains(validPrintTypeValues, parameters.PrintType) {
		log.Fatal("Invalid print type parameter")
	}

	validProjectionValues := []string{"", "full", "lite"}
	if !slices.Contains(validProjectionValues, parameters.Projection) {
		log.Fatal("Invalid projection parameter")
	}

	validOrderByValues := []string{"", "relevance", "newest"}
	if !slices.Contains(validOrderByValues, parameters.OrderBy) {
		log.Fatal("Invalid order by parameter")
	}
}
