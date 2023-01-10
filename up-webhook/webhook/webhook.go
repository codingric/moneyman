package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/codingric/moneyman/pkg/tracing"
	"github.com/codingric/moneyman/up-webhook/backend"
	"github.com/codingric/moneyman/up-webhook/up"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/codes"
)

func RunWebhook(ctx context.Context) error {
	http.Handle("/", otelhttp.NewHandler(http.HandlerFunc(WebhookHandler), "incoming.up-webhook"))
	log.Info().Msg("Webhook listening on port " + viper.GetString("port"))
	return http.ListenAndServe("0.0.0.0:"+viper.GetString("port"), nil)
}

func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.NewSpan("WebhookHandler", r.Context())
	defer span.End()

	log.Trace().Msgf("%s %s", r.Method, r.RequestURI)
	sig, _ := hex.DecodeString(
		r.Header.Get("X-Up-Authenticity-Signature"),
	)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Error reading body")
		log.Error().Err(err).Msg("Error reading body")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	log.Trace().Msgf("Body:\n\n%s", string(body))

	valid_sig, err := validateSignature(body, sig)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failure validating signature")
		log.Error().Msg("Failure validating signature")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !valid_sig {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failure validating signature")
		log.Error().Msg("Signature failed validation")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var event up.UpWebhookEvent
	err = json.Unmarshal(body, &event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse WebhookEvent")
		log.Error().Err(err).Msg("Failed to parse WebhookEvent")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Debug().Msgf("WebhookEvent: %v", event)

	if event.Data.Attributes.EventType == "TRANSACTION_CREATED" {

		var trans up.UpTransaction
		if err := trans.Get(event.Data.Relationships.Transaction.Data.Id, ctx); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to get Transaction")
			log.Error().Msgf("Failed to get Transaction: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		backend := backend.BackendTransaction{
			Created:     trans.Data.Attributes.CreatedAt,
			Amount:      trans.Data.Attributes.Amount.Value,
			Description: trans.Data.Attributes.Description,
			Account:     trans.Data.Relationships.Account.Data.Id,
		}
		if err := backend.Post(ctx); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to save backend transaction")
			log.Error().Msgf("Failed to save backend Transaction: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	w.Write([]byte("OK\n"))
}

func validateSignature(body []byte, signature []byte) (bool, error) {
	if viper.GetString("secret_key") == "" {
		return false, errors.New("missing secret_key")
	}
	mac := hmac.New(sha256.New, []byte(viper.GetString("secret_key")))
	mac.Write(body)
	generated := mac.Sum(nil)
	log.Debug().Msgf("Signatures: received=%x, generated=%x", signature, generated)
	return hmac.Equal(signature, generated), nil
}
