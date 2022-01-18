package main

import (
	"fmt"
	"io/ioutil"
	"models"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func RunGetRequest(url string) (statusCode int, response string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}

	statusCode = resp.StatusCode
	p, _ := ioutil.ReadAll(resp.Body)
	response = string(p)
	return
}

func RunPostRequest(url string, payload string) (statusCode int, response string, err error) {
	resp, err := http.Post(url, "application/json", strings.NewReader(payload))
	if err != nil {
		return
	}

	statusCode = resp.StatusCode
	p, _ := ioutil.ReadAll(resp.Body)
	response = string(p)
	return
}

func TestTransactions(t *testing.T) {
	//gin.SetMode(gin.ReleaseMode)
	models.ConnectDatabase(":memory:")
	ts := httptest.NewServer(setupServer())

	defer ts.Close()

	tests := [][]string{
		{"GET", fmt.Sprintf("%s/transactions", ts.URL), `{"data":[]}`, "200", ""},
		{"GET", fmt.Sprintf("%s/transactions/1", ts.URL), `{"error":"Record not found!"}`, "400", ``},
		{"GET", fmt.Sprintf("%s/transactions?idx=1", ts.URL), `{"data":null}`, "200", ""},
		{"POST", fmt.Sprintf("%s/transactions", ts.URL), `{"error":"EOF"}`, "400", ``},
		{"POST", fmt.Sprintf("%s/transactions", ts.URL), `{"error":"Key: 'CreateTransactionInput.Account' Error:Field validation for 'Account' failed on the 'required' tag"}`, "400", `{"created":"2000-01-01T00:00:00+11:00","amount":"12.50","description":"test"}`},
		{"POST", fmt.Sprintf("%s/transactions", ts.URL), `{"data":{"id":1,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:00+11:00"}}`, "200", `{"created":"2000-01-01T00:00:00+11:00","amount":"12.50","description":"test","account":"1234567890"}`},
		{"GET", fmt.Sprintf("%s/transactions/1", ts.URL), `{"data":{"id":1,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:00+11:00"}}`, "200", ``},
		{"POST", fmt.Sprintf("%s/transactions", ts.URL), `{"data":{"id":2,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}}`, "200", `{"created":"2000-01-01T00:00:01+11:00","amount":"12.50","description":"test","account":"1234567890"}`},
		{"GET", fmt.Sprintf("%s/transactions/2", ts.URL), `{"data":{"id":2,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}}`, "200", ``},
		{"POST", fmt.Sprintf("%s/transactions", ts.URL), `{"error":{"Code":19,"ExtendedCode":2067,"SystemErrno":0}}`, "400", `{"created":"2000-01-01T00:00:01+11:00","amount":"12.50","description":"test","account":"1234567890"}`},
		{"GET", fmt.Sprintf("%s/transactions?id=1", ts.URL), `{"data":[{"id":1,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:00+11:00"}]}`, "200", ""},
		{"GET", fmt.Sprintf("%s/transactions?id__gt=1", ts.URL), `{"data":[{"id":2,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}]}`, "200", ""},
		{"POST", fmt.Sprintf("%s/transactions", ts.URL), `{"data":{"id":3,"description":"target","amount":12.5,"account":987654321,"created":"2000-01-03T00:00:01+11:00"}}`, "200", `{"created":"2000-01-03T00:00:01+11:00","amount":"12.50","description":"target","account":"0987654321"}`},
		{"GET", fmt.Sprintf("%s/transactions?description__like=target", ts.URL), `{"data":[{"id":3,"description":"target","amount":12.5,"account":987654321,"created":"2000-01-03T00:00:01+11:00"}]}`, "200", ""},
	}

	for _, test := range tests {
		var code int
		var response string
		var err error
		if test[0] == "GET" {
			code, response, err = RunGetRequest(test[1])
		} else {
			code, response, err = RunPostRequest(test[1], test[4])
		}

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		c, _ := strconv.Atoi(test[3])

		if code != c {
			t.Fatalf("%s %s - Expected status code %v, got %v", test[0], test[1], test[3], code)
		}

		if response != test[2] {
			t.Fatalf("%s %s - Expected '%s' got '%s'", test[0], test[1], test[2], response)
		}
	}
}
