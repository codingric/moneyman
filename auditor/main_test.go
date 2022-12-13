package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"bou.ke/monkey"
	"filippo.io/age"
	"github.com/jarcoal/httpmock"
	"github.com/rapidloop/skv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var success_resp = `{
	"account_sid": "ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
	"api_version": "2010-04-01",
	"body": "This is the ship that made the Kessel Run in fourteen parsecs?",
	"date_created": "Thu, 30 Jul 2015 20:12:31 +0000",
	"date_sent": "Thu, 30 Jul 2015 20:12:33 +0000",
	"date_updated": "Thu, 30 Jul 2015 20:12:33 +0000",
	"direction": "outbound-api",
	"error_code": null,
	"error_message": null,
	"from": "+15017122661",
	"messaging_service_sid": "MGXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
	"num_media": "0",
	"num_segments": "1",
	"price": null,
	"price_unit": null,
	"sid": "SMXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
	"status": "sent",
	"subresource_uris": {
	  "media": "/2010-04-01/Accounts/ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX/Messages/SMXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX/Media.json"
	},
	"to": "+15558675310",
	"uri": "/2010-04-01/Accounts/ACXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX/Messages/SMXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX.json"
  }`

func GetParams(target string) map[string]string {
	matches := make(map[string]string)
	re := regexp.MustCompile(`(?P<key>[\w]+)=(?P<value>[^\s&]+)`)
	m := re.FindAllStringSubmatch(target, -1)
	for _, k := range m {
		matches[k[1]] = k[2]
	}
	return matches
}

var config = []byte(`checks:
  - type: repay
    days: 3
    match: WOOLWORTHS
    from: Food
    to: Spending
  - type: amount
    expected: $65.00
    threshold: 20%
    match: pineapple
