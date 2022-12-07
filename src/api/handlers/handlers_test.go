package handlers

import (
	"fmt"
	"moneyman/auth"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bou.ke/monkey"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func Test_ReadyHandler(t *testing.T) {

	req, _ := http.NewRequest("GET", "/healthz/ready", nil)
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(ReadyHandler)

	handler.ServeHTTP(w, req)
	assert.Equal(t, w.Result().StatusCode, 200, "Incorrect return status code")

	response_body, _ := w.Body.ReadString('\n')
	assert.Equal(t, response_body, "OK\n", "Body should be OK")
}

func Test_AuthenticateHandler(t *testing.T) {

	logrus.SetLevel(logrus.FatalLevel)

	type I struct {
		body   string
		method string
		auth   string
	}
	type GT struct {
		token string
		err   string
	}
	type E struct {
		status int
		body   string
	}
	tests := []struct {
		name           string
		inputs         I
		generate_token GT
		expected       E
	}{
		{
			"Empty GET",
			I{"", "", ""},
			GT{"", ""},
			E{400, http.StatusText(400)},
		},
		{
			"GET",
			I{"dddd", "GET", "xxx"},
			GT{"", ""},
			E{400, http.StatusText(400)},
		},
		{
			"POST NoAuth",
			I{"", "POST", ""},
			GT{"", ""},
			E{400, http.StatusText(400)},
		},
		{
			"POST Incorrect Auth",
			I{"", "POST", "Basic dXNlcjpwYXNz"},
			GT{"", ""},
			E{403, http.StatusText(http.StatusForbidden)},
		},
		{
			"POST Incorrect Auth",
			I{"", "POST", "Basic dXNlcjpwYXNz"},
			GT{"fake token", ""},
			E{200, `{"token":"fake token"}`},
		},
		{
			"POST 500",
			I{"", "POST", "Basic dXNlcjpwYXNz"},
			GT{"", "Something went wrong"},
			E{500, http.StatusText(500)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			// setup
			req, _ := http.NewRequest(test.inputs.method, "/authenticate", strings.NewReader(test.inputs.body))
			if test.inputs.auth != "" {
				req.Header.Add("Authorization", test.inputs.auth)
			}
			w := httptest.NewRecorder()
			monkey.Patch(auth.GenerateToken, func(string, string) (t string, e error) {
				t = test.generate_token.token
				if test.generate_token.err != "" {
					e = fmt.Errorf(test.generate_token.err)
				}
				return
			})
			//run
			http.HandlerFunc(AuthenticateHandler).ServeHTTP(w, req)
			//validate results
			assert.Equal(tt, w.Result().StatusCode, test.expected.status)
			assert.Equal(tt, test.expected.body, strings.TrimRight(w.Body.String(), "\n"))
		})
	}
}
