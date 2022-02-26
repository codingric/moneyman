package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/spf13/viper"
)

func RunServer() error {
	http.HandleFunc("/", WebhookHandler)
	logger.Info("Webhook listening on port " + viper.GetString("port"))
	return http.ListenAndServe("0.0.0.0:"+viper.GetString("port"), nil)
}

func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	logger.Trace("%s %s", r.Method, r.RequestURI)
	receivedSignature, _ := hex.DecodeString(
		r.Header.Get("X-Up-Authenticity-Signature"),
	)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	logger.Debug("Body:\n\n%s", string(body))

	mac := hmac.New(sha256.New, []byte(viper.GetString("secret_key")))
	mac.Write(body)
	signature := mac.Sum(nil)

	if !hmac.Equal(signature, receivedSignature) {
		logger.Error("Invalid signature: %x, %x", signature, receivedSignature)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	w.Write([]byte("OK"))
}
