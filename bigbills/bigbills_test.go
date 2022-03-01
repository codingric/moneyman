package main

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
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
      "private_key_id": "4b4f01db58295bfed0435ca0ef82367b019a61e6",
      "private_key": "-----BEGIN PRIVATE KEY-----\nMIIBVQIBADANBgkqhkiG9w0BAQEFAASCAT8wggE7AgEAAkEA0M4SIaljOBievzj6\nP01crPCOMbZ4krHR4qmrpTAAuiVKguaQ7zxRBitExfRj4kHSJ6pDgml4UTdpBQ+A\nYVWxjwIDAQABAkEAlbQZJc83FsH4Fut356jEmC1EFXpYyfv5mgPBz4YyD0JfTz/L\noMEQZExt2dNLLSftCUCEEhTgVn9KhNE7AbkHwQIhAPMV6EKa3MEjbnQ50hz3zMWR\nmtmqdyvlIAzEAN/3nv1hAiEA2+Xt14J9UIOtI5bYMNkEQGukXl8f+vksBTHlVbS1\npO8CIHFY8r8z9OG+Qr/BQl8tkPdDCMCcQwtdwI8TShElJqahAiBBtRf34K8PYvlW\nfBmHpyFFGqUX6fbFaLVFrB4qGQB6EwIhAJ/Kib37tH2BumOPxu+gP/KJsIOAKdXX\nUN0yFBuXahyi\n-----END PRIVATE KEY-----",
      "client_email": "sauser@project-1234567890123.iam.gserviceaccount.com",
      "client_id": "0000000000000000000",
      "auth_uri": "https://accounts.google.com/o/oauth2/auth",
      "token_uri": "https://oauth2.googleapis.com/token",
      "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
      "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/sauser%40project-1234567890123.iam.gserviceaccount.com"
  }
