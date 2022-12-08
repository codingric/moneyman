package bigbills

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/sheets/v4"
)

var test_config_yaml = []byte(`---
sid: 00000000000
token: 00000000000
spreadsheet_id: testspreadsheetid
spreadsheet_range: "Big Bills!M2:P"
account_id: "000000"
backend: https://fake.com/api/transactions
mobiles:
  - "+6140000000"
credentials: |
  {
      "type": "service_account",
      "project_id": "project-1234567890123",
      "private_key_id": "radacted",
      "private_key": "-----BEGIN PRIVATE KEY-----\nP01crPCOMbZ4krHR4qmrpTAAuiVKguaQ7zxRBitExfRj4kHSJ6pDgml4UTdpBQ+A\nYVWxjwIDAQABAkEAlbQZJc83FsH4Fut356jEmC1EFXpYyfv5mgPBz4YyD0JfTz/L\noMEQZExt2dNLLSftCUCEEhTgVn9KhNE7AbkHwQIhAPMV6EKa3MEjbnQ50hz3zMWR\nmtmqdyvlIAzEAN/3nv1hAiEA2+Xt14J9UIOtI5bYMNkEQGukXl8f+vksBTHlVbS1\npO8CIHFY8r8z9OG+Qr/BQl8tkPdDCMCcQwtdwI8TShElJqahAiBBtRf34K8PYvlW\nfBmHpyFFGqUX6fbFaLVFrB4qGQB6EwIhAJ/Kib37tH2BumOPxu+gP/KJsIOAKdXX\nUN0yFBuXahyi\n-----END PRIVATE KEY-----",
      "client_email": "sauser@project-1234567890123.iam.gserviceaccount.com",
      "client_id": "0000000000000000000",
      "auth_uri": "https://accounts.google.com/o/oauth2/auth",
      "token_uri": "https://oauth2.googleapis.com/token",
      "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
      "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/sauser%40project-1234567890123.iam.gserviceaccount.com"
  }
`)

type MockRoundTripper func(req *http.Request) (res *http.Response, err error)

func (m MockRoundTripper) RoundTrip(req *http.Request) (res *http.Response, err error) {
	return m(req)
}

func LoadTestConfig() {
	settings = &Settings{
		SpreadsheetId:    "longid",
		SpreadsheetRange: "Tab!A2:B",
		AccountId:        "000000",
		Credentials:      "{}",
		Transactions:     "http://fake.com/api/transactions",
	}
}

func TestMain(m *testing.M) {
	LoadTestConfig()
	//log.Logger = zerolog.New(&bytes.Buffer{}).With().Logger()
	sheetsService = &sheets.Service{
		Spreadsheets: &sheets.SpreadsheetsService{
			Values: &sheets.SpreadsheetsValuesService{},
		},
	}
	retCode := m.Run()
	os.Exit(retCode)
}

func Test_findcell(t *testing.T) {
	assert.Equal(t, "A2:B2", findcell("A1:B", 1, true))
	assert.Equal(t, "A4", findcell("A2", 2, true))
	assert.Equal(t, "A2", findcell("A1", 1, false))
	assert.Equal(t, "A3:B3", findcell("A1:B100", 2, true))
}

func TestBigBillsDateCheckRepayments(t *testing.T) {
	LoadTestConfig()
	n := time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)
	p := time.Date(2000, 1, 2, 0, 0, 0, 0, time.Local)

	b := BigBillDate{Date: n, Amount: 100}
	update_called := false
	monkey.PatchInstanceMethod(reflect.TypeOf((*BigBillDate)(nil)), "UpdatePaid", func(o *BigBillDate, tt time.Time) error {
		update_called = true
		assert.Equal(t, p, tt)
		return nil
	})
	httpClient = &http.Client{Transport: MockRoundTripper(func(req *http.Request) (r *http.Response, e error) {
		assert.Equal(t, "/api/transactions?amount=-100.00&account=000000&created__gt=2000-01-01T00:00:00", req.URL.RequestURI())
		r = &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Body:       io.NopCloser(bytes.NewBufferString(`{"data":[{"id":1,"description":"","amount":-100.00,"account":37366510,"created":"2000-01-02T00:00:00+11:00"}]}`)),
		}
		return
	})}
	paid, err := b.CheckRepayments()
	assert.Equal(t, true, paid)
	assert.Nil(t, err)
	assert.True(t, update_called, "UpdatePaid not called")

	httpClient = &http.Client{Transport: MockRoundTripper(func(req *http.Request) (r *http.Response, e error) {
		assert.Equal(t, "/api/transactions?amount=-100.00&account=000000&created__gt=2000-01-01T00:00:00", req.URL.RequestURI())
		r = &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Body:       io.NopCloser(bytes.NewBufferString(`{"data":null}`)),
		}
		return
	})}
	paid, err = b.CheckRepayments()
	assert.Equal(t, false, paid)
	assert.Nil(t, err)
	assert.True(t, update_called, "UpdatePaid not called")
	monkey.UnpatchAll()
}