backend: https://api.mock/api/transactions
notify:
  sid: mocksid
  token: mocktoken
  mobiles:
    - +61412345678`)

type MockRoundTripper func(req *http.Request) (res *http.Response, err error)

func (m MockRoundTripper) RoundTrip(req *http.Request) (res *http.Response, err error) {
	return m(req)
}

func SetUpConfig() {

	viper.SetConfigType("yaml") // or viper.SetConfigType("YAML")

	viper.ReadConfig(bytes.NewBuffer(config))
	viper.RegisterAlias("verbose", "v")
	viper.Set("verbose", false)

	log.Logger.Level(zerolog.FatalLevel)
}

var (
	Transactions = []APITransaction{
		{Id: 1, Description: "WOOLWORTHS", Amount: -1.11, Account: 1234, Created: time.Date(2000, time.January, 1, 0, 0, 1, 0, time.UTC)},
		{Id: 2, Description: "WOOLWORTHS", Amount: -2.22, Account: 1234, Created: time.Date(2000, time.January, 2, 0, 0, 1, 0, time.UTC)},
		{Id: 3, Description: "WOOLWORTHS", Amount: -3.33, Account: 1234, Created: time.Date(2000, time.January, 3, 0, 0, 1, 0, time.UTC)},
		{Id: 4, Description: "Bill", Amount: -60.00, Account: 1234, Created: time.Date(2000, time.January, 3, 10, 0, 1, 0, time.UTC)},
		{Id: 5, Description: "Salary", Amount: 800.00, Account: 1234, Created: time.Date(2000, time.January, 3, 14, 0, 1, 0, time.UTC)},
		{Id: 6, Description: "Deposit from Food to Spending", Amount: 1.11, Account: 1234, Created: time.Date(2000, time.January, 4, 0, 0, 1, 0, time.UTC)},
	}
)

func FilterTransactions(m map[string]string) []APITransaction {
	f := []APITransaction{}
	for _, i := range Transactions {
		if a, err := strconv.ParseFloat(m["amount"], 32); err == nil && i.Amount != float32(a) {
			continue
		}
		if a, err := strconv.ParseFloat(m["amount__lt"], 32); err == nil && i.Amount >= float32(a) {
			continue
		}
		if a, err := strconv.ParseFloat(m["amount__gt"], 32); err == nil && i.Amount <= float32(a) {
			continue
		}
		if a, err := strconv.ParseFloat(m["amount__ne"], 32); err == nil && i.Amount == float32(a) {
			continue
		}
		if id, err := strconv.ParseInt(m["id"], 10, 64); err == nil && i.Id != id {
			continue
		}
		if c, err := time.Parse("2006-01-02T15:04:05", m["created__gt"]); err == nil && i.Created.Unix() <= c.Unix() {
			continue
		}
		if c, err := time.Parse("2006-01-02T15:04:05", m["created__lt"]); err == nil && i.Created.Unix() >= c.Unix() {
			continue
		}
		if m["description__like"] != "" && !strings.Contains(i.Description, m["description__like"]) {
			continue
		}
		f = append(f, i)
	}
	return f
}

func MockedAPIResponder(req *http.Request) (*http.Response, error) {
	url_, _ := url.QueryUnescape(req.URL.RequestURI())
	m := GetParams(url_)
	data := APIResponse{Data: FilterTransactions(m)}
	resp, _ := httpmock.NewJsonResponse(200, data)
	return resp, nil
}

func SetMockFailureTransactions() {
	httpmock.RegisterResponder("GET", "=~^https://api.mock/api/",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(503, "Service Error")
			return resp, nil
		})
}

func SetMockTwilio() {
	httpmock.RegisterResponder("POST", "https://api.twilio.com/2010-04-01/Accounts/mocksid/Messages",
		func(req *http.Request) (*http.Response, error) {
			v, _ := httputil.DumpRequest(req, true)
			req_body := strings.Split(string(v), "\r\n")
			if req_body[2] != "Authorization: Basic bW9ja3NpZDptb2NrdG9rZW4=" {
				resp := httpmock.NewStringResponse(401, `<TwilioResponse><RestException><Code>20003</Code><Detail></Detail><Message>Authenticate</Message><MoreInfo>https://www.twilio.com/docs/errors/20003</MoreInfo><Status>401</Status></RestException></TwilioResponse>`)
				return resp, nil
			}
			if req_body[6] != "Body=mockmessage&From=Budget&To=61412345678" {
				resp := httpmock.NewStringResponse(400, `<TwilioResponse><RestException><Code>30000</Code><Detail></Detail><Message>Fake Error</Message><MoreInfo>fake.com</MoreInfo><Status>400</Status></RestException></TwilioResponse>`)
				return resp, nil
			}
			resp := httpmock.NewStringResponse(201, success_resp)
			return resp, nil
		})
}

func TestMain(m *testing.M) {
	SetUpConfig()
	retCode := m.Run()
	os.Exit(retCode)
}

