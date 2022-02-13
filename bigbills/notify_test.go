package main

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

//go:noinline
func TestNotify(t *testing.T) {
	config := `sid: sid
token: token
mobiles:
  - "+614000000000"
`
	type D struct {
		err      string
		req_hash string
		res_body string
		res_code int
	}
	tests := []struct {
		name      string
		config    string
		client_do *D
		result    string
		err       string
	}{
		{
			"Empty",
			``,
			nil,
			"",
			"Missing notify config",
		},
		{
			"Err",
			config,
			&D{"Failed", "", "", 500},
			"",
			"Failed",
		},
		{
			"Success",
			config,
			&D{"", "d17f49f3923f417101424791606f24a6", "<Success />", 201},
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
			viper.SetConfigType("yaml")
			viper.ReadConfig(bytes.NewBuffer([]byte(test.config)))

			monkey.PatchInstanceMethod(
				reflect.TypeOf(http.DefaultClient),
				"Do",
				func(c *http.Client, req *http.Request) (*http.Response, error) {
					if test.client_do.req_hash != "" {
						v, _ := httputil.DumpRequest(req, true)
						digest := md5.New()
						digest.Write(v)
						hash := digest.Sum(nil)
						assert.Equal(st, test.client_do.req_hash, fmt.Sprintf("%x", hash))
					}
					if test.client_do.err != "" {
						return nil, errors.New(test.client_do.err)
					}
					h := &http.Response{StatusCode: test.client_do.res_code, Body: ioutil.NopCloser(bytes.NewReader([]byte(test.client_do.res_body)))}
					return h, nil
				})
			defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Do")

			result, err := Notify("message")

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
