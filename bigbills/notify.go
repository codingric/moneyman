package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func Notify(message string, config AppConfig) (response string, err error) {

	endpoint := "https://api.twilio.com/2010-04-01/Accounts/" + config.Sid + "/Messages"

	for _, mobile := range config.Mobiles {
		params := url.Values{}
		params.Set("From", "Budget")
		params.Set("To", mobile)
		params.Set("Body", message)

		body := *strings.NewReader(params.Encode())

		client := &http.Client{}
		req, _ := http.NewRequest("POST", endpoint, &body)
		req.SetBasicAuth(config.Sid, config.Token)
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		res, err2 := client.Do(req)

		if *verbose {
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