func TestBigBillsDateUpdatePaid(t *testing.T) {
	LoadTestConfig()
	n := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	b := BigBillDate{}

	expected := sheets.ValueRange{MajorDimension: "", Range: "", Values: [][]interface{}{{"01/01/2000"}}}
	monkey.PatchInstanceMethod(reflect.TypeOf((*sheets.SpreadsheetsValuesService)(nil)), "Update", func(s *sheets.SpreadsheetsValuesService, spreadsheetId string, range_ string, valuerange *sheets.ValueRange) (o *sheets.SpreadsheetsValuesUpdateCall) {
		assert.Equal(t, "longid", spreadsheetId)
		assert.Equal(t, "Tab!A2:B", range_)
		assert.Equal(t, &expected, valuerange)
		return &sheets.SpreadsheetsValuesUpdateCall{}
	})
	monkey.PatchInstanceMethod(reflect.TypeOf((*sheets.SpreadsheetsValuesUpdateCall)(nil)), "ValueInputOption", func(s *sheets.SpreadsheetsValuesUpdateCall, valueInputOption string) *sheets.SpreadsheetsValuesUpdateCall {
		return s
	})
	monkey.PatchInstanceMethod(reflect.TypeOf((*sheets.SpreadsheetsValuesUpdateCall)(nil)), "Do", func(s *sheets.SpreadsheetsValuesUpdateCall, opts ...googleapi.CallOption) (*sheets.UpdateValuesResponse, error) {
		return nil, nil
	})
	defer monkey.UnpatchAll()

	b.UpdatePaid(n)
}

func TestBigBillsGetLate(t *testing.T) {
	past := time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC)
	now := time.Date(2000, time.January, 4, 0, 0, 0, 0, time.UTC)
	future := time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC)
	monkey.Patch(time.Now, func() time.Time {
		return now
	})
	monkey.PatchInstanceMethod(reflect.TypeOf((*BigBillDate)(nil)), "CheckRepayments", func(*BigBillDate) (paid bool, err error) {
		return false, nil
	})
	defer monkey.UnpatchAll()
	type E struct {
		result []LateBigBill
		err    string
	}
	tests := []struct {
		name   string
		setup  BigBills
		expect E
	}{
		{
			"Empty",
			BigBills{},
			E{},
		},
		{
			"Paid",
			BigBills{[]BigBillDate{{past, 0, &recent, 0}}},
			E{},
		},
		{
			"Future",
			BigBills{[]BigBillDate{{future, 0, nil, 0}}},
			E{},
		},
		{
			"Unpaid",
			BigBills{[]BigBillDate{{past, 0, nil, 0}}},
			E{[]LateBigBill{{Date: past, Amount: 0, Days: 2}}, ""},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(st *testing.T) {

			result, err := test.setup.GetLate()
			if test.expect.err == "" {
				assert.Nil(st, err)
			} else {
				assert.Equal(st, test.expect.err, err.Error())
			}

			assert.Equal(st, test.expect.result, result)
		})
	}
}

func TestBigBillsHydrate(t *testing.T) {
	past := time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC)
	type P struct {
		result *sheets.ValueRange
		err    error
	}
	type E struct {
		result BigBills
		err    string
	}
	tests := []struct {
		name   string
		patch  P
		expect E
	}{
		{
			"Empty",
			P{&sheets.ValueRange{}, nil},
			E{BigBills{}, ""},
		},
		{
			"Err",
			P{&sheets.ValueRange{}, errors.New("Something went wrong")},
			E{BigBills{}, "Something went wrong"},
		},
		{
			"Unpaid",
			P{&sheets.ValueRange{Values: [][]interface{}{{"2000-01-02", "100.00", "-100.00"}}}, nil},
			E{BigBills{[]BigBillDate{{past, 100.00, nil, 0}}}, ""},
		},
		{
			"Paid",
			P{&sheets.ValueRange{Values: [][]interface{}{{"2000-01-02", "100.00", "-100.00", "2000-01-03"}}}, nil},
			E{BigBills{[]BigBillDate{{past, 100.00, &recent, 0}}}, ""},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(st *testing.T) {
			monkey.PatchInstanceMethod(
				reflect.TypeOf((*sheets.SpreadsheetsValuesGetCall)(nil)),
				"Do",
				func(s *sheets.SpreadsheetsValuesGetCall, opts ...googleapi.CallOption) (*sheets.ValueRange, error) {
					return test.patch.result, test.patch.err
				})
			defer monkey.UnpatchAll()

			var bills BigBills
			err := bills.Hydrate()
			if test.expect.err == "" {
				assert.Nil(st, err)
			} else {
				assert.Equal(st, test.expect.err, err.Error())
			}

			assert.Equal(st, test.expect.result, bills)
		})
	}
}