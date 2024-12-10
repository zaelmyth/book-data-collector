package isbndb

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"
)

const apiTimeout = 30
const apiUrlBasic = "https://api2.isbndb.com"
const apiUrlPremium = "https://api.premium.isbndb.com"
const apiUrlPro = "https://api.pro.isbndb.com"

func call(method string, url string, query url.Values) []byte {
	httpClient := http.Client{
		Timeout: apiTimeout * time.Second,
	}
	request := createRequest(method, url, query)

	response, err := httpClient.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(response.Body)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	return body
}

func toStruct[T interface{}](response []byte, responseStruct T) T {
	jsonError := json.Unmarshal(response, &responseStruct)
	if jsonError != nil {
		log.Fatal(jsonError)
	}

	return responseStruct
}

func createRequest(method string, url string, query url.Values) *http.Request {
	apiUrl := getApiUrl()

	if method == "post" {
		bodyBuffer := getBodyBuffer(query)
		request, err := http.NewRequest(http.MethodPost, apiUrl+url, bodyBuffer)
		if err != nil {
			log.Fatal(err)
		}

		request.Header.Add("Authorization", os.Getenv("ISBNDB_API_KEY"))
		request.Header.Add("Accept", "application/json")
		request.Header.Add("Content-Type", "application/json")

		return request
	}

	request, err := http.NewRequest(http.MethodGet, apiUrl+url, nil)
	if err != nil {
		log.Fatal(err)
	}

	request.Header.Add("Authorization", os.Getenv("ISBNDB_API_KEY"))
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")
	request.URL.RawQuery = query.Encode()

	return request
}

func getApiUrl() string {
	validSubscriptionTypes := []string{"basic", "premium", "pro"}
	subscriptionType := os.Getenv("ISBNDB_SUBSCRIPTION_TYPE")

	if !slices.Contains(validSubscriptionTypes, subscriptionType) {
		log.Fatal("Unknown subscription type")
	}

	if subscriptionType == "basic" {
		return apiUrlBasic
	}

	if subscriptionType == "premium" {
		return apiUrlPremium
	}

	return apiUrlPro
}

func getBodyBuffer(query url.Values) *bytes.Buffer {
	var bodyString []string
	for key, values := range query {
		bodyString = append(bodyString, key+"="+strings.Join(values, ","))
	}

	body := []byte(strings.Join(bodyString, "&"))

	return bytes.NewBuffer(body)
}
