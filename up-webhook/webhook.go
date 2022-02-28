package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/spf13/viper"
)

func RunWebhook() error {
	http.HandleFunc("/", WebhookHandler)
	logger.Info("Webhook listening on port " + viper.GetString("port"))
	return http.ListenAndServe("0.0.0.0:"+viper.GetString("port"), nil)
}

func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	logger.Trace("%s %s", r.Method, r.RequestURI)
	sig, _ := hex.DecodeString(
		r.Header.Get("X-Up-Authenticity-Signature"),
	)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Error("Error reading body: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	logger.Trace("Body:\n\n%s", string(body))

	valid_sig, err := validateSignature(body, sig)
	if err != nil {
		logger.Error("Failure validating signature")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !valid_sig {
		logger.Error("Signature failed validation")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var event UpWebhookEvent
	err = json.Unmarshal(body, &event)
	if err != nil {
		logger.Error("Failed to parse WebhookEvent: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	logger.Debug("WebhookEvent: %v", event)

	w.Write([]byte("OK\n"))
}

func validateSignature(body []byte, signature []byte) (bool, error) {
	if viper.GetString("secret_key") == "" {
		return false, errors.New("missing secret_key")
	}
	mac := hmac.New(sha256.New, []byte(viper.GetString("secret_key")))
	mac.Write(body)
	generated := mac.Sum(nil)
	logger.Debug("Signatures: received=%x, generated=%x", signature, generated)
	return hmac.Equal(signature, generated), nil
}
