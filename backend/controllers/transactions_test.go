package controllers

import (
	"bytes"
	"io/ioutil"
	"models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	models.Debug = true
	models.ConnectDatabase(":memory:", false)
	m.Run()
}

func TestCreateTransaction(t *testing.T) {

	tests := []struct {
		name        string
		post        []byte
		status_code int
		expect      string
	}{
		{
			"InvalidRequest",
			[]byte{},
			400,
			`{"error":"EOF"}`,
		},
		{
			"Successful",
			[]byte(`{"created":"2000-01-01T00:00:01+11:00","amount":"12.50","description":"test","account":"1234567890"}`),
			200,
			`{"data":{"id":1,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}}`,
		},
		{
			"Duplicate",
			[]byte(`{"created":"2000-01-01T00:00:01+11:00","amount":"12.50","description":"test","account":"1234567890"}`),
			400,
			`{"error":{"Code":19,"ExtendedCode":2067,"SystemErrno":0}}`,
		},
		{
			"SameTime",
			[]byte(`{"created":"2000-01-01T00:00:01+11:00","amount":"20.50","description":"test two","account":"1234567890"}`),
			200,
			`{"data":{"id":2,"description":"test two","amount":20.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(st *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			r, _ := http.NewRequest("POST", "/", bytes.NewBuffer(test.post))

			c.Request = r

			CreateTransaction(c)

			assert.Equal(st, w.Code, test.status_code)
			b, _ := ioutil.ReadAll(w.Body)
			assert.Equal(st, test.expect, string(b))
		})
	}
}

func TestFindTransaction(t *testing.T) {

	tests := []struct {
		name        string
		params      []gin.Param
		status_code int
		expect      string
	}{
		{
			"InvalidRequest",
			[]gin.Param{},
			400,
			`{"error":"Record not found!"}`,
		},
		{
			"Successful",
			[]gin.Param{{"id", "1"}},
			200,
			`{"data":{"id":1,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(st *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = test.params

			FindTransaction(c)

			assert.Equal(st, w.Code, test.status_code)
			b, _ := ioutil.ReadAll(w.Body)
			assert.Equal(st, test.expect, string(b))
		})
	}
}

func TestFindTransactions(t *testing.T) {

	tests := []struct {
		name        string
		url         string
		status_code int
		expect      string
	}{
		{
			"InvalidRequest",
			`x?=&=`,
			200,
			`{"data":null}`,
		},
		{
			"Like",
			`x?description__like=two`,
			200,
			`{"data":[{"id":2,"description":"test two","amount":20.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}]}`,
		},
		{
			"GreaterThan",
			`x?amount__gt=15`,
			200,
			`{"data":[{"id":2,"description":"test two","amount":20.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}]}`,
		},
		{
			"LessThan",
			`x?amount__lt=15`,
			200,
			`{"data":[{"id":1,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}]}`,
		},
		{
			"GreaterEqual",
			`x?amount__ge=20.5`,
			200,
			`{"data":[{"id":2,"description":"test two","amount":20.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}]}`,
		},
		{
			"LessEqual",
			`x?amount__le=12.5`,
			200,
			`{"data":[{"id":1,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}]}`,
		},
		{
			"Date",
			`x?created__gt=2000-01-01T00:00:00`,
			200,
			`{"data":[{"id":1,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"},{"id":2,"description":"test two","amount":20.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}]}`,
		},
		{
			"NotEqual",
			`x?id__ne=2`,
			200,
			`{"data":[{"id":1,"description":"test","amount":12.5,"account":1234567890,"created":"2000-01-01T00:00:01+11:00"}]}`,
		},
		{
			"InvalidOperator",
			`x?id__xx=2`,
			400,
			`{"error":"invalid operator xx"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(st *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			r, _ := http.NewRequest("GET", test.url, nil)
			c.Request = r

			FindTransactions(c)

			assert.Equal(st, w.Code, test.status_code)
			b, _ := ioutil.ReadAll(w.Body)
			assert.Equal(st, test.expect, string(b))
		})
	}
}
