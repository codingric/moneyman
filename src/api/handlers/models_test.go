package handlers

import (
	"fmt"
	"moneyman/auth"
	"moneyman/models"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"bou.ke/monkey"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func Test_ModelHandler(t *testing.T) {

	logrus.SetLevel(logrus.FatalLevel)

	type I struct {
		body   string
		method string
		auth   string
	}
	type VT struct {
		user *auth.AuthenticatedUser
		err  string
	}
	type E struct {
		status   int
		body     string
		versions int64
	}
	tests := []struct {
		name     string
		inputs   I
		vt       VT
		expected E
	}{
		{
			"GET No Auth",
			I{"", "GET", ""},
			VT{nil, "no bearer token"},
			E{403, http.StatusText(http.StatusForbidden), 0},
		},
		{
			"GET Invalid Auth",
			I{"", "GET", "xxx"},
			VT{nil, "no bearer token"},
			E{403, http.StatusText(http.StatusForbidden), 0},
		},
		{
			"GET Invalid Token",
			I{"", "GET", "Bearer dXNlcjpwYXNz"},
			VT{nil, "Invalid"},
			E{403, http.StatusText(http.StatusForbidden), 0},
		},
		{
			"GET Valid",
			I{"", "GET", "Bearer dXNlcjpwYXNz"},
			VT{&auth.AuthenticatedUser{Name: "U", Email: "u@u.com"}, ""},
			E{200, `[]`, 0},
		},
		{
			"POST invalid body",
			I{"", "POST", "Bearer dXNlcjpwYXNz"},
			VT{&auth.AuthenticatedUser{Name: "U", Email: "u@u.com"}, ""},
			E{400, http.StatusText(400), 0},
		},
		{
			"POST missing required",
			I{`{"name":"POST invalid version"}`, "POST", "Bearer dXNlcjpwYXNz"},
			VT{user: &auth.AuthenticatedUser{Name: "U", Email: "u@u.com"}, err: ""},
			E{400, `{"message":"missing required","status_code":400}`, 0},
		},
		{
			"POST happy",
			I{`{"name":"POST happy", "notes":"Happy path"}`, "POST", "Bearer dXNlcjpwYXNz"},
			VT{&auth.AuthenticatedUser{Name: "U", Email: "u@u.com"}, ""},
			E{200, `{"ID":0,"CreatedAt":"0001-01-01T00:00:00Z","UpdatedAt":"0001-01-01T00:00:00Z","DeletedAt":null,"name":"POST happy","notes":"Happy path","amount":0,"enabled":false,"creator_id":0,"from_account_id":0,"to_account_id":0,"start":"0001-01-01T00:00:00Z","end":"0001-01-01T00:00:00Z","period":""}`, 1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			// setup
			viper.Set("database.dsn", ":memory:")
			models.InitDB()
			b := models.Budgets{}
			req, _ := http.NewRequest(test.inputs.method, "/api/v1/version", strings.NewReader(test.inputs.body))
			if test.inputs.auth != "" {
				req.Header.Add("Authorization", test.inputs.auth)
			}
			w := httptest.NewRecorder()
			monkey.Patch(auth.ValidateBearerToken, func(h string) (a *auth.AuthenticatedUser, e error) {
				if test.vt.user != nil {
					a = &auth.AuthenticatedUser{
						Name:  test.vt.user.Name,
						Email: test.vt.user.Email,
					}
				}
				if test.vt.err != "" {
					e = fmt.Errorf(test.vt.err)
				}
				return
			})
			monkey.PatchInstanceMethod(
				reflect.TypeOf((*models.Budget)(nil)),
				"Add",
				func(vv *models.Budget) (e error) {
					if vv.Notes == "" {
						return fmt.Errorf("missing required")
					}
					b = append(b, models.Budget{Notes: vv.Notes})
					return
				},
			)
			defer monkey.UnpatchAll()
			//run
			http.HandlerFunc(ModelHandler[*models.BudgetModelFactory]).ServeHTTP(w, req)
			//validate results
			// pretty.Printf("v.Versions:\n%# v", v.Versions)
			assert.Equal(tt, test.expected.status, w.Result().StatusCode)
			assert.Equal(tt, test.expected.body, strings.TrimRight(w.Body.String(), "\n"))
			assert.Equal(tt, test.expected.versions, int64(len(b)), "Total versions in database")
		})
	}
}
