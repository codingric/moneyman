package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/mitchellh/mapstructure"
	"github.com/monaco-io/request"
	"github.com/rapidloop/skv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/teambition/rrule-go"
)

var (
	ageKey     *age.X25519Identity
	httpClient *http.Client
	skvStore   *skv.KVStore
)

func init() {
	httpClient = &http.Client{}
}

func initStore() {
	var err error
	skvStore, err = skv.Open(viper.GetString("store"))
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to init KV store")
	}
}

func LoadConf() {
	flag.Bool("dryrun", false, "Log instead of SMS")
	flag.String("agekey", "/etc/auditor/age.key", "Age key")
	flag.String("loglevel", zerolog.ErrorLevel.String(), "trace|debug|info|error|fatal|panic|disabled")
	flag.String("store", "/var/run/auditor/notifications.db", "Location for KV store")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/auditor/")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal().Msgf("Unable to read config: %v", err)
	}

	lvl := zerolog.InfoLevel
	lvl, err = zerolog.ParseLevel(viper.GetString("loglevel"))
	if err != nil {
		log.Error().Err(err).Msgf("Ignoring --loglevel")
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	log.Info().Msgf("Config loaded: `%s`", viper.ConfigFileUsed())

	ageKey, err = loadAgeKey(viper.GetString("agekey"))
	if err != nil {
		log.Fatal().Msgf("Unable to load age key: %s", err.Error())
	}
}

func loadAgeKey(p string) (a *age.X25519Identity, e error) {
	b, err := os.ReadFile(p) // just pass the file name
	if err != nil {
		return nil, err
	}
	// Remove comments
	re := regexp.MustCompile(`(s?)#.*\n`)
	c := re.ReplaceAll(b, nil)
	str := string(c) // convert content to a 'string'

	a, e = age.ParseX25519Identity(strings.Trim(str, "\n"))
	return
}

func decodeAge(s string, a *age.X25519Identity) string {
	enc := strings.TrimPrefix(s, "age:")
	eb, _ := base64.StdEncoding.DecodeString(enc)
	r := bytes.NewReader(eb)
	d, _ := age.Decrypt(r, a)
	b := &bytes.Buffer{}
	io.Copy(b, d)
	return b.String()
}

func ageHookFunc(a *age.X25519Identity) mapstructure.DecodeHookFuncType {
	// Wrapped in a function call to add optional input parameters (eg. separator)
	return func(
		f reflect.Type, // data type
		t reflect.Type, // target data type
		data interface{}, // raw data
	) (interface{}, error) {

		// Check if the data type matches the expected one
		if f.Kind() != reflect.String {
			return data, nil
		}

		// Check if the target type matches the expected one
		if t.Kind() != reflect.String {
			return data, nil
		}

		if !strings.HasPrefix(data.(string), "age:") {
			return data, nil
		}

		return decodeAge(data.(string), a), nil
	}
}

func main() {
	LoadConf()

	if err := RunChecks(); err != nil {
		log.Fatal().Msgf("Failure: %s", err.Error())
	}
}

type Checks []Check

type Check struct {
	Name string `mapstructure:"name"`
	Type string `mapstructure:"type"`
}

type Repay struct {
	Name  string `mapstructure:"name"`
	Match string `mapstructure:"match"`
	Days  int    `mapstructure:"days"`
	From  string `mapstructure:"from"`
	To    string `mapstructure:"to"`
}

type Amount struct {
	Name      string  `mapstructure:"name"`
	Match     string  `mapstructure:"match"`
	Days      int     `mapstructure:"days"`
	Expected  float64 `mapstructure:"expected"`
	Threshold string  `mapstructure:"threshold"`
	Rrule     string  `mapstructure:"rrule"`
}

func RunChecks() error {
	var checks Checks

	viper.UnmarshalKey("checks", &checks)
	for i, check := range checks {

		log.Info().Msgf("Checking: %s (%s)", check.Name, check.Type)
		var message string
		switch check.Type {
		case "repay":
			var err error
			var c Repay
			viper.UnmarshalKey(fmt.Sprintf("checks.%d", i), &c)
			message, err = CheckRepay(c)
			if err != nil {
				return err
			}
		case "amount":
			var e error
			var c Amount
			viper.UnmarshalKey(fmt.Sprintf("checks.%d", i), &c)
			message, e = CheckAmount(c)
			if e != nil {
				return e
			}
		default:
			return errors.New("Invalid check type: " + check.Type)
		}
		if message != "" {
			err := Notify(message)
			if err != nil {
				log.Error().Err(err).Msgf("Notify error: %s", err.Error())
				return err
			}
		}

	}
	return nil
}

type APIResponse struct {
	Data []APITransaction `json:"data"`
}

type APITransaction struct {
	Id          int64     `json:"id"`
	Description string    `json:"description"`
	Amount      float32   `json:"amount"`
	Account     int64     `json:"account"`
	Created     time.Time `json:"created"`
}

func QueryBackend(params map[string]string) (result APIResponse, err error) {
	client := request.Client{
		URL:    viper.GetString("backend"),
		Query:  params,
		Method: "GET",
	}
	p := url.Values{}
	for k, v := range params {
		p.Set(k, v)
	}

	log.Trace().Str("params", p.Encode()).Msg("Query backend")

	check := client.Send()

	//check.Response().Request.URL

	if !check.OK() {
		err = check.Error()
		return
	}

	if check.Response().StatusCode != 200 {
		err = errors.New(strings.ToLower(check.String()))
		return
	}

	if !check.Scan(&result).OK() {
		err = check.Error()
		return
	}
	return
}

