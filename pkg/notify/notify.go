package notify

import (
	"context"
	"crypto/md5"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	fage "filippo.io/age"
	"github.com/codingric/moneyman/pkg/age"
	"github.com/rapidloop/skv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	ageKey     *fage.X25519Identity
	httpClient *http.Client
	skvStore   *skv.KVStore
)

func init() {
	httpClient = &http.Client{}
	httpClient.Transport = otelhttp.NewTransport(http.DefaultTransport)
}

func initStore() {
	var err error
	skvStore, err = skv.Open(viper.GetString("store"))
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to init KV store")
	}
}

type TwilioResponse struct {
	XMLName       xml.Name            `xml:"TwilioResponse"`
	RestException TwilioRestException `xml:"RestException"`
}

type TwilioRestException struct {
	XMLName  xml.Name `xml:"RestException"`
	Code     int64    `xml:"Code"`
	Message  string   `xml:"Message"`
	MoreInfo string   `xml:"MoreInfo"`
	Status   int64    `xml:"Status"`
}

type Settings struct {
	Sid     string   `mapstructure:"sid"`
	Token   string   `mapstructure:"token"`
	Mobiles []string `mapstructure:"mobiles"`
}

func Notify(message string, ctx context.Context) (sent int, err error) {
	var settings *Settings

	if err := viper.UnmarshalKey("notify", &settings, viper.DecodeHook(age.AgeHookFunc(ageKey))); err != nil || settings == nil {
		log.Error().Err(err).Msg("Unable to load notify config")
		return 0, fmt.Errorf("unable to load settings")
	}

	if skvStore == nil {
		initStore()
	}

	endpoint := "https://api.twilio.com/2010-04-01/Accounts/" + settings.Sid + "/Messages"

	for _, m := range settings.Mobiles {
		body := url.Values{
			"To":   []string{m},
			"From": []string{"Budget"},
			"Body": []string{message},
		}

		hash := fmt.Sprintf("%x", md5.Sum([]byte(body.Encode())))
		if err := skvStore.Get(hash, nil); err == nil {
			log.Info().Str("hash", hash).Msgf("Message already sent to %s", m)
			continue
		}
		skvStore.Put(hash, m)

		req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(body.Encode()))

		req.SetBasicAuth(settings.Sid, settings.Token)

		if viper.GetBool("dryrun") {
			log.Info().Msgf("(DRYRUN) SMS : %s", message)
			continue
		}
		//resp, err := otelhttp.NewTransport(http.DefaultTransport).RoundTrip(req)
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to make request")
			return sent, fmt.Errorf("failed to make request")
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case 201:
			log.Debug().Msg("Sent SMS Successfully")
			sent = sent + 1
			continue
		case 401:
			return sent, errors.New("authentication failure")
		case 400:
			var resp_xml TwilioResponse
			var resp_body []byte
			resp_body, _ = io.ReadAll(resp.Body)
			err = xml.Unmarshal(resp_body, &resp_xml)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to read response")
				return sent, fmt.Errorf("failed to read responses")
			}
			return sent, errors.New(resp_xml.RestException.Message)
		default:
			return sent, errors.New("twilio responded with failure")
		}
	}
	return sent, nil
}
