package client

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const apiTimeoutSeconds = 120

func Call[T any](method string, url string, data url.Values, headers map[string]string, responseStruct T) (T, int) {
	httpClient := http.Client{
		Timeout: apiTimeoutSeconds * time.Second,
	}

	var request *http.Request
	if method == "post" {
		request = createPostRequest(data, url, headers)
	} else {
		request = createGetRequest(url, data, headers)
	}

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

	validStatusCodes := []int{http.StatusOK, http.StatusNotFound, http.StatusGatewayTimeout, http.StatusTooManyRequests}
	if !slices.Contains(validStatusCodes, response.StatusCode) {
		log.Fatal(response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	jsonError := json.Unmarshal(body, &responseStruct)
	if jsonError != nil {
		log.Fatal(jsonError)
	}

	return responseStruct, response.StatusCode
}

func createGetRequest(url string, data url.Values, headers map[string]string) *http.Request {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Add(key, value)
	}

	request.URL.RawQuery = data.Encode()

	return request
}

func createPostRequest(data url.Values, url string, headers map[string]string) *http.Request {
	var bodyString []string
	for key, values := range data {
		bodyString = append(bodyString, key+"="+strings.Join(values, ","))
	}

	body := []byte(strings.Join(bodyString, "&"))
	bodyBuffer := bytes.NewBuffer(body)

	request, err := http.NewRequest(http.MethodPost, url, bodyBuffer)
	if err != nil {
		log.Fatal(err)
	}

	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Add(key, value)
	}

	return request
}
