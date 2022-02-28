package main

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
)

func Test_BackendTransaction(t *testing.T) {
	type setup struct {
		transaction *BackendTransaction
		requesterr  bool
		doerr       bool
		statuscode  int
	}
	type expected struct {
		err     string
		reqbody string
	}
	tests := []struct {
		name     string
		setup    setup
		expected expected
	}{
		{
			"Empty",
			setup{transaction: nil},
			expected{err: "uninitialized transaction"},
		},
		{
			"AlreadySuccessful",
			setup{transaction: &BackendTransaction{Successful: true}},
			expected{},
		},
		{
			"FailedRequest",
			setup{transaction: &BackendTransaction{}, requesterr: true},
			expected{err: "http.NewRequest error"},
		},
		{
			"FailedResponse",
			setup{transaction: &BackendTransaction{}, doerr: true},
			expected{err: "http.Client.Do error"},
		},
		{
			"FailedResponse",
			setup{transaction: &BackendTransaction{}, doerr: true},
			expected{err: "http.Client.Do error"},
		},
		{
			"Non200Status",
			setup{transaction: &BackendTransaction{}, statuscode: 500},
			expected{err: "unsucessful statuscode returned"},
		},
		{
			"OK",
			setup{transaction: &BackendTransaction{
				Created:     time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local),
				Amount:      "100.00",
				Description: "Mock",
				Account:     "1234567890",
			}, statuscode: 200},
			expected{reqbody: `{"created":"2000-01-01T00:00:00+11:00","amount":"100.00","description":"Mock","account":"1234567890"}`},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {

			var req_guard *monkey.PatchGuard
			req_guard = monkey.Patch(http.NewRequest, func(method string, url string, body io.Reader) (r *http.Request, e error) {
				if test.setup.requesterr {
					e = errors.New("http.NewRequest error")
				} else {
					req_guard.Unpatch()
					defer req_guard.Restore()
					return http.NewRequest(method, url, body)
				}
				return
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Do", func(c *http.Client, req *http.Request) (r *http.Response, e error) {
				if test.expected.reqbody != "" {
					body, _ := ioutil.ReadAll(req.Body)
					assert.Equal(tt, test.expected.reqbody, string(body))
				}
				if test.setup.doerr {
					e = errors.New("http.Client.Do error")
				}
				r = &http.Response{StatusCode: test.setup.statuscode, Body: ioutil.NopCloser(bytes.NewBufferString("{}"))}
				return
			})

			err := test.setup.transaction.Post()
			monkey.UnpatchAll()

			if test.expected.err != "" && assert.NotNil(tt, err) {
				assert.Equal(tt, test.expected.err, err.Error())
			} else {
				assert.Nil(tt, err)
			}
		})
	}
}
