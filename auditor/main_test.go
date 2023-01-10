package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/codingric/moneyman/pkg/notify"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rzajac/zltest"
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
		err   string
		log   string
		level zerolog.Level
	}
	tests := []struct {
		name    string
		fixture []string
		expect  E
	}{
		{
			name:    "Rubbish values",
			fixture: []string{"-c", "/nonexistant/nonexistant.yaml"},
			expect:  E{err: "unable to load config: `/nonexistant/nonexistant.yaml`"},
		},
		{
			name:    "invalid loglevel",
			fixture: []string{"-c", confpath, "-l", "invalid"},
			expect:  E{log: "Ignoring --loglevel", err: "unable to load agekey: `/etc/auditor/age.key`"},
		},
		{
			name:    "invalid key",
			fixture: []string{"-c", confpath, "-l", "debug", "-a", "/non/existant/age.key"},
			expect:  E{err: "unable to load agekey: `/non/existant/age.key`"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			tst := zltest.New(tt)
			log.Logger = zerolog.New(tst).With().Timestamp().Logger()

			monkey.Patch(pflag.Parse, func() {
				pflag.CommandLine.Parse(test.fixture)
			})

			defer monkey.UnpatchAll()
			err := LoadConf(context.Background())
			if test.expect.err != "" {
				if assert.Error(tt, err) {
					assert.Equal(tt, test.expect.err, err.Error())
				}
			} else {
				assert.Nil(tt, err)
			}
			if test.expect.log != "" {
				ent := tst.Entries()
				ent.ExpMsg(test.expect.log)
				//ent.ExpLevel(test.expect.level)
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
			E{err: "failed not parse result"},
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

			result, err := QueryBackend(map[string]string{"param": "value"}, context.Background())
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
			"RRule overdue",
			F{args: Amount{Name: "Overdue", Match: "RRule", Rrule: "FREQ=WEEKLY;DTSTART=19991228T000000Z", Days: 7, Expected: 11.11}},
			E{
				params: map[string]string{"amount__ne": "11.11", "created__gt": "1999-12-21T00:00:00", "description__like": "RRule"},
				result: "Payment for Overdue ($11.11) overdue 4 days",
			},
		},
		{
			"RRule out of range",
			F{args: Amount{Match: "RRule", Rrule: "FREQ=WEEKLY;DTSTART=19991201T000000Z"}},
			E{},
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
			monkey.Patch(QueryBackend, func(p map[string]string, c context.Context) (a APIResponse, err error) {
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

			result, err := CheckAmount(test.fixture.args, context.Background())
			assert.Equal(tt, test.expected.result, result)
			if test.expected.params != nil {
				if assert.NotNil(tt, called, "Expected calls to Backend") {
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
	type N struct {
		sent int
		err  error
	}
	type F struct {
		config      string
		checkamount R
		notify      N
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
				notify:      N{err: errors.New("notify failed")},
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
			monkey.Patch(CheckAmount, func(p Amount, c context.Context) (msg string, err error) {
				checkamount_params = Amount(p)
				return test.fixture.checkamount.result, test.fixture.checkamount.err
			})
			monkey.Patch(notify.Notify, func(message string, c context.Context) (sent int, err error) {
				notify_params = message
				return test.fixture.notify.sent, test.fixture.notify.err
			})

			err := RunChecks(context.Background())

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
