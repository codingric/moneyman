package main

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/monaco-io/request"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var ageKey *age.X25519Identity

func LoadConf() {
	flag.Bool("dryrun", false, "Log instead of SMS")
	flag.String("agekey", "/etc/auditor/age.key", "Age key")
	flag.String("loglevel", zerolog.ErrorLevel.String(), "trace|debug|info|error|fatal|panic|disabled")

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

type ConfigChecks struct {
	Checks []map[string]string `mapstructure:"checks"`
}

func RunChecks() error {
	var checks []map[string]string
	viper.UnmarshalKey("checks", &checks)
	for _, check := range checks {
		log.Info().Msgf("Checking: %s (%s)", check["match"], check["type"])
		days, _ := strconv.Atoi(check["days"])
		message := string("")
		switch check["type"] {
		case "repay":
			var err error
			message, err = CheckRepay(check["match"], check["from"], check["to"], days)
			if err != nil {
				return err
			}
		case "amount":
			var e error
			expected, _ := strconv.ParseFloat(check["expected"], 64)
			message, e = CheckAmount(check["match"], expected, check["threshold"], days)
			if e != nil {
				return e
			}
		default:
			return errors.New("Invalid check type: " + check["type"])
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

	check := client.Send()

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

func CheckAmount(match string, expected float64, threshold string, days int) (msg string, err error) {
	past := time.Now().AddDate(0, 0, days*-1)
	thresh := 0.0
	params := map[string]string{
		"description__like": match,
		"created__gt":       past.Format("2006-01-02T15:04:05"),
	}
	if threshold == "" {
		params["amount__ne"] = fmt.Sprintf("%0.2f", expected)
	} else if strings.Contains(threshold, "%") {
		t, _ := strconv.ParseFloat(strings.Replace(threshold, "%", "", 1), 64)
		thresh = math.Abs(expected * (t / 100.0))
		params["amount__lt"] = fmt.Sprintf("%0.2f", expected-thresh)

	} else {
		thresh, _ = strconv.ParseFloat(strings.Replace(threshold, "$", "", 1), 64)
		params["amount__lt"] = fmt.Sprintf("%0.2f", expected-math.Abs(thresh))

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
				math.Abs(expected),
			)
		}
	}

	return
}

func CheckRepay(match string, from string, to string, days int) (msg string, err error) {
	past := time.Now().AddDate(0, 0, days*-1)
	params := map[string]string{
		"description__like": match,
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
			"description__like": from,
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
		msg = fmt.Sprintf("Move money from %s to %s:", from, to)
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
	if viper.GetBool("dryrun") {
		log.Info().Msgf("(DRYRUN) SMS : %s", message)
		return nil
	}

	var settings *Settings
	if err := viper.UnmarshalKey("notify", &settings, viper.DecodeHook(ageHookFunc(ageKey))); err != nil {
		log.Error().Err(err).Msg("Unable to load notify config")
	}

	endpoint := "https://api.twilio.com/2010-04-01/Accounts/" + settings.Sid + "/Messages"

	for _, m := range settings.Mobiles {
		body := map[string]string{
			"To":   m,
			"From": "Budget",
			"Body": message,
		}
		client := request.Client{
			Method: "POST",
			URL:    endpoint,
			BasicAuth: request.BasicAuth{
				Username: settings.Sid,
				Password: settings.Token,
			},
			URLEncodedForm: body,
		}
		resp := client.Send()

		switch resp.Response().StatusCode {
		case 201:
			log.Debug().Msg("Sent SMS Successfully")
			return nil
		case 401:
			return errors.New("authentication failure")
		case 400:
			var xml TwilioResponse
			resp.ScanXML(&xml)
			return errors.New(xml.RestException.Message)
		default:
			return errors.New("twilio responded with failure")
		}
	}
	return errors.New("no mobiles to send to")
}
