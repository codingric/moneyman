package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/spf13/viper"
)

type UpData struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

type UpLinks struct {
	Related string `json:"related"`
}

type UpTypes struct {
	Data  UpData  `json:"data"`
	Links UpLinks `json:"links"`
}

type UpWebhookEvent struct {
	Data struct {
		Type       string `json:"type"`
		Id         string `json:"id"`
		Attributes struct {
			EventType string    `json:"eventType"`
			CreatedAt time.Time `json:"createdAt"`
		} `json:"attributes"`
		Relationships struct {
			Webhook     UpTypes `json:"webhook"`
			Transaction UpTypes `json:"transaction"`
		} `json:"relationships"`
	} `json:"data"`
}

type UpTransaction struct {
	Data TransactionResource `json:"data"`
}

type TransactionResource struct {
	Type          string                   `json:"type"`
	Id            string                   `json:"id"`
	Attributes    TransactionAttributes    `json:"attributes"`
	Relationships TransactionRelationships `json:"relationships"`
	/*Links         TransactionLinks*/
}

type TransactionAttributes struct {
	Status      TransactionStatusEnum `json:"status"`
	RawText     string                `json:"rawText"`
	Description string                `json:"description"`
	Message     string                `json:"message"`
	Amount      Amount                `json:"amount"`
	SettledAt   time.Time             `json:"settledAt"`
	CreatedAt   time.Time             `json:"createdAt"`
	//IsCategorizable bool                  `json:"isCategorizable"`
	//HoldInfo        HoldInfo              `json:"holdInfo"`
	//RoundUp         RoundUp               `json:"roundUp"`
	//Cashback        Cashback              `json:"cashback"`
	//ForeignAmount   Amount                `json:"foreignAmount"`
}

type TransactionRelationships struct {
	Account UpTypes `json:"account"`
}

type Amount struct {
	CurrencyCode     string `json:"currencyCode"`
	Value            string `json:"value"`
	ValueInBaseUnits int64  `json:"valueInBaseUnits"`
}

type TransactionStatusEnum string

func (t *UpTransaction) Get(id string) error {
	url := fmt.Sprintf("https://api.up.com.au/api/v1/transactions/%s", id)
	req, _ := http.NewRequest("GET", url, bytes.NewBuffer([]byte(``)))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", viper.GetString("bearer")))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error("Failed to get transaction (%s): %s", id, err.Error())
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		logger.Error("Failed StatusCode transaction (%s): %s", id, resp.StatusCode)
	}

	resp_bytes, _ := ioutil.ReadAll(resp.Body)
	logger.Debug("Response: %s", string(resp_bytes))

	if err := json.Unmarshal(resp_bytes, &t); err != nil {
		logger.Error("Failure parsing response: Transaction(%s) - %s", id, err.Error())
		return err
	}

	if t.Data.Id == "" || t.Data.Attributes.Amount.ValueInBaseUnits == 0 {
		logger.Error("Response not valid")
		return fmt.Errorf("transaction %s unable to be retreived", id)
	}

	logger.Debug("Transaction: %v", t)

	return nil
}
