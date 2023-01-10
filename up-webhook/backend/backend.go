package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/codingric/moneyman/pkg/tracing"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var (
	httpClient *http.Client
)

func init() {
	httpClient = &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
}

type BackendTransaction struct {
	Created     time.Time `json:"created" binding:"required"`
	Amount      string    `json:"amount" binding:"required"`
	Description string    `json:"description" binding:"required"`
	Account     string    `json:"account" binding:"required"`
	Successful  bool      `json:"-"`
}

func (t *BackendTransaction) Post(ctx context.Context) error {
	ctx, span := tracing.NewSpan("UpTransaction.Get", ctx)
	defer span.End()

	if t == nil {
		return errors.New("uninitialized transaction")
	}
	span.SetAttributes(
		attribute.String("created", t.Created.Format("2006/01/02")),
		attribute.String("account", t.Account),
		attribute.String("description", t.Description),
		attribute.String("amount", t.Amount),
	)
	if t.Successful {
		return nil
	}

	payload, _ := json.Marshal(t)

	req, err := http.NewRequestWithContext(ctx, "POST", viper.GetString("backend"), bytes.NewBuffer(payload))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Unable to create Request")
		log.Error().Err(err).Msg("Unable to create Request")
		return err
	}

	resp, err := httpClient.Do(req)

	log.Trace().Msgf("POST %s:\n%s", viper.GetString("backend"), string(payload))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failure calling backend")
		log.Error().Err(err).Msg("Failure calling backend")
		return err
	}

	raw, _ := ioutil.ReadAll(resp.Body)
	body := string(raw)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		span.RecordError(err)
		span.SetStatus(codes.Error, "unsucessful statuscode returned")
		log.Trace().Msgf("Backend reponse: %s", body)
		return errors.New("unsucessful statuscode returned")
	}

	return nil
}
