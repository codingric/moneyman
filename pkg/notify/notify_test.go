package notify

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"bou.ke/monkey"
	fage "filippo.io/age"
	"github.com/alicebob/miniredis/v2"
	"github.com/codingric/moneyman/pkg/age"
	"github.com/go-redis/redis/v8"
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

	config := `[notify]
sid = "twilliosid"
token = "twiliotoken"
mobiles = ["+6140000000"]`
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
		wait    time.Duration
		redis   string
	}
	type E struct {
		err       string
		sent      int
		httpcalls int
		store     bool
	}
	type T struct {
		name    string
		fixture F
		expect  E
	}
	tests := []T{
		{
			name:    "No config",
			fixture: F{config: ``, message: "no config"},
			expect:  E{err: "unable to load settings"},
		},
		{
			name:    "redis host error",
			fixture: F{config: config, message: "redis host error", redis: "nonexistant:12345"},
			expect:  E{err: "failure reaching redis"},
		},
		{
			name:    "Exist store",
			fixture: F{config: config, message: "Exist store", store: true},
			expect:  E{sent: 0},
		},
		{
			name:    "Test expired store",
			fixture: F{config: config, message: "Expired store", store: true, wait: 3, twillio: &H{code: 201, body: []byte("OK")}},
			expect:  E{sent: 1, httpcalls: 1, store: true},
		},
		{
			name:    "HTTP error",
			fixture: F{config: config, message: "HTTP error", twillio: &H{err: errors.New("something went wrong")}},
			expect:  E{err: "failed to make request", httpcalls: 1},
		},
		{
			name: "Auth error",
			fixture: F{
				config:  config,
				message: "HTTP error",
				twillio: &H{code: 401, body: []byte("")},
			},
			expect: E{err: "authentication failure", httpcalls: 1},
		},
		{
			name: "XML error",
			fixture: F{
				config:  config,
				message: "Other error",
				twillio: &H{code: 400, body: []byte(`<TwilioResponse><RestException><Code>30000</Code><Detail></Detail><Message>Fake Error</Message><MoreInfo>fake.com</MoreInfo><Status>400</Status></RestException></TwilioResponse>`)},
			},
			expect: E{err: "Fake Error", httpcalls: 1},
		},
		{
			name: "Unmarshalble error",
			fixture: F{
				config:  config,
				message: "Unmarshalble error",
				twillio: &H{code: 400, body: []byte(`<<Tw~!!`)},
			},
			expect: E{err: "failed to read responses", httpcalls: 1},
		},
		{
			name: "Other error",
			fixture: F{config: config,
				message: "Other error",
				twillio: &H{code: 500, body: []byte("Server Error")},
			},
			expect: E{err: "twilio responded with failure", httpcalls: 1},
		},
		{
			name: "Happy",
			fixture: F{
				config:  config,
				message: "Happy path",
				twillio: &H{code: 201, body: []byte("OK")},
			},
			expect: E{sent: 1, httpcalls: 1, store: true},
		},
		{
			name: "Dryrun",
			fixture: F{
				config:  config + "\ndryrun: true",
				message: "Dryrun",
				twillio: &H{code: 201, body: []byte("OK")},
			},
			expect: E{sent: 0, httpcalls: 0},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			defer monkey.UnpatchAll()
			viper.SetConfigType("toml")
			viper.ReadConfig(bytes.NewBuffer([]byte(test.fixture.config)))
			httpcalls := 0
			httpClient = &http.Client{Transport: MockRoundTripper(func(req *http.Request) (res *http.Response, err error) {
				httpcalls += 1
				return &http.Response{StatusCode: test.fixture.twillio.code, Body: io.NopCloser(bytes.NewBuffer(test.fixture.twillio.body))}, test.fixture.twillio.err
			})}
			monkey.Patch(age.DecodeAge, func(s string, a *fage.X25519Identity) string { return s })
			monkey.Patch(generateHash, func(string) (s string) {
				return "something"
			})
			s := miniredis.RunT(t)
			redisClient = redis.NewClient(&redis.Options{
				Addr:     s.Addr(), // host:port of the redis server
				Password: "",       // no password set
				DB:       0,        // use default DB
			})
			if test.fixture.store {
				s.Set("something", "something")
				s.SetTTL("something", 1*time.Second)
				s.FastForward(test.fixture.wait * time.Second)
				if s.Exists("someting") {
					tt.Error("shouldnt exist")
				}
			}
			if test.fixture.redis != "" {
				redisClient = redis.NewClient(&redis.Options{
					Addr:     test.fixture.redis, // host:port of the redis server
					Password: "",                 // no password set
					DB:       0,                  // use default DB
				})
			}

			sent, err := Notify(test.fixture.message, context.Background())
			if test.expect.err == "" {
				assert.Nil(tt, err)
			} else {
				assert.Error(tt, err)
				assert.EqualError(tt, err, test.expect.err)
			}
			assert.Equal(tt, test.expect.sent, sent, "Sent messages")
			assert.Equal(tt, test.expect.httpcalls, httpcalls, "HTTP calls")
			if test.expect.store {
				s.CheckGet(t, "something", "+6140000000")
			}
		})
	}
}