func TestNotify(t *testing.T) {
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

func TestRunChecks(t *testing.T) {
	type CRP struct {
		match string
		from  string
		to    string
		days  int
	}
	type CRR struct {
		msg string
		err error
	}
	type NP struct{ message string }
	type NR struct{ err error }
	type CAP struct {
		match     string
		expected  float64
		threshold string
		days      int
	}
	type CAR struct {
		msg string
		err error
	}

	configs := map[string]string{"empty": "", "replay": `checks:
  - type: repay
    days: 3
    match: WOOLWORTHS
    from: Food
    to: Spending`, "amount": `checks:
  - type: amount
    expected: 65.00
    threshold: 20%
    match: pineapple
    days: 3`, "invalid": `checks:
  - type: invalid`}

	S := func(v string) *string { return &v }

	tests := []struct {
		name   string
		config string
		err    *string
		crp    *CRP
		crr    *CRR
		cap    *CAP
		car    *CAR
		np     *NP
		nr     *NR
		crc    int
		cac    int
		nc     int
	}{
		{
			name:   "EmptyConfig",
			config: configs["empty"],
		},
		{
			name:   "ReplaySuccess",
			config: configs["replay"],
			crp:    &CRP{"WOOLWORTHS", "Food", "Spending", 3},
			crr:    &CRR{"CheckReplay", nil},
			np:     &NP{"CheckReplay"},
			crc:    1, nc: 1,
		},
		{
			name:   "ReplayFailure",
			config: configs["replay"],
			crr:    &CRR{"", errors.New("CheckReplay")},
			crc:    1,
			err:    S("CheckReplay"),
		},
		{
			name:   "AmountSuccess",
			config: configs["amount"],
			cap:    &CAP{"pineapple", 65.00, "20%", 3},
			cac:    1,
		},
		{
			name:   "AmountFailure",
			config: configs["amount"],
			car:    &CAR{"", errors.New("AmountFailure")},
			err:    S("AmountFailure"),
			cac:    1,
		},
		{
			name:   "Invalid",
			config: configs["invalid"],
			err:    S("Invalid check type: invalid"),
		},
		{
			name:   "NotifyFailure",
			config: configs["replay"],
			crr:    &CRR{"CheckReplay", nil},
			nr:     &NR{errors.New("NotifyFailure")},
			err:    S("NotifyFailure"),
			crc:    1, nc: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(st *testing.T) {
			var (
				checkreplay_called int
				notify_called      int
				checkamount_called int
			)

			viper.ReadConfig(bytes.NewBuffer([]byte(test.config)))

			monkey.Patch(CheckRepay, func(r Repay) (msg string, err error) {
				checkreplay_called++
				if test.crp != nil {
					params := CRP{r.Match, r.From, r.To, r.Days}
					assert.Equal(st, *test.crp, params, "CheckRepay params")
				}
				if test.crr != nil {
					msg = test.crr.msg
					err = test.crr.err
				}
				return
			})
			monkey.Patch(Notify, func(message string) (err error) {
				notify_called++
				if test.np != nil {
					params := NP{message}
					assert.Equal(st, *test.np, params, "Notify params")
				}

				if test.nr != nil {
					err = test.nr.err
				}
				return
			})
			monkey.Patch(CheckAmount, func(c Amount) (msg string, err error) {
				checkamount_called++
				if test.cap != nil {
					params := CAP{c.Match, c.Expected, c.Threshold, c.Days}
					assert.Equal(st, *test.cap, params, "CheckAmount params")
				}
				if test.car != nil {
					msg = test.car.msg
					err = test.car.err
				}
				return
			})
			defer monkey.UnpatchAll()
			err := RunChecks()

			if test.err == nil {
				assert.Nil(st, err)
			} else {
				assert.NotNil(st, err)
				if err != nil {
					assert.Equal(st, *test.err, err.Error())
				}
			}
			assert.Equal(st, test.crc, checkreplay_called, "checkreplay_called")
			assert.Equal(st, test.cac, checkamount_called, "checkamount_called")
			assert.Equal(st, test.nc, notify_called, "notify_called")
		})
	}
}

func MockAPIEmptyResponder(req *http.Request) (*http.Response, error) {
	resp, err := httpmock.NewJsonResponse(200, APIResponse{})
	return resp, err
}

func MockAPIServerErrorReponder(req *http.Request) (*http.Response, error) {
	return httpmock.NewStringResponse(503, "Server Error"), nil
}
func MockAPINonJsonReponder(req *http.Request) (*http.Response, error) {
	return httpmock.NewStringResponse(200, "Server Error"), nil
}

func TestQueryBackend(t *testing.T) {
	sample := APIResponse{[]APITransaction{Transactions[0]}}
	test_data := []struct {
		name          string
		arg           map[string]string
		expected      APIResponse
		error_message string
		handler       func(req *http.Request) (*http.Response, error)
	}{
		{"HappyPath", map[string]string{"id": "1"}, sample, "", MockedAPIResponder},
		{"HappyPathEmpty", nil, APIResponse{}, "", MockAPIEmptyResponder},
		{"ErrorNotOK", nil, APIResponse{}, `no responder found`, httpmock.ConnectionFailure},
		{"ErrorNot200", nil, APIResponse{}, `server error`, MockAPIServerErrorReponder},
		{"FailedScan", nil, APIResponse{}, `content type unsupported: `, MockAPINonJsonReponder},
	}

	for _, tt := range test_data {
		t.Run(tt.name, func(t *testing.T) {

			monkey.PatchInstanceMethod(
				reflect.TypeOf(http.DefaultClient),
				"Do",
				func(c *http.Client, req *http.Request) (*http.Response, error) {
					resp, err := tt.handler(req)
					return resp, err
				})
			defer monkey.UnpatchInstanceMethod(reflect.TypeOf(http.DefaultClient), "Do")

			result, err := QueryBackend(tt.arg)
			if tt.error_message == "" {
				assert.Nil(t, err)
				assert.Equal(t, tt.expected, result)
				return
			}
			assert.NotNil(t, err)
			assert.Equal(t, tt.error_message, err.Error())
		})
	}
}

func Contains(s []int, e int64) bool {
	for _, v := range s {
		if v == int(e) {
			return true
		}
	}
	return false
}

func TestRunCheckAmount(t *testing.T) {
	type A struct {
		match     string
		amount    float64
		threshold string
		days      int
	}

	type H map[string]string

	monkey.Patch(time.Now, func() time.Time {
		return time.Date(2000, time.January, 4, 0, 0, 0, 0, time.UTC)
	})
	defer monkey.UnpatchAll()

	test_data := []struct {
		name            string
		args            A
		expected        string
		expected_params map[string]string
		error_message   string
		query_repsonse  []int
	}{
		{
			"Negative",
			A{"B", -50.0, "10%", 5},
			"Unexpected amounts:\nMon 3 Jan Bill for $60.00 expecting $50.00",
			H{"amount__lt": "-55.00", "created__gt": "1999-12-30T00:00:00", "description__like": "B"},
			"",
			[]int{4},
		},
		{
			"Unmatched",
			A{"B", -50.0, "10%", 5},
			"",
			H{"amount__lt": "-55.00", "created__gt": "1999-12-30T00:00:00", "description__like": "B"},
			"",
			[]int{},
		},
		{
			"Positive",
			A{"Sal", 1000, "100", 5},
			"Unexpected amounts:\nMon 3 Jan Salary for $800.00 expecting $1000.00",
			H{"amount__lt": "900.00", "created__gt": "1999-12-30T00:00:00", "description__like": "Sal"},
			"",
			[]int{5},
		},
		{
			"Exact",
			A{"X", 1000, "", 5},
			"Unexpected amounts:\nMon 3 Jan Salary for $800.00 expecting $1000.00",
			H{"amount__ne": "1000.00", "created__gt": "1999-12-30T00:00:00", "description__like": "X"},
			"",
			[]int{5},
		},
		{
			"Err",
			A{"X", 1000, "", 5},
			"Unexpected amounts:\nMon 3 Jan Salary for $800.00 expecting $1000.00",
			H{"amount__ne": "1000.00", "created__gt": "1999-12-30T00:00:00", "description__like": "X"},
			"something",
			[]int{5},
		},
	}

	for _, td := range test_data {
		t.Run(td.name, func(s *testing.T) {
			var called map[string]string
			monkey.Patch(QueryBackend, func(p map[string]string) (a APIResponse, err error) {
				called = p
				if td.error_message != "" {
					err = errors.New(td.error_message)
					return
				}
				for _, x := range Transactions {
					if Contains(td.query_repsonse, x.Id) {
						a.Data = append(a.Data, x)
					}
				}
				return
			})
			defer monkey.Unpatch(QueryBackend)

			result, err := CheckAmount(Amount{Match: td.args.match, Days: td.args.days, Expected: td.args.amount, Threshold: td.args.threshold})
			if td.error_message == "" {
				assert.Nil(s, err)
				assert.Equal(s, td.expected, result)
				assert.Equal(s, td.expected_params, called)
				return
			}
			assert.NotNil(s, err)
			assert.Equal(s, td.error_message, err.Error())
		})
	}
}

func TestCheckRepay(t *testing.T) {
	monkey.Patch(time.Now, func() time.Time {
		return time.Date(2000, time.January, 4, 0, 0, 0, 0, time.UTC)
	})
	defer monkey.UnpatchAll()

	type A struct {
		match string
		from  string
		to    string
		days  int
	}

	type E struct {
		err_message string
		calls       int
	}

	test_data := []struct {
		name            string
		args            A
		expected        string
		expected_params []map[string]string
		err             *E
	}{
		{
			"Empty",
			A{"NOTFOUND", "NotFound", "NotFound", -3},
			"",
			[]map[string]string{{"amount__lt": "0.00", "created__gt": "2000-01-07T00:00:00", "description__like": "NOTFOUND"}},
			nil,
		},
		{
			"Matches",
			A{"WOOLWORTHS", "Food", "Spending", 3},
			"Move money from Food to Spending:\n$2.22 from Sun 2 Jan\n$3.33 from Mon 3 Jan",
			[]map[string]string{
				{"amount__lt": "0.00", "created__gt": "2000-01-01T00:00:00", "description__like": "WOOLWORTHS"},
				{"amount": "1.11", "created__gt": "2000-01-01T00:00:01", "description__like": "Food"},
				{"amount": "2.22", "created__gt": "2000-01-02T00:00:01", "description__like": "Food"},
				{"amount": "3.33", "created__gt": "2000-01-03T00:00:01", "description__like": "Food"},
			},
			nil,
		},
		{
			"SingleMatch",
			A{"WOOLWORTHS", "Food", "Spending", 1},
			"Move money from Food to Spending:\n$3.33 from Mon 3 Jan",
			[]map[string]string{
				{"amount__lt": "0.00", "created__gt": "2000-01-03T00:00:00", "description__like": "WOOLWORTHS"},
				{"amount": "3.33", "created__gt": "2000-01-03T00:00:01", "description__like": "Food"},
			},
			nil,
		},
		{
			"ErrInitial",
			A{"WOOLWORTHS", "Food", "Spending", 3},
			"",
			[]map[string]string{
				{"amount__lt": "0.00", "created__gt": "2000-01-01T00:00:00", "description__like": "WOOLWORTHS"},
			},
			&E{"Failure", 0},
		},
		{
			"ErrSubsequent",
			A{"WOOLWORTHS", "Food", "Spending", 3},
			"",
			[]map[string]string{
				{"amount__lt": "0.00", "created__gt": "2000-01-01T00:00:00", "description__like": "WOOLWORTHS"},
				{"amount": "1.11", "created__gt": "2000-01-01T00:00:01", "description__like": "Food"},
			},
			&E{"Failure", 1},
		},
	}

	for _, td := range test_data {
		t.Run(td.name, func(s *testing.T) {
			var called []map[string]string
			monkey.Patch(QueryBackend, func(p map[string]string) (a APIResponse, err error) {
				called = append(called, p)
				if td.err != nil && len(called) > td.err.calls {
					err = errors.New(td.err.err_message)
					return a, err
				}
				a.Data = FilterTransactions(p)
				return
			})
			defer monkey.Unpatch(QueryBackend)

			result, err := CheckRepay(Repay{Match: td.args.match, Days: td.args.days, From: td.args.from, To: td.args.to})
			if td.err == nil {
				assert.Nil(s, err)
				assert.Equal(s, td.expected, result)
				assert.Equal(s, td.expected_params, called, "Incorrect QueryBackend params")
				return
			}
			assert.NotNil(s, err)
			assert.Equal(s, td.err.err_message, err.Error())
			assert.Equal(s, td.expected_params, called, "Incorrect QueryBackend params")
		})
	}
}
