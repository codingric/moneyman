package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/spf13/viper"
)

type BackendTransaction struct {
	Created     time.Time `json:"created" binding:"required"`
	Amount      string    `json:"amount" binding:"required"`
	Description string    `json:"description" binding:"required"`
	Account     string    `json:"account" binding:"required"`
	Successful  bool      `json:"-"`
}

func (t *BackendTransaction) Post() error {
	if t == nil {
		return errors.New("uninitialized transaction")
	}
	if t.Successful {
		return nil
	}

	payload, _ := json.Marshal(t)

	req, err := http.NewRequest("POST", viper.GetString("backend"), bytes.NewBuffer(payload))
	if err != nil {
		logger.Error("Unable to create Request: %s", err.Error())
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	logger.Trace("POST %s:\n%s", viper.GetString("backend"), string(payload))

	if err != nil {
		logger.Error("Failure calling backend: %s", err.Error())
		return err
	}

	raw, _ := ioutil.ReadAll(resp.Body)
	body := string(raw)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		logger.Trace("Backend reponse: %s", body)
		return errors.New("unsucessful statuscode returned")
	}

	return nil
}
