package notify

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

type MockRoundTripper func(req *http.Request) (res *http.Response, err error)

func (m MockRoundTripper) RoundTrip(req *http.Request) (res *http.Response, err error) {
	return m(req)
}

func TestMain(m *testing.M) {
	log.Logger = zerolog.New(&bytes.Buffer{}).With().Logger()
	os.Exit(m.Run())
}

//go:noinline
func TestNotify(t *testing.T) {
	config := &Settings{
		Sid:     "sid",
		Token:   "token",
		Mobiles: []string{"+6140000000"},
	}
	type D struct {
		err      string
		req_hash string
		res_body string
		res_code int
	}
	tests := []struct {
		name      string
		config    *Settings
		client_do *D
		result    string
		err       string
	}{
		{
			"Empty",
			nil,
			nil,
			"",
			"missing notify config",
		},
		{
			"Err",
			config,
			&D{"Failed", "", "", 500},
			"",
			"Post \"https://api.twilio.com/2010-04-01/Accounts/sid/Messages\": Failed",
		},
		{
			"Success",
			config,
			&D{"", "ced9f50b77dcd5841e353784258f7219", "<Success />", 201},
			"<Success />",
			"",
		},
		{
			"Non201",
			config,
			&D{"", "", "<Something />", 200},
			"<Something />",
			"ReponseCode: 200",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(st *testing.T) {
			settings = test.config
			log.Logger.Level(zerolog.Disabled)

			m := MockRoundTripper(func(req *http.Request) (r *http.Response, e error) {
				if test.client_do.req_hash != "" {
					v, _ := httputil.DumpRequest(req, true)
					fmt.Printf("%s", v)
					digest := md5.New()
					digest.Write(v)
					hash := digest.Sum(nil)
					assert.Equal(st, test.client_do.req_hash, fmt.Sprintf("%x", hash), "Request hash doesn't match")
				}
				if test.client_do.err != "" {
					return nil, errors.New(test.client_do.err)
				}
				h := &http.Response{StatusCode: test.client_do.res_code, Body: ioutil.NopCloser(bytes.NewReader([]byte(test.client_do.res_body)))}
				return h, nil
			})
			client = &http.Client{Transport: m}

			result, err := Notify("message", context.Background())

			if test.err == "" {
				assert.Nil(st, err)
			} else {
				assert.NotNil(st, err)
				if err != nil {
					assert.Equal(st, test.err, err.Error())
				}
			}
			assert.Equal(st, test.result, result)
		})
	}
}
