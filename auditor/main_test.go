package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"bou.ke/monkey"
	"filippo.io/age"
	"github.com/rapidloop/skv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type MockRoundTripper func(req *http.Request) (res *http.Response, err error)

func (m MockRoundTripper) RoundTrip(req *http.Request) (res *http.Response, err error) {
	return m(req)
}

type MockHook struct {
	HookFunc func(e *zerolog.Event, level zerolog.Level, msg string)
}

func (h MockHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	h.HookFunc(e, level, msg)
}

func TestLoadConf(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	buf := bytes.NewBufferString("")
	log.Logger = log.Output(buf)
	key := `# created: 2022-12-14T09:40:56+11:00
# public key: age176vjsyk43d46tf5hcttcxe4kumzjxw4zrncmvemuxnt0kfvv7ffq4u7mlw
AGE-SECRET-KEY-1UNDK53VGXQXXY6KNF7D865UW7Y4ADEJTRUPMEX9499AWCTSELMQSNCJU8P`

	keyfile, err := os.CreateTemp("/tmp", "testing*.key")
	if err != nil {
		log.Fatal().Err(err)
	}
	keypath := keyfile.Name()
	defer os.Remove(keypath)
	io.WriteString(keyfile, key)
	keyfile.Close()

	conffile, err := os.CreateTemp("/tmp", "testing*.yaml")
	if err != nil {
		log.Fatal().Err(err)
	}
	confpath := conffile.Name()
	defer os.Remove(conffile.Name())
	io.WriteString(conffile, "notify:\n  sid: testsid")
	conffile.Close()

	type E struct {
		msg string
	}
	tests := []struct {
		name    string
		fixture []string
		expect  E
	}{
		{
			name:    "Rubbish values",
			fixture: []string{"--config", "/nonexistant/nonexistant.yaml"},
			expect:  E{msg: "fatal: Unable to read config: `/nonexistant/nonexistant.yaml`"},
		},
		{
			name:    "invalid loglevel",
			fixture: []string{"--config", confpath, "--loglevel", "invalid"},
			expect:  E{msg: "error: Ignoring --loglevel"},
		},
		{
			name:    "invalid key",
			fixture: []string{"--config", confpath, "--loglevel", "debug", "--agekey", "/non/existant/age.key"},
			expect:  E{msg: "fatal: Failed to open age key: `/non/existant/age.key`"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			hooker := func(e *zerolog.Event, level zerolog.Level, msg string) {
				switch level {
				case zerolog.ErrorLevel, zerolog.FatalLevel:
					monkey.Patch(os.Exit, func(int) {
						panic(level.String() + ": " + msg)
					})
					panic(level.String() + ": " + msg)
				}
			}
			log.Logger = log.Logger.Hook(MockHook{HookFunc: hooker})
			monkey.Patch(pflag.Parse, func() {
				pflag.CommandLine.Parse(test.fixture)
			})

			defer monkey.UnpatchAll()
			if test.expect.msg != "" {
				assert.PanicsWithValue(tt, test.expect.msg, func() { LoadConf() })
				//assert.Equal(tt, test.expect.panic, fatal())
			} else {
				assert.NotPanics(tt, func() { LoadConf() })
			}
		})

	}

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
		err string
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
			expect:  E{"unable to load settings"},
		},
		{
			name:    "Exist store",
			fixture: F{config, "Exist store", true, nil},
			expect:  E{},
		},
		{
			name:    "HTTP error",
			fixture: F{config, "HTTP error", false, &H{err: errors.New("something went wrong")}},
			expect:  E{"failed to make request"},
		},
		{
			name:    "Auth error",
			fixture: F{config, "HTTP error", false, &H{code: 401, body: []byte("")}},
			expect:  E{"authentication failure"},
		},
		{
			name:    "XML error",
			fixture: F{config, "Other error", false, &H{code: 400, body: []byte(`<TwilioResponse><RestException><Code>30000</Code><Detail></Detail><Message>Fake Error</Message><MoreInfo>fake.com</MoreInfo><Status>400</Status></RestException></TwilioResponse>`)}},
			expect:  E{"Fake Error"},
		},
		{
			name:    "Unmarshalble error",
			fixture: F{config, "Unmarshalble error", false, &H{code: 400, body: []byte(`<<Tw~!!`)}},
			expect:  E{"failed to read responses"},
		},
		{
			name:    "Other error",
			fixture: F{config, "Other error", false, &H{code: 500, body: []byte("Server Error")}},
			expect:  E{"twilio responded with failure"},
		},
		{
			name:    "Happy",
			fixture: F{config, "Happy path", false, &H{code: 201, body: []byte("OK")}},
			expect:  E{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			defer monkey.UnpatchAll()
			viper.SetConfigType("yaml")
			viper.ReadConfig(bytes.NewBuffer([]byte(test.fixture.config)))
			skvStore = &skv.KVStore{}
			httpClient = &http.Client{Transport: MockRoundTripper(func(req *http.Request) (res *http.Response, err error) {
				return &http.Response{StatusCode: test.fixture.twillio.code, Body: io.NopCloser(bytes.NewBuffer(test.fixture.twillio.body))}, test.fixture.twillio.err
			})}
			monkey.Patch(decodeAge, func(s string, a *age.X25519Identity) string { return s })
			monkey.PatchInstanceMethod(reflect.TypeOf((*skv.KVStore)(nil)), "Get", func(*skv.KVStore, string, interface{}) error {
				if test.fixture.store {
					return nil
				}
				return errors.New("")
			})
			monkey.PatchInstanceMethod(reflect.TypeOf((*skv.KVStore)(nil)), "Put", func(*skv.KVStore, string, interface{}) error { return nil })

			err := Notify(test.fixture.message)
			if test.expect.err == "" {
				assert.Nil(tt, err)
			} else {
				assert.Error(tt, err)
				assert.EqualError(tt, err, test.expect.err)
			}
		})
	}
}

