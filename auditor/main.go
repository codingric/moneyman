package main

import (
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"log"
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
	RunChecks()
}

func RunChecks() error {
	for name := range viper.GetStringMap("checks") {
		from := viper.GetString("checks." + name + ".from")
		to := viper.GetString("checks." + name + ".to")
		outstanding, err := RunCheck(name, from)
		if err != nil {
			log.Printf("Error during check: %s", err.Error())
			return err
		}

		if len(outstanding) > 0 {
			message := fmt.Sprintf("Move money from %s to %s:", from, to)
			for _, row := range outstanding {
				message = fmt.Sprintf("%s\n%s", message, row)
			}
			err = Notify(message)
			if err != nil {
				log.Printf("Error during notify: %s", err.Error())
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

func RunCheck(name string, from string) (result []string, err error) {
	if viper.GetBool("verbose") {
		log.Println("Checking:", name)
	}
	past := time.Now().AddDate(0, 0, -3)
	params := map[string]string{
		"description__like": name,
		"created__gt":       past.Format("2006-01-02T15:04:05"),
		"amount__lt":        "0",
	}

	client := request.Client{
		URL:    viper.GetString("backend"),
		Query:  params,
		Method: "GET",
	}

	var response APIResponse

	check := client.Send() //.Scan(&response)

	if check.Response().StatusCode != 200 {
		err = errors.New(strings.ToLower(check.String()))
		return
	}

	if !check.Scan(&response).OK() {
		err = check.Error()
		return
	}

	for _, transaction := range response.Data {
		p := map[string]string{
			"created__gt":       transaction.Created.Format("2006-01-02T15:04:05"),
			"description__like": from,
			"amount":            fmt.Sprintf("%0.2f", transaction.Amount*-1),
		}
		c := request.Client{
			Method: "GET",
			URL:    viper.GetString("backend"),
			Query:  p,
		}
		var rr APIResponse
		cc := c.Send()
		if cc.Response().StatusCode != 200 {
			err = errors.New(strings.ToLower(cc.String()))
			return
		}

		if !cc.Scan(&rr).OK() {
			err = cc.Error()
			return
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

		if resp.Response().StatusCode != 200 {
			switch resp.Response().StatusCode {
			case 401:
				err = errors.New("authentication failure")
				break
			case 400:
				var xml TwilioResponse
				resp.ScanXML(&xml)
				err = errors.New(xml.RestException.Message)
				break
			default:
				err = errors.New("twilio responded with failure")
			}
			return err
		}

		if viper.GetBool("verbose") {
			log.Println("Sent SMS Successfully")
		}
	}
	return nil
}