`)

func LoadTestConfig() {
	viper.SetConfigType("yaml")
	viper.ReadConfig(bytes.NewBuffer(test_config_yaml))
	viper.Set("verbose", true)
	//viper.Debug()
}

// func TestMain(m *testing.M) {
// 	LoadTestConfig()
// 	retCode := m.Run()
// 	os.Exit(retCode)
// }

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
			monkey.Patch(GetBigBillsRange, func() (*sheets.ValueRange, error) {
				return test.patch.result, test.patch.err
			})
			defer monkey.Unpatch(GetBigBillsRange)

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

func TestBigBillsGetLate(t *testing.T) {
	past := time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC)
	now := time.Date(2000, time.January, 4, 0, 0, 0, 0, time.UTC)
	future := time.Date(2000, time.January, 10, 0, 0, 0, 0, time.UTC)
	monkey.Patch(time.Now, func() time.Time {
		return now
	})
	var b *BigBillDate
	monkey.PatchInstanceMethod(reflect.TypeOf(b), "CheckRepayments", func(*BigBillDate) (paid bool, err error) {
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

func TestGetBigBillsRange(t *testing.T) {
	type E struct {
		result *sheets.ValueRange
		err    string
	}
	type S struct {
		result *sheets.Service
		err    error
	}
	type D struct {
		result *sheets.ValueRange
		err    error
	}
	tests := []struct {
		name    string
		service S
		do      D
		expect  E
	}{
		{
			"Creds",
			S{&sheets.Service{Spreadsheets: &sheets.SpreadsheetsService{Values: &sheets.SpreadsheetsValuesService{}}}, nil},
			D{nil, nil},
			E{nil, ""},
		},
		{
			"ServiceErr",
			S{nil, errors.New("Failure")},
			D{},
			E{nil, "Failure"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(st *testing.T) {

			monkey.Patch(sheets.NewService, func(ctx context.Context, opts ...option.ClientOption) (*sheets.Service, error) {
				return test.service.result, test.service.err
			})
			defer monkey.Unpatch(sheets.NewService)
			var obj *sheets.SpreadsheetsValuesGetCall
			monkey.PatchInstanceMethod(reflect.TypeOf(obj), "Do", func(s *sheets.SpreadsheetsValuesGetCall, opts ...googleapi.CallOption) (*sheets.ValueRange, error) {
				return test.do.result, test.do.err
			})
			defer monkey.UnpatchInstanceMethod(reflect.TypeOf(obj), "Do")

			result, err := GetBigBillsRange()

			if test.expect.err == "" {
				assert.Nil(st, err)
			} else {
				assert.NotNil(st, err)
				if err != nil {
					assert.Equal(st, test.expect.err, err.Error())
				}
			}

			assert.Equal(st, test.expect.result, result)
		})
	}

}

func TestBigBillsDateUpdatePaid(t *testing.T) {
	LoadTestConfig()
	n := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	b := BigBillDate{}

	var x *sheets.SpreadsheetsValuesUpdateCall

	monkey.PatchInstanceMethod(reflect.TypeOf(x), "ValueInputOption", func(s *sheets.SpreadsheetsValuesUpdateCall, valueInputOption string) *sheets.SpreadsheetsValuesUpdateCall {
		return s
	})
	monkey.PatchInstanceMethod(reflect.TypeOf(x), "Do", func(s *sheets.SpreadsheetsValuesUpdateCall, opts ...googleapi.CallOption) (*sheets.UpdateValuesResponse, error) {
		return nil, nil
	})

	expected := sheets.ValueRange{MajorDimension: "", Range: "", Values: [][]interface{}{{"01/01/2000"}}}
	srv, _ := sheets.NewService(context.Background())
	obj := sheets.NewSpreadsheetsValuesService(srv)
	monkey.PatchInstanceMethod(reflect.TypeOf(obj), "Update", func(s *sheets.SpreadsheetsValuesService, spreadsheetId string, range_ string, valuerange *sheets.ValueRange) (o *sheets.SpreadsheetsValuesUpdateCall) {
		assert.Equal(t, "testspreadsheetid", spreadsheetId)
		assert.Equal(t, "Big Bills!P2", range_)
		assert.Equal(t, &expected, valuerange)
		return &sheets.SpreadsheetsValuesUpdateCall{}
	})
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(obj), "Update")

	b.UpdatePaid(n)
}

func TestBigBillsDateCheckRepayments(t *testing.T) {
	LoadTestConfig()
	n := time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)
	p := time.Date(2000, 1, 2, 0, 0, 0, 0, time.Local)

	b := BigBillDate{Date: n, Amount: 100}
	var x *BigBillDate
	update_called := false
	monkey.PatchInstanceMethod(reflect.TypeOf(x), "UpdatePaid", func(o *BigBillDate, tt time.Time) error {
		update_called = true
		assert.Equal(t, p, tt)
		return nil
	})
	monkey.Patch(http.Get, func(u string) (resp *http.Response, err error) {
		assert.Equal(t, "https://fake.com/api/transactions?amount=-100.00&account=000000&created__gt=2000-01-01T00:00:00", u)
		r := http.Response{StatusCode: 200, Status: "200 OK", Body: ioutil.NopCloser(bytes.NewBufferString(`{"data":[{"id":1,"description":"","amount":-100.00,"account":37366510,"created":"2000-01-02T00:00:00+11:00"}]}`))}
		return &r, nil
	})
	paid, err := b.CheckRepayments()
	assert.Equal(t, true, paid)
	assert.Nil(t, err)
	assert.True(t, update_called, "UpdatePaid not called")

	monkey.Patch(http.Get, func(u string) (resp *http.Response, err error) {
		assert.Equal(t, "https://fake.com/api/transactions?amount=-100.00&account=000000&created__gt=2000-01-01T00:00:00", u)
		r := http.Response{StatusCode: 200, Status: "200 OK", Body: ioutil.NopCloser(bytes.NewBufferString(`{"data":null}`))}
		return &r, nil
	})
	paid, err = b.CheckRepayments()
	assert.Equal(t, false, paid)
	assert.Nil(t, err)
	assert.True(t, update_called, "UpdatePaid not called")
}
