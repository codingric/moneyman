package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/jarcoal/httpmock"
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

func FindAllGroups(regex string, target string) map[string]string {
	matches := make(map[string]string)
	re := regexp.MustCompile(regex)
	n := re.SubexpNames()
	for i, m := range re.FindStringSubmatch(target) {
		if n[i] != "" {
			matches[n[i]] = m
		}
	}
	return matches
}

func SetUpConfig() {
	var config = []byte(`
checks:
  MOCK:
    from: mockfrom
    to: mockto
filters:
  days: 3
backend: https://api.mock/api/transactions
notify:
  sid: mocksid
  token: mocktoken
  mobiles:
    - +61412345678`)

	viper.SetConfigType("yaml") // or viper.SetConfigType("YAML")

	viper.ReadConfig(bytes.NewBuffer(config))
	viper.RegisterAlias("verbose", "v")
	viper.Set("verbose", false)
}

func SetMockSuccessTransactions() {
	all := []APITransaction{
		{Id: 1, Description: "MOCK", Amount: -1.11, Account: 1234, Created: time.Now()},
		{Id: 2, Description: "MOCK", Amount: -2.22, Account: 1234, Created: time.Now()},
		{Id: 3, Description: "MOCK", Amount: -3.33, Account: 1234, Created: time.Now()},
		{Id: 4, Description: "mockfrom", Amount: 1.11, Account: 1234, Created: time.Now()},
	}
	filterByAmount := func(url string) []APITransaction {
		f := []APITransaction{}
		m := FindAllGroups(`.*amount(?P<op>__lt|__gt)?=(?P<amount>\d+(\.\d\d)?).*`, url)
		a, _ := strconv.ParseFloat(m["amount"], 32)
		for _, i := range all {
			if m["op"] == "" {
				if i.Amount == float32(a) {
					f = append(f, i)
				}
			}
			if m["op"] == "__lt" {
				if i.Amount < float32(a) {
					f = append(f, i)
				}
			}
		}
		return f
	}
	httpmock.RegisterResponder("GET", "=~^https://api.mock/api/",
		func(req *http.Request) (*http.Response, error) {
			data := APIResponse{Data: filterByAmount(req.URL.RequestURI())}
			resp, _ := httpmock.NewJsonResponse(200, data)
			return resp, nil
		})

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
			resp := httpmock.NewStringResponse(200, success_resp)
			return resp, nil
		})
}

func SetUpRunChecksMocks() {
	monkey.Patch(RunCheck, func(name string, from string) (result []string, err error) {
		return []string{"Patched"}, nil
	})
	monkey.Patch(Notify, func(message string) (err error) {
		return nil
	})
}

func TearDownRunChecksMocks() {
	monkey.Unpatch(RunCheck)
	monkey.Unpatch(Notify)
}

func TestMain(m *testing.M) {
	SetUpConfig()
	retCode := m.Run()
	os.Exit(retCode)
}

func TestRunCheck(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	SetMockSuccessTransactions()
	viper.Set("verbose", true)

	expected := []string{fmt.Sprintf("$2.22 from %s", time.Now().Format("Mon 2 Jan")), fmt.Sprintf("$3.33 from %s", time.Now().Format("Mon 2 Jan"))}

	result, err := RunCheck("MOCK", "mockfrom")
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
}

func TestNotify(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	SetMockTwilio()

	err := Notify("mockmessage")
	assert.Nil(t, err)
}

func TestNotifyFailOnException(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	SetMockTwilio()

	err := Notify("failure")
	assert.NotNil(t, err)
}

func TestNotifyFailOnAuth(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	SetMockTwilio()

	viper.Set("notify.token", "xxx")
	err := Notify("mockmessage")
	assert.NotNil(t, err)
}

func TestNegativeRunCheck(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	SetMockFailureTransactions()

	expected := []string(nil)

	result, err := RunCheck("MOCK", "mockfrom")
	assert.NotNil(t, err)
	assert.Equal(t, expected, result)
}

func TestRunChecks(t *testing.T) {
	SetUpRunChecksMocks()
	defer TearDownRunChecksMocks()

	err := RunChecks()
	assert.Nil(t, err)
}

func TestRunChecksFailOnChecks(t *testing.T) {
	monkey.Patch(RunCheck, func(name string, from string) (result []string, err error) {
		return []string{"Patched"}, errors.New("Failure")
	})
	defer monkey.Unpatch(RunCheck)

	err := RunChecks()
	assert.NotNil(t, err)
}
func TestRunChecksFailOnNotify(t *testing.T) {
	monkey.Patch(RunCheck, func(name string, from string) (result []string, err error) {
		return []string{"Patched"}, nil
	})
	monkey.Patch(Notify, func(message string) (err error) {
		return errors.New(`error`)
	})
	defer monkey.Unpatch(RunCheck)
	defer monkey.Unpatch(Notify)

	err := RunChecks()
	assert.NotNil(t, err)
}