func TestQueryBackend(t *testing.T) {
	type H struct {
		body   string
		status int
		err    error
	}
	type E struct {
		result *APIResponse
		err    string
	}
	test_data := []struct {
		name     string
		http     H
		expected E
	}{
		{
			"HTTP Err",
			H{err: errors.New("http error")},
			E{err: "Get \"fake.com?param=value\": http error"},
		},
		{
			"HTTP !200",
			H{status: 400, body: "Something went wrong"},
			E{err: "something went wrong"},
		},
		{
			"Invalid JSON",
			H{status: 200, body: "{Non_'Json"},
			E{err: "Failed not parse result"},
		},
		{
			"Happy",
			H{status: 200, body: `{"data":[{"id":1,"description":"test","amount":1.0,"account":1234567890,"created":"2000-01-01T00:00:00+11:00"}]}`},
			E{result: &APIResponse{Data: []APITransaction{{Id: 1, Description: "test", Amount: 1, Account: 1234567890, Created: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.Local)}}}},
		},
	}

	for _, test := range test_data {
		t.Run(test.name, func(tt *testing.T) {
			viper.Set("backend", "fake.com")
			httpClient = &http.Client{Transport: MockRoundTripper(func(req *http.Request) (res *http.Response, err error) {
				if test.http.body != "" {
					res = &http.Response{Body: io.NopCloser(bytes.NewBufferString(test.http.body)), StatusCode: test.http.status}
				}
				err = test.http.err
				return
			})}

			result, err := QueryBackend(map[string]string{"param": "value"})
			if test.expected.err != "" {
				if assert.NotNil(tt, err) {
					assert.Equal(tt, test.expected.err, err.Error())
				}
			}
			if test.expected.result != nil {
				if assert.NotNil(tt, result) {
					assert.Equal(tt, *test.expected.result, result)
				}
			}
		})
	}
}

