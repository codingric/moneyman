package notify

import (
	"bigbills/config"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/codingric/moneyman/pkg/tracing"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Settings struct {
	Sid     string
	Token   string
	Mobiles []string
}

var (
	client   *http.Client
	settings *Settings
)

func init() {
	client = &http.Client{}
	client.Transport = otelhttp.NewTransport(http.DefaultTransport)
}

func Notify(message string, ctx context.Context) (response string, err error) {
	_, span := tracing.NewSpan("notify", ctx)
	defer span.End()
	if settings == nil {
		config.Unmarshal("notify", &settings)
	}
	if settings == nil || settings.Sid == "" || len(settings.Mobiles) == 0 || settings.Token == "" {
		log.Error().Msgf("Notify missing config")
		return "", errors.New("missing notify config")
	}
	endpoint := "https://api.twilio.com/2010-04-01/Accounts/" + settings.Sid + "/Messages"

	for _, mobile := range settings.Mobiles {
		params := url.Values{}
		params.Set("From", "Budget")
		params.Set("To", mobile)
		params.Set("Body", message)

		body := *strings.NewReader(params.Encode())

		req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, &body)
		req.SetBasicAuth(
			settings.Sid,
			settings.Token,
		)
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		res, err2 := client.Do(req)

		log.Debug().Msgf("Sent message to %s", mobile)

		if err2 != nil {
			log.Error().Msgf("HTTP error: %s", err2.Error())
			err = err2
			return
		}
		defer res.Body.Close()

		raw, _ := ioutil.ReadAll(res.Body)
		response = string(raw)

		if res.StatusCode != 201 {
			err = errors.New("ReponseCode: " + strconv.Itoa(res.StatusCode))
			log.Trace().Str("Response", response).Int("StatusCode", res.StatusCode).Msg("Response from Twillo")
		}
	}
	return
}
