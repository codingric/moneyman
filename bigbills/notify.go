package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

func Notify(message string) (response string, err error) {

	if viper.GetString("sid") == "" || len(viper.GetStringSlice("mobiles")) == 0 || viper.GetString("token") == "" {
		return "", errors.New("Missing notify config")
	}

	endpoint := "https://api.twilio.com/2010-04-01/Accounts/" + viper.GetString("sid") + "/Messages"

	for _, mobile := range viper.GetStringSlice("mobiles") {
		params := url.Values{}
		params.Set("From", "Budget")
		params.Set("To", mobile)
		params.Set("Body", message)

		body := *strings.NewReader(params.Encode())

		client := &http.Client{}
		req, _ := http.NewRequest("POST", endpoint, &body)
		req.SetBasicAuth(viper.GetString("sid"), viper.GetString("token"))
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		res, err2 := client.Do(req)

		if viper.GetBool("verbose") {
			log.Printf("Sent message to %s", mobile)
		}

		if err2 != nil {
			err = err2
			return
		}
		defer res.Body.Close()

		raw, _ := ioutil.ReadAll(res.Body)
		response = string(raw)

		if res.StatusCode != 201 {
			err = errors.New("ReponseCode: " + strconv.Itoa(res.StatusCode))
			log.Printf("Response: %v", response)
		}
	}
	return
}
