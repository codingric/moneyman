package main

import (
	"fmt"
	"io/ioutil"
	"models"
	"net/http"
	"net/http/httptest"
	"os"
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

func RunTest(t *testing.T, method string, url string, payload string, expected string, statusCode string) {

	var code int
	var response string
	var err error
	if method == "GET" {
		code, response, err = RunGetRequest(fmt.Sprintf("%s/%s", TestServer.URL, url))
	} else {
		code, response, err = RunPostRequest(fmt.Sprintf("%s/%s", TestServer.URL, url), payload)
	}

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	c, _ := strconv.Atoi(statusCode)

	if code != c {
		t.Fatalf("%s %s - Expected status code %v, got %v", method, url, statusCode, code)
	}

	if response != expected {
		t.Fatalf("%s %s - Expected '%s' got '%s'", method, url, expected, response)
	}

}

var TestServer *httptest.Server

func TestMain(m *testing.M) {
	models.ConnectDatabase(":memory:", false)
	TestServer = httptest.NewServer(setupServer(false))
	defer TestServer.Close()
	retCode := m.Run()
	os.Exit(retCode)
}
func TestCleanTransactions(t *testing.T) {
	RunTest(t, "GET", "transactions", ``, `{"data":[]}`, "200")
}

func TestRecordNotFound(t *testing.T) {
	RunTest(t, "GET", "transactions/1", ``, `{"error":"Record not found!"}`, "400")
}

func TestInvalidFilter(t *testing.T) {
	RunTest(t, "GET", "transactions?idx=1", "", `{"data":null}`, "200")
}

func TestEmptyPost(t *testing.T) {
	RunTest(t, "POST", "transactions", ``, `{"error":"EOF"}`, "400")
}

func TestCreateMissingField(t *testing.T) {
	RunTest(t, "POST", "transactions", `{"created":"2000-01-01T00:00:00+11:00","amount":"12.50","description":"test"}`, `{"error":"Key: 'CreateTransactionInput.Account' Error:Field validation for 'Account' failed on the 'required' tag"}`, "400")
}

func TestCreateSucessfully(t *testing.T) {
	RunTest(t, "POST", "transactions", `{"created":"2000-01-01T00:00:01+11:00","amount":"12.50","description":"test","account":"1234567890"}`, `{"data":{"id":1,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}}`, "200")
	RunTest(t, "POST", "transactions", `{"created":"2000-01-01T00:00:02+11:00","amount":"12.50","description":"test","account":"1234567890"}`, `{"data":{"id":2,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:02+11:00"}}`, "200")
}

func TestRetrieveSuccessfully(t *testing.T) {
	RunTest(t, "GET", "transactions/1", ``, `{"data":{"id":1,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}}`, "200")
}

func TestCreateDuplicate(t *testing.T) {
	RunTest(t, "POST", "transactions", `{"created":"2000-01-01T00:00:01+11:00","amount":"12.50","description":"test","account":"1234567890"}`, `{"error":{"Code":19,"ExtendedCode":2067,"SystemErrno":0}}`, "400")
}

func TestFilterById(t *testing.T) {
	RunTest(t, "GET", "transactions?id=1", "", `{"data":[{"id":1,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}]}`, "200")
}

func TestFilterGreaterThan(t *testing.T) {
	RunTest(t, "GET", "transactions?id__gt=1", "", `{"data":[{"id":2,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:02+11:00"}]}`, "200")
}

func TestFilterLike(t *testing.T) {
	RunTest(t, "POST", "transactions", `{"created":"2000-01-03T00:00:03+11:00","amount":"12.50","description":"target","account":"0987654321"}`, `{"data":{"id":3,"description":"target","amount":12.5,"account":987654321,"created":"2000-01-03T00:00:03+11:00"}}`, "200")
	RunTest(t, "GET", "transactions?description__like=target", "", `{"data":[{"id":3,"description":"target","amount":12.5,"account":987654321,"created":"2000-01-03T00:00:03+11:00"}]}`, "200")
}

func TestFilterByCreated(t *testing.T) {
	RunTest(t, "GET", "transactions?created__gt=2000-01-03T00:00:00", "", `{"data":[{"id":3,"description":"target","amount":12.5,"account":987654321,"created":"2000-01-03T00:00:03+11:00"}]}`, "200")
}