func TestRunCheckAmount(t *testing.T) {
	type E struct {
		result string
		err    string
		params map[string]string
	}

	type Q struct {
		err    error
		result APIResponse
	}

	type F struct {
		args  Amount
		query Q
	}

	test_data := []struct {
		name     string
		fixture  F
		expected E
	}{
		{
			"Basic",
			F{args: Amount{Match: "Basic"}},
			E{
				params: map[string]string{"amount__ne": "0.00", "created__gt": "2000-01-01T00:00:00", "description__like": "Basic"},
			},
		},
		{
			"Invalid Rrule",
			F{args: Amount{Rrule: "xs#$"}},
			E{
				err: "rrule invalid",
			},
		},
		{
			"RRule",
			F{args: Amount{Match: "RRule", Rrule: "FREQ=WEEKLY;DTSTART=19991201T000000Z"}},
			E{
				params: map[string]string{"amount__ne": "0.00", "created__gt": "1999-12-29T00:00:00", "description__like": "RRule"},
			},
		},
		{
			"Days",
			F{args: Amount{Match: "Days", Days: 3}},
			E{
				params: map[string]string{"amount__ne": "0.00", "created__gt": "1999-12-29T00:00:00", "description__like": "Days"},
			},
		},
		{
			"Threshold",
			F{args: Amount{Match: "Threshold", Threshold: "10", Expected: 100.0}},
			E{
				params: map[string]string{"amount__gt": "90.00", "amount__lt": "110.00", "created__gt": "2000-01-01T00:00:00", "description__like": "Threshold"},
			},
		},
		{
			"Threshold Percentage",
			F{args: Amount{Match: "Threshold", Threshold: "10%", Expected: 1000.0}},
			E{
				params: map[string]string{"amount__gt": "900.00", "amount__lt": "1100.00", "created__gt": "2000-01-01T00:00:00", "description__like": "Threshold"},
			},
		},
		{
			"QueryError",
			F{query: Q{err: errors.New("something went wrong")}},
			E{
				err: "something went wrong",
			},
		},
		{
			"Result",
			F{query: Q{result: APIResponse{Data: []APITransaction{{Amount: 1.0}}}}},
			E{
				result: "Unexpected amounts:\nMon 1 Jan  for $1.00 expecting $0.00",
			},
		},
	}

	for _, test := range test_data {
		t.Run(test.name, func(tt *testing.T) {
			defer monkey.UnpatchAll()
			var called map[string]string
			monkey.Patch(QueryBackend, func(p map[string]string) (a APIResponse, err error) {
				called = map[string]string{}
				for k, v := range p {
					called[k] = v
				}
				a = test.fixture.query.result
				err = test.fixture.query.err
				return
			})
			monkey.Patch(time.Now, func() time.Time {
				return time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
			})

			result, err := CheckAmount(test.fixture.args)
			assert.Equal(tt, test.expected.result, result)
			if test.expected.params != nil {
				if assert.NotNil(tt, called) {
					assert.Equal(tt, test.expected.params, called)
				}
			}
			if test.expected.err == "" {
				assert.Nil(tt, err)
			} else {
				if assert.NotNil(tt, err) {
					assert.Equal(tt, test.expected.err, err.Error())
				}
			}
		})
	}
}

