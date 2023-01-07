package notify

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"bou.ke/monkey"
	fage "filippo.io/age"
	"github.com/codingric/moneyman/pkg/age"
	"github.com/rapidloop/skv"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type MockRoundTripper func(req *http.Request) (res *http.Response, err error)

func (m MockRoundTripper) RoundTrip(req *http.Request) (res *http.Response, err error) {
	return m(req)
}

func TestNotify(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	config := `
notify:
  sid: "sometihng"
  token: "something"
  mobiles:
  - "+6140000000"
store: "/tmp/test.db"`
	type H struct {
		body []byte
		code int
		err  error
	}
	type F struct {
		config  string
		message string
		store   bool
		twillio *H
	}
	type E struct {
		err       string
		sent      int
		httpcalls int
	}
	type T struct {
		name    string
		fixture F
		expect  E
	}
	tests := []T{
		{
			name:    "No config",
			fixture: F{``, "no config", false, nil},
			expect:  E{"unable to load settings", 0, 0},
		},
		{
			name:    "Exist store",
			fixture: F{config, "Exist store", true, nil},
			expect:  E{},
		},
		{
			name:    "HTTP error",
			fixture: F{config, "HTTP error", false, &H{err: errors.New("something went wrong")}},
			expect:  E{err: "failed to make request", httpcalls: 1},
		},
		{
			name:    "Auth error",
			fixture: F{config, "HTTP error", false, &H{code: 401, body: []byte("")}},
			expect:  E{err: "authentication failure", httpcalls: 1},
		},
		{
			name:    "XML error",
			fixture: F{config, "Other error", false, &H{code: 400, body: []byte(`<TwilioResponse><RestException><Code>30000</Code><Detail></Detail><Message>Fake Error</Message><MoreInfo>fake.com</MoreInfo><Status>400</Status></RestException></TwilioResponse>`)}},
			expect:  E{err: "Fake Error", httpcalls: 1},
		},
		{
			name:    "Unmarshalble error",
			fixture: F{config, "Unmarshalble error", false, &H{code: 400, body: []byte(`<<Tw~!!`)}},
			expect:  E{err: "failed to read responses", httpcalls: 1},
		},
		{
			name:    "Other error",
			fixture: F{config, "Other error", false, &H{code: 500, body: []byte("Server Error")}},
			expect:  E{err: "twilio responded with failure", httpcalls: 1},
		},
		{
			name:    "Happy",
			fixture: F{config, "Happy path", false, &H{code: 201, body: []byte("OK")}},
			expect:  E{sent: 1, httpcalls: 1},
		},
		{
			name:    "Dryrun",
			fixture: F{config + "\ndryrun: true", "Dryrun", false, &H{code: 201, body: []byte("OK")}},
			expect:  E{sent: 0, httpcalls: 0},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			defer monkey.UnpatchAll()
			viper.SetConfigType("yaml")
			viper.ReadConfig(bytes.NewBuffer([]byte(test.fixture.config)))
			skvStore = &skv.KVStore{}
			httpcalls := 0
			httpClient = &http.Client{Transport: MockRoundTripper(func(req *http.Request) (res *http.Response, err error) {
				httpcalls += 1
				return &http.Response{StatusCode: test.fixture.twillio.code, Body: io.NopCloser(bytes.NewBuffer(test.fixture.twillio.body))}, test.fixture.twillio.err
			})}
			monkey.Patch(age.DecodeAge, func(s string, a *fage.X25519Identity) string { return s })
			monkey.PatchInstanceMethod(reflect.TypeOf((*skv.KVStore)(nil)), "Get", func(*skv.KVStore, string, interface{}) error {
				if test.fixture.store {
					return nil
				}
				return errors.New("")
			})
			monkey.PatchInstanceMethod(reflect.TypeOf((*skv.KVStore)(nil)), "Put", func(*skv.KVStore, string, interface{}) error { return nil })

			sent, err := Notify(test.fixture.message, context.Background())
			if test.expect.err == "" {
				assert.Nil(tt, err)
			} else {
				assert.Error(tt, err)
				assert.EqualError(tt, err, test.expect.err)
			}
			assert.Equal(tt, test.expect.sent, sent)
			assert.Equal(tt, test.expect.httpcalls, httpcalls)
		})
	}
}
