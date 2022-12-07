package auth

import (
	"fmt"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jtblin/go-ldap-client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func Test_Authenticate(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	type I struct {
		user         string
		pw           string
		auth_ok      bool
		auth_fields  map[string]string
		auth_err     string
		group_groups []string
		group_err    string
	}
	type R struct {
		user string
		err  string
	}
	tests := []struct {
		name     string
		input    I
		expected R
	}{
		{
			"Happy",
			I{"rsalinas", "secret", false, map[string]string{"name": "Ricardo Salinas", "mail": "ricardo@x.com"}, "", []string{"Required"}, ""},
			R{"Ricardo Salinas", ""},
		},
		{
			"Invalid Password",
			I{"rsalinas", "secret", false, map[string]string{}, "Invalid Credentials", []string{}, ""},
			R{"", ""},
		},
		{
			"Missing Group",
			I{"rsalinas", "secret", false, map[string]string{}, "", []string{}, ""},
			R{"", ""},
		},
		{
			"LDAP Auth Error",
			I{"rsalinas", "secret", false, map[string]string{}, "Auth Error", []string{}, ""},
			R{"", "Auth Error"},
		},
		{
			"LDAP Group Error",
			I{"rsalinas", "secret", false, map[string]string{}, "", []string{}, "Group Error"},
			R{"", "Group Error"},
		},
	}
	for _, test := range tests {
		c := &ldap.LDAPClient{}
		monkey.PatchInstanceMethod(reflect.TypeOf(c), "Authenticate", func(*ldap.LDAPClient, string, string) (ok bool, f map[string]string, err error) {
			ok = test.input.auth_ok
			f = test.input.auth_fields
			if test.input.auth_err != "" {
				err = fmt.Errorf(test.input.auth_err)
			}
			return
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(c), "GetGroupsOfUser", func(*ldap.LDAPClient, string) (g []string, err error) {
			g = test.input.group_groups
			if test.input.group_err != "" {
				err = fmt.Errorf(test.input.group_err)
			}
			return
		})
		viper.Set("auth.required_group", "Required")
		t.Run(test.name, func(tt *testing.T) {
			u, e := Authenticate(test.input.user, test.input.pw)
			//validate results
			if test.expected.user == "" {
				assert.Nil(tt, u)
			} else {
				if assert.NotNil(tt, u) {
					assert.Equal(tt, test.expected.user, u.Name)
				}
			}

			if test.expected.err == "" {
				assert.NoError(tt, e)
			} else {
				assert.EqualError(tt, e, test.expected.err)
			}
		})
		monkey.UnpatchAll()
	}
}

func Test_GenerateToken(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	type A struct {
		user *AuthenticatedUser
		err  string
	}
	type R struct {
		token string
		err   string
	}
	tests := []struct {
		name     string
		auth     A
		signed   R
		expected R
	}{
		{
			"Happy",
			A{&AuthenticatedUser{"joe", "joe@mail.com"}, ""},
			R{"token", ""},
			R{"token", ""},
		},
		{
			"Invalid auth",
			A{nil, ""},
			R{"", ""},
			R{"", ""},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			// setup
			j := jwt.NewWithClaims(jwt.SigningMethodHS256, AuthClaims{
				AuthenticatedUser: AuthenticatedUser{Name: "joe", Email: "joe@mail.com"},
				StandardClaims:    jwt.StandardClaims{NotBefore: 0},
			})
			monkey.PatchInstanceMethod(reflect.TypeOf(j), "SignedString", func(*jwt.Token, interface{}) (t string, e error) {
				if test.expected.token != "" {
					t = test.expected.token
				}
				if test.expected.err != "" {
					e = fmt.Errorf(test.expected.err)
				}
				return
			})
			monkey.Patch(Authenticate, func(string, string) (u *AuthenticatedUser, e error) {
				if test.auth.user != nil {
					u = test.auth.user
				}
				if test.auth.err != "" {
					e = fmt.Errorf(test.auth.err)
				}
				return
			})
			//run
			token, err := GenerateToken("joe", "pass")
			//validate results
			assert.Equal(tt, token, test.expected.token)
			if test.expected.err == "" {
				assert.Nil(tt, err)
			} else {
				assert.EqualErrorf(tt, err, test.expected.err, "")
			}
			monkey.UnpatchAll()
		})
	}
}

func Test_VerifyToken(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	type I struct {
		token  string
		secret string
	}
	type R struct {
		user string
		err  string
	}
	tests := []struct {
		name     string
		input    I
		expected R
	}{
		{
			"Happy",
			I{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiUmljYXJkbyBTYWxpbmFzIiwiZW1haWwiOiJyaWNhcmRvLnNhbGluYXNAYWx0YXZlYy5jb20iLCJuYmYiOjE0NDQ0Nzg0MDB9.x50li3L7MEex4dStwp9M7mgFnAKdTl-XC_wyAWUM6TA", "hoxYdbX3Ww"},
			R{"Ricardo Salinas", ""},
		},
		{
			"Invalid secret",
			I{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiUmljYXJkbyBTYWxpbmFzIiwiZW1haWwiOiJyaWNhcmRvLnNhbGluYXNAYWx0YXZlYy5jb20iLCJuYmYiOjE0NDQ0Nzg0MDB9.x50li3L7MEex4dStwp9M7mgFnAKdTl-XC_wyAWUM6TA", "ho"},
			R{"", "signature is invalid"},
		},
		{
			"Invalid token",
			I{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiUmljYXJkbyBTYWxpbmFzIiwiZW1haWwiOiJyaWNhcmRvLnNhbGluYXNAYWx0YXZlYy5jb20iLCJuYmYiOjE0NDzg0MDB9.x50li3L7MEex4dp9M7mgFnAKdTl-XC_wyAWUM6TA", "hoxYdbX3Ww"},
			R{"", "illegal base64 data at input byte 104"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			// setup)
			viper.Set("server.secret", test.input.secret)
			//run
			user, err := VerifyToken(test.input.token)
			//validate results
			assert.Equal(tt, test.input.secret, viper.GetString("server.secret"))
			if test.expected.err == "" {
				assert.Nil(tt, err)
			} else {
				assert.EqualErrorf(tt, err, test.expected.err, "")
			}
			if test.expected.user != "" {
				assert.NotNil(tt, user)
				assert.Equal(tt, user.Name, test.expected.user)
			}
		})
	}
}