func TestRunChecks(t *testing.T) {
	type R struct {
		result string
		err    error
	}
	type F struct {
		config      string
		checkamount R
		notify      R
	}
	type E struct {
		err         string
		checkamount Amount
		notify      string
	}

	configs := map[string]string{
		"empty": "",
		"amount": `checks:
  - type: amount
    name: test
    expected: 65.00
    threshold: 20%
    match: pineapple
    days: 3`,
		"invalid": `checks:
  - type: invalid`,
		"error": "*}}--ss",
	}

	tests := []struct {
		name    string
		fixture F
		expect  E
	}{
		{
			name:    "EmptyConfig",
			fixture: F{config: configs["empty"]},
			expect:  E{},
		},
		{
			name: "ErrorAmount",
			fixture: F{
				config:      configs["amount"],
				checkamount: R{err: errors.New("something failed")},
			},
			expect: E{
				err:         "something failed",
				checkamount: Amount{Name: "test", Match: "pineapple", Days: 3, Expected: 65, Threshold: "20%", Rrule: ""},
			},
		},
		{
			name: "InvalidType",
			fixture: F{
				config: configs["invalid"],
			},
			expect: E{err: "Invalid check type: invalid"},
		},
		{
			name: "NotifyError",
			fixture: F{
				config:      configs["amount"],
				checkamount: R{result: "something OK"},
				notify:      R{err: errors.New("notify failed")},
			},
			expect: E{
				checkamount: Amount{Name: "test", Match: "pineapple", Days: 3, Expected: 65, Threshold: "20%", Rrule: ""},
				notify:      "something OK",
				err:         "notify failed",
			},
		},
		{
			name: "OK",
			fixture: F{
				config:      configs["amount"],
				checkamount: R{result: "something OK"},
			},
			expect: E{
				checkamount: Amount{Name: "test", Match: "pineapple", Days: 3, Expected: 65, Threshold: "20%", Rrule: ""},
				notify:      "something OK",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(st *testing.T) {
			var (
				checkamount_params Amount
				notify_params      string
			)
			viper.SetConfigType("yaml")
			viper.ReadConfig(bytes.NewBufferString(test.fixture.config))

			defer monkey.UnpatchAll()
			monkey.Patch(CheckAmount, func(p Amount) (msg string, err error) {
				checkamount_params = Amount(p)
				return test.fixture.checkamount.result, test.fixture.checkamount.err
			})
			monkey.Patch(Notify, func(message string) (err error) {
				notify_params = message
				return test.fixture.notify.err
			})

			err := RunChecks()

			if test.expect.err == "" {
				assert.Nil(st, err)
			} else {
				assert.NotNil(st, err)
				if err != nil {
					assert.Equal(st, test.expect.err, err.Error(), "Resulting error")
				}
			}
			assert.Equal(st, test.expect.checkamount, checkamount_params, "CheckAmount parameters")
			assert.Equal(st, test.expect.notify, notify_params, "Notify paramaters")
		})
	}
}

func TestAgeFunctions(t *testing.T) {
	key := []byte(`# created: 2022-12-14T09:40:56+11:00
# public key: age176vjsyk43d46tf5hcttcxe4kumzjxw4zrncmvemuxnt0kfvv7ffq4u7mlw
AGE-SECRET-KEY-1UNDK53VGXQXXY6KNF7D865UW7Y4ADEJTRUPMEX9499AWCTSELMQSNCJU8P`)
	a, err := loadAgeKey(key)
	assert.Nil(t, err)
	assert.NotNil(t, a)
	assert.Equal(t, "age176vjsyk43d46tf5hcttcxe4kumzjxw4zrncmvemuxnt0kfvv7ffq4u7mlw", a.Recipient().String())

	msg := `age:YWdlLWVuY3J5cHRpb24ub3JnL3YxCi0+IFgyNTUxOSBKU2w4RmV5bndqMTJ3YW9oZi8rMm0xdjJ1N096VEVpekFsVzJaK2trM3kwCmtoMWQ1VElQUEovcWF3TnVSeXQ3b3l0WnZIOUdyZXU1aFl5WjBEK1pXdW8KLS0tIEdUcG0zUjNYWW5icGlPRFpITVNKcXY5MkVlZGdxR2VBTVNtMTI5Q0lvbGcK3TgMVBzvlx2f2/8xzmgW04VL1P83UyYdrODKq3TLinRgyTLFNd1xI08Sv0hEyio=`
	dec := decodeAge(msg, a)
	assert.Equal(t, "testing message", dec)
}

// func TestCheckRepay(t *testing.T) {
// 	monkey.Patch(time.Now, func() time.Time {
// 		return time.Date(2000, time.January, 4, 0, 0, 0, 0, time.UTC)
// 	})
// 	defer monkey.UnpatchAll()

// 	type A struct {
// 		match string
// 		from  string
// 		to    string
// 		days  int
// 	}

// 	type E struct {
// 		err_message string
// 		calls       int
// 	}

// 	test_data := []struct {
// 		name            string
// 		args            A
// 		expected        string
// 		expected_params []map[string]string
// 		err             *E
// 	}{
// 		{
// 			"Empty",
// 			F{args: Amount{"NOTFOUND", "NotFound", "NotFound", -3},
// 			"",
// 			[]map[string]string{{"amount__lt": "0.00", "created__gt": "2000-01-07T00:00:00", "description__like": "NOTFOUND"}},
// 			nil,
// 		},
// 		{
// 			"Matches",
// 			F{args: Amount{"WOOLWORTHS", "Food", "Spending", 3},
// 			"Move money from Food to Spending:\n$2.22 from Sun 2 Jan\n$3.33 from Mon 3 Jan",
// 			[]map[string]string{
// 				{"amount__lt": "0.00", "created__gt": "2000-01-01T00:00:00", "description__like": "WOOLWORTHS"},
// 				{"amount": "1.11", "created__gt": "2000-01-01T00:00:01", "description__like": "Food"},
// 				{"amount": "2.22", "created__gt": "2000-01-02T00:00:01", "description__like": "Food"},
// 				{"amount": "3.33", "created__gt": "2000-01-03T00:00:01", "description__like": "Food"},
// 			},
// 			nil,
// 		},
// 		{
// 			"SingleMatch",
// 			F{args: Amount{"WOOLWORTHS", "Food", "Spending", 1},
// 			"Move money from Food to Spending:\n$3.33 from Mon 3 Jan",
// 			[]map[string]string{
// 				{"amount__lt": "0.00", "created__gt": "2000-01-03T00:00:00", "description__like": "WOOLWORTHS"},
// 				{"amount": "3.33", "created__gt": "2000-01-03T00:00:01", "description__like": "Food"},
// 			},
// 			nil,
// 		},
// 		{
// 			"ErrInitial",
// 			F{args: Amount{"WOOLWORTHS", "Food", "Spending", 3},
// 			"",
// 			[]map[string]string{
// 				{"amount__lt": "0.00", "created__gt": "2000-01-01T00:00:00", "description__like": "WOOLWORTHS"},
// 			},
// 			&E{"Failure", 0},
// 		},
// 		{
// 			"ErrSubsequent",
// 			F{args: Amount{"WOOLWORTHS", "Food", "Spending", 3},
// 			"",
// 			[]map[string]string{
// 				{"amount__lt": "0.00", "created__gt": "2000-01-01T00:00:00", "description__like": "WOOLWORTHS"},
// 				{"amount": "1.11", "created__gt": "2000-01-01T00:00:01", "description__like": "Food"},
// 			},
// 			&E{"Failure", 1},
// 		},
// 	}

// 	for _, td := range test_data {
// 		t.Run(td.name, func(s *testing.T) {
// 			var called []map[string]string
// 			monkey.Patch(QueryBackend, func(p map[string]string) (a APIResponse, err error) {
// 				called = append(called, p)
// 				if td.err != nil && len(called) > td.err.calls {
// 					err = errors.New(td.err.err_message)
// 					return a, err
// 				}
// 				a.Data = FilterTransactions(p)
// 				return
// 			})
// 			defer monkey.Unpatch(QueryBackend)

// 			result, err := CheckRepay(Repay{Match: td.args.match, Days: td.args.days, From: td.args.from, To: td.args.to})
// 			if td.err == nil {
// 				assert.Nil(s, err)
// 				assert.Equal(s, td.expected, result)
// 				assert.Equal(s, td.expected_params, called, "Incorrect QueryBackend params")
// 				return
// 			}
// 			assert.NotNil(s, err)
// 			assert.Equal(s, td.err.err_message, err.Error())
// 			assert.Equal(s, td.expected_params, called, "Incorrect QueryBackend params")
// 		})
// 	}
// }
