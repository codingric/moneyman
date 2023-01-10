package up

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.PanicLevel)
	os.Exit(m.Run())
}

type mockService struct {
	Returner func(req *http.Request) (*http.Response, error)
}

func (u *mockService) Do(req *http.Request) (*http.Response, error) {
	return u.Returner(req)
}

func Test_UpTransactionGet(t *testing.T) {
	type setup struct {
		do_err     bool
		do_body    string
		statuscode int
		//unmarshal_err bool
		data_id string
	}
	type expected struct {
		req_body string
		err      string
		data_id  string
	}

	tests := []struct {
		name     string
		setup    setup
		expected expected
	}{
		{
			"EmptyId",
			setup{do_err: true},
			expected{err: "Transaction.Get requires id"},
		},
		{
			"FailedDo",
			setup{do_err: true, data_id: "mock"},
			expected{err: "Failure while requesting Transaction(mock)"},
		},
		{
			"FailedStatusCode",
			setup{do_body: "{}", statuscode: 400, data_id: "mock"},
			expected{err: "Failure StatusCode while requsting Transaction(mock): 400"},
		},
		{
			"FailedUnmarshal",
			setup{do_body: "{", statuscode: 200, data_id: "mock"},
			expected{err: "Failure parsing response for Transaction(mock)"},
		},
		{
			"FailedHydrate",
			setup{do_body: "{}", statuscode: 200, data_id: "mock"},
			expected{err: "Failure retrieving data for Transaction(mock)"},
		},
		{
			"OK",
			setup{do_body: `{"data":{"id":"mock"}}`, statuscode: 200, data_id: "mock"},
			expected{data_id: "mock"},
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(tt *testing.T) {

			mock := &mockService{}
			mock.Returner = func(req *http.Request) (r *http.Response, e error) {
				if test.expected.req_body != "" {
					body, _ := ioutil.ReadAll(req.Body)
					assert.Equal(tt, test.expected.req_body, string(body), "Error wording")
				}
				if test.setup.do_err {
					return nil, errors.New("Failure from client.Do")
				}
				if test.setup.do_body != "" {
					r = &http.Response{StatusCode: test.setup.statuscode, Body: ioutil.NopCloser(bytes.NewBufferString(test.setup.do_body))}
				}
				return r, e
			}
			UpService = mock

			var trans UpTransaction
			err := trans.Get(test.setup.data_id, context.Background())
			if test.expected.err != "" {
				if assert.NotNil(tt, err) {
					assert.Equal(tt, test.expected.err, err.Error())
				}
			} else {
				assert.Nil(tt, err)
			}
			if test.expected.data_id != "" {
				assert.Equal(tt, test.expected.data_id, trans.Data.Id)
			}
		})
	}
}
