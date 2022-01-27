package main

import (
	"flag"
	"fmt"
	"log"
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

func RunChecks() {
	for name := range viper.GetStringMap("checks") {
		from := viper.GetString("checks." + name + ".from")
		to := viper.GetString("checks." + name + ".to")
		outstanding := RunCheck(name, from)

		if len(outstanding) > 0 {
			message := fmt.Sprintf("Move money from %s to %s:", from, to)
			for _, row := range outstanding {
				message = fmt.Sprintf("%s\n%s", message, row)
			}
			Notify(message)
		}
	}
}

type APIResponse struct {
	Data []APITransaction `json:"data"`
}

type APITransaction struct {
	Id          string    `json:"id" gorm:"primary_key"`
	Description string    `json:"description"`
	Amount      float32   `json:"amount"`
	Account     string    `json:"account"`
	Created     time.Time `json:"created"`
}

func RunCheck(name string, from string) (result []string) {
	if viper.GetBool("verbose") {
		log.Println("Checking:", name)
	}
	past := time.Now().AddDate(0, 0, -3)
	params := map[string]string{
		"description__like": name,
		"created__gt":       past.Format("2006-01-02T15:04:05"),
	}

	client := request.Client{
		URL:    viper.GetString("backend"),
		Query:  params,
		Method: "GET",
	}

	var response APIResponse

	client.Send().Scan(&response)

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
		c.Send().Scan(&rr)

		if len(rr.Data) == 0 {
			m := fmt.Sprintf("$%0.2f from %s", transaction.Amount*-1, transaction.Created.Format("Mon 2 Jan"))
			result = append(result, m)
		}
	}

	return
}

func Notify(message string) {
	if viper.GetBool("dryrun") {
		log.Println("SMS Constructed\n", message)
		return
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
		client.Send()
		if viper.GetBool("verbose") {
			log.Println("Sent SMS Successfully:\n", message)
		}
	}

}