func CheckAmount(c Amount) (msg string, err error) {
	past := time.Now().AddDate(0, 0, -c.Days)
	if c.Rrule != "" {
		rr, err := rrule.StrToRRule(c.Rrule)
		if err != nil {
			log.Error().Err(err).Msgf("Unable to parse rrule '%s'", c.Rrule)
			return "", fmt.Errorf("rrule invalid")
		}

		past = rr.Before(time.Now(), true).AddDate(0, 0, -c.Days)
	}

	thresh := 0.0
	params := map[string]string{
		"description__like": c.Match,
		"created__gt":       past.Format("2006-01-02T15:04:05"),
	}
	if c.Threshold == "" {
		params["amount__ne"] = fmt.Sprintf("%0.2f", c.Expected)
	} else if strings.Contains(c.Threshold, "%") {
		t, _ := strconv.ParseFloat(strings.Replace(c.Threshold, "%", "", 1), 64)
		thresh = math.Abs(c.Expected * (t / 100.0))
		params["amount__lt"] = fmt.Sprintf("%0.2f", c.Expected-thresh)
	} else {
		thresh, _ = strconv.ParseFloat(strings.Replace(c.Threshold, "$", "", 1), 64)
		params["amount__lt"] = fmt.Sprintf("%0.2f", c.Expected-math.Abs(thresh))

	}

	response, err := QueryBackend(params)
	if err != nil {
		return
	}

	if len(response.Data) > 0 {
		msg = "Unexpected amounts:"
		for _, t := range response.Data {
			msg = fmt.Sprintf(
				"%s\n%s %s for $%0.2f expecting $%0.2f",
				msg,
				t.Created.Format("Mon 2 Jan"),
				t.Description,
				math.Abs(float64(t.Amount)),
				math.Abs(c.Expected),
			)
		}
	}

	return
}

func CheckRepay(c Repay) (msg string, err error) {
	past := time.Now().AddDate(0, 0, c.Days*-1)
	params := map[string]string{
		"description__like": c.Match,
		"created__gt":       past.Format("2006-01-02T15:04:05"),
		"amount__lt":        "0.00",
	}
	var result []string

	response, err := QueryBackend(params)
	if err != nil {
		return
	}

	for _, transaction := range response.Data {
		p := map[string]string{
			"created__gt":       transaction.Created.Format("2006-01-02T15:04:05"),
			"description__like": c.From,
			"amount":            fmt.Sprintf("%0.2f", transaction.Amount*-1),
		}

		rr, err := QueryBackend(p)
		if err != nil {
			return msg, err
		}

		if len(rr.Data) == 0 {
			m := fmt.Sprintf("$%0.2f from %s", transaction.Amount*-1, transaction.Created.Format("Mon 2 Jan"))
			result = append(result, m)
		}
	}

	if len(response.Data) > 0 {
		unpaid := len(result)
		found := len(response.Data)
		log.Info().Msgf("Transactions: %d/%d repaid", found-unpaid, found)
	}
	if len(result) > 0 {
		msg = fmt.Sprintf("Move money from %s to %s:", c.From, c.To)
		for _, row := range result {
			msg = fmt.Sprintf("%s\n%s", msg, row)
		}
	}
	return
}

type TwilioResponse struct {
	XMLName       xml.Name            `xml:"TwilioResponse"`
	RestException TwilioRestException `xml:"RestException"`
}

type TwilioRestException struct {
	XMLName  xml.Name `xml:"RestException"`
	Code     int64    `xml:"Code"`
	Message  string   `xml:"Message"`
	MoreInfo string   `xml:"MoreInfo"`
	Status   int64    `xml:"Status"`
}

type Settings struct {
	Sid     string   `mapstructure:"sid"`
	Token   string   `mapstructure:"token"`
	Mobiles []string `mapstructure:"mobiles"`
}

func Notify(message string) (err error) {
	var settings *Settings
	var store *skv.KVStore

	if err := viper.UnmarshalKey("notify", &settings, viper.DecodeHook(ageHookFunc(ageKey))); err != nil || settings == nil {
		log.Error().Err(err).Msg("Unable to load notify config")
		return fmt.Errorf("unable to load settings")
	}

	if skvStore == nil {
		initStore()
	}

	endpoint := "https://api.twilio.com/2010-04-01/Accounts/" + settings.Sid + "/Messages"

	for _, m := range settings.Mobiles {
		body := url.Values{
			"To":   []string{m},
			"From": []string{"Budget"},
			"Body": []string{message},
		}

		hash := fmt.Sprintf("%x", md5.Sum([]byte(body.Encode())))
		if err := store.Get(hash, nil); err == nil {
			log.Info().Str("hash", hash).Msgf("Message already sent to %s", m)
			continue
		}
		store.Put(hash, m)

		req, _ := http.NewRequest("POST", endpoint, strings.NewReader(body.Encode()))
		req.SetBasicAuth(settings.Sid, settings.Token)

		if viper.GetBool("dryrun") {
			log.Info().Msgf("(DRYRUN) SMS : %s", message)
			continue
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to make request")
			return fmt.Errorf("failed to make request")
		}

		switch resp.StatusCode {
		case 201:
			log.Debug().Msg("Sent SMS Successfully")
			return nil
		case 401:
			return errors.New("authentication failure")
		case 400:
			var resp_xml TwilioResponse
			var resp_body []byte
			resp_body, _ = io.ReadAll(resp.Body)
			err = xml.Unmarshal(resp_body, &resp_xml)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to read response")
				return fmt.Errorf("failed to read responses")
			}
			return errors.New(resp_xml.RestException.Message)
		default:
			return errors.New("twilio responded with failure")
		}
	}
	return nil
}
