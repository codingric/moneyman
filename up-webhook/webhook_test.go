package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"bou.ke/monkey"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func Test_validateSignature(t *testing.T) {
	type I struct {
		body   string
		sig    string
		secret string
	}
	type E struct {
		result bool
		err    string
	}
	tests := []struct {
		name     string
		inputs   I
		expected E
	}{
		{
			"Empty",
			I{"", "", ""},
			E{false, "missing secret_key"},
		},
		{
			"Not matching",
			I{"", "f4b534dac27930fce40fa1cfbcabc053d7d7e1e30ecb143c27fcc1460f6b8c01", "secretkey"},
			E{false, ""},
		},
		{
			"Matching",
			I{"{}", "c7e3384754f9e646d99ba5c50b11c1bb0f02bed736e7912206f72850f941978e", "secretkey"},
			E{true, ""},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			// setup
			body := []byte(test.inputs.body)
			sig, _ := hex.DecodeString(test.inputs.sig)
			viper.Set("secret_key", test.inputs.secret)

			// run test
			result, err := validateSignature(body, sig)

			//validate results
			assert.Equal(tt, test.expected.result, result)
			if test.expected.err != "" {
				assert.NotNil(tt, err)
				assert.EqualError(tt, err, test.expected.err)
			} else {
				assert.Nil(tt, err)
			}
		})
	}
}

func Test_WebhookHandler(t *testing.T) {
	type setup struct {
		body        string
		validsig    bool
		validsigerr bool
		ioutilerr   bool
	}
	type expected struct {
		response string
		code     int
	}
	tests := []struct {
		name     string
		setup    setup
		expected expected
	}{
		{
			"FailedSignature",
			setup{validsigerr: true},
			expected{"Internal Server Error\n", 500},
		},
		{
			"InvalidSignatureSignature",
			setup{validsig: false},
			expected{"Unauthorized\n", 401},
		},
		{
			"FailedReadBody",
			setup{ioutilerr: true},
			expected{"Bad Request\n", 400},
		},
		{
			"InvalidBody",
			setup{validsig: true, body: `{:"{/"';]`},
			expected{"Bad Request\n", 400},
		},
		{
			"OK",
			setup{validsig: true, body: `{"data":{"id":"133ee2bf-0b09-4bb6-946f-fc36e8bb0936","type":"webhook-events","attributes":{"createdAt":"2022-02-27T08:28:54+11:00","eventType":"TRANSACTION_CREATED"},"relationships":{"webhook":{"data":{"id":"f7910b7e-23a4-4d37-bb0b-2a5975edc9ad","type":"webhooks"},"links":{"related":"https://api.up.com.au/api/v1/webhooks/f7910b7e-23a4-4d37-bb0b-2a5975edc9ad"}},"transaction":{"data":{"id":"9fa021d6-e26a-400a-b2a1-2daa4cc71ead","type":"transactions"},"links":{"related":"https://api.up.com.au/api/v1/transactions/9fa021d6-e26a-400a-b2a1-2daa4cc71ead"}}}}}`},
			expected{"OK\n", 200},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			// setup
			request, _ := http.NewRequest("POST", "/create", bytes.NewBuffer([]byte(test.setup.body)))
			response := httptest.NewRecorder()
			monkey.Patch(validateSignature, func([]byte, []byte) (v bool, e error) {
				if test.setup.validsigerr {
					e = errors.New("mock error")
				}
				v = test.setup.validsig
				return
			})
			if test.setup.ioutilerr {
				monkey.Patch(ioutil.ReadAll, func(r io.Reader) ([]byte, error) {
					return []byte{}, errors.New("mock error")
				})
			}

			// run test
			WebhookHandler(response, request)
			monkey.UnpatchAll()

			resp_body, _ := ioutil.ReadAll(response.Body)

			//validate results
			assert.Equal(tt, test.expected.code, response.Code)
			assert.Equal(tt, test.expected.response, string(resp_body))
		})
	}
}

func Test_RunWebhook(t *testing.T) {
	type I struct {
		port int
		err  bool
	}
	type E struct {
		addr string
	}
	tests := []struct {
		name     string
		inputs   I
		expected E
	}{
		{
			"Failure",
			I{port: 3333, err: true},
			E{"0.0.0.0:3333"},
		},
		{
			"OK",
			I{port: 8080},
			E{"0.0.0.0:8080"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			monkey.Patch(http.HandleFunc, func(pattern string, handler func(http.ResponseWriter, *http.Request)) {})
			viper.Set("port", test.inputs.port)
			monkey.Patch(http.ListenAndServe, func(addr string, handler http.Handler) error {
				assert.Equal(tt, test.expected.addr, addr)
				if test.inputs.err {
					return errors.New("mock error")
				}
				return nil
			})

			result := RunWebhook()

			if test.inputs.err {
				assert.NotNil(tt, result)
			} else {
				assert.Nil(tt, result)
			}

			monkey.UnpatchAll()
		})
	}
}
