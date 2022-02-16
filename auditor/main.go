package main

import (
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/monaco-io/request"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func LoadConf() {
	flag.Bool("v", false, "Verbose")
	flag.Bool("dryrun", false, "Log instead of SMS")

	viper.RegisterAlias("verbose", "v")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/auditor/")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Fatalf("%v\n", err)
	}

	if viper.GetBool("verbose") {
		log.Println("Verbose: ON")
		log.Printf("Config: `%s`\n", viper.ConfigFileUsed())
	} else {
		log.Println("Verbose: OFF")
	}
}

func main() {
	LoadConf()

	if err := RunChecks(); err != nil {
		log.Fatalf("Failure: %s", err.Error())
	}
}

type ConfigChecks struct {
	Checks []map[string]string `mapstructure:"checks"`
}

func RunChecks() error {
	var checks []map[string]string
	viper.UnmarshalKey("checks", &checks)
	for _, check := range checks {
		if viper.GetBool("verbose") {
			log.Printf("Checking: %s (%s)", check["match"], check["type"])
		}
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
				log.Printf("Notify error: %s", err.Error())
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

	if len(response.Data) > 0 && viper.GetBool("verbose") {
		unpaid := len(result)
		found := len(response.Data)
		log.Printf("Transactions: %d/%d repaid", found-unpaid, found)
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

func Notify(message string) (err error) {
	if viper.GetBool("dryrun") {
		log.Println("SMS Constructed\n", message)
		return nil
	}

	endpoint := "https://api.twilio.com/2010-04-01/Accounts/" + viper.GetString("notify.sid") + "/Messages"

	for _, m := range viper.GetStringSlice("notify.mobiles") {
		body := map[string]string{
			"To":   m,
			"From": "Budget",
			"Body": message,
		}
		client := request.Client{
			Method:         "POST",
			URL:            endpoint,
			BasicAuth:      request.BasicAuth{viper.GetString("notify.sid"), viper.GetString("notify.token")},
			URLEncodedForm: body,
		}
		resp := client.Send()

		switch resp.Response().StatusCode {
		case 201:
			if viper.GetBool("verbose") {
				log.Println("Sent SMS Successfully")
			}
			continue
		case 401:
			err = errors.New("authentication failure")
		case 400:
			var xml TwilioResponse
			resp.ScanXML(&xml)
			err = errors.New(xml.RestException.Message)
		default:
			err = errors.New("twilio responded with failure")
		}
		return err
	}
	return nil
}
