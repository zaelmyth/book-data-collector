package isbndb

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

const API_TIMEOUT = 30
const API_URL_BASIC = "https://api2.isbndb.com"
const API_URL_PREMIUM = "https://api.premium.isbndb.com"
const API_URL_PRO = "https://api.pro.isbndb.com"

func call(url string) []byte {
	httpClient := http.Client{
		Timeout: API_TIMEOUT * time.Second,
	}

	validSubscriptionTypes := []string{"basic", "premium", "pro"}
	subscriptionType := os.Getenv("ISBNDB_SUBSCRIPTION_TYPE")
	if !slices.Contains(validSubscriptionTypes, subscriptionType) {
		log.Fatal("Unknown subscription type")
	}

	apiUrl := API_URL_BASIC
	if subscriptionType == "premium" {
		apiUrl = API_URL_PREMIUM
	}
	if subscriptionType == "pro" {
		apiUrl = API_URL_PRO
	}

	request, error := http.NewRequest(http.MethodGet, apiUrl+url, nil)
	if error != nil {
		log.Fatal(error)
	}

	request.Header.Add("Authorization", os.Getenv("ISBNDB_API_KEY"))
	request.Header.Add("Accept", "application/json")

	response, error := httpClient.Do(request)
	if error != nil {
		log.Fatal(error)
	}
	defer response.Body.Close()

	body, error := io.ReadAll(response.Body)
	if error != nil {
		log.Fatal(error)
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
