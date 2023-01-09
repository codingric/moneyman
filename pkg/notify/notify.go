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

	"github.com/codingric/moneyman/pkg/age"
	"github.com/codingric/moneyman/pkg/tracing"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	httpClient  *http.Client
	redisClient *redis.Client
)

func init() {
	httpClient = &http.Client{}
	httpClient.Transport = otelhttp.NewTransport(http.DefaultTransport)
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // host:port of the redis server
		Password: "",               // no password set
		DB:       0,                // use default DB
	})
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
	ctx, span := tracing.NewSpan("pkg.notify", ctx)
	defer span.End()

	var settings *Settings

	if err := viper.UnmarshalKey("notify", &settings, viper.DecodeHook(age.AgeHookFunc(age.AgeKey))); err != nil || settings == nil {
		log.Error().Err(err).Msg("Unable to load notify config")
		span.RecordError(err, trace.WithAttributes(
			attribute.String("message", message),
		))
		span.SetStatus(codes.Error, "Unable to load settings")
		return 0, fmt.Errorf("unable to load settings")
	}

	endpoint := "https://api.twilio.com/2010-04-01/Accounts/" + settings.Sid + "/Messages"

	for _, m := range settings.Mobiles {
		body := url.Values{
			"To":   []string{m},
			"From": []string{"Budget"},
			"Body": []string{message},
		}

		hash := generateHash(body.Encode())
		_, err := redisClient.Get(ctx, hash).Result()
		if err != redis.Nil {
			log.Info().Str("hash", hash).Msgf("Message already sent to %s", m)
			span.AddEvent("Message already sent", trace.WithAttributes(
				attribute.String("message", message),
				attribute.String("number", m),
				attribute.String("hash", hash),
			))
			continue
		}
		redisClient.Set(ctx, hash, m, 3600*24)

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
			span.RecordError(err, trace.WithAttributes(
				attribute.String("message", message),
			))
			span.SetStatus(codes.Error, "failed to make request")
			return sent, fmt.Errorf("failed to make request")
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case 201:
			log.Debug().Msg("Sent SMS Successfully")
			sent = sent + 1
			span.AddEvent("Sent message successfully", trace.WithAttributes(
				attribute.String("message", message),
				attribute.String("number", m),
			))
			continue
		case 401:
			span.RecordError(err, trace.WithAttributes(
				attribute.String("message", message),
			))
			span.SetStatus(codes.Error, "authentication failure")
			return sent, errors.New("authentication failure")
		case 400:
			var resp_xml TwilioResponse
			var resp_body []byte
			resp_body, _ = io.ReadAll(resp.Body)
			err = xml.Unmarshal(resp_body, &resp_xml)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to read response")
				span.RecordError(err, trace.WithAttributes(
					attribute.String("message", message),
				))
				span.SetStatus(codes.Error, "failed to read responses")
				return sent, fmt.Errorf("failed to read responses")
			}
			span.SetStatus(codes.Error, resp_xml.RestException.Message)
			return sent, errors.New(resp_xml.RestException.Message)
		default:
			span.SetStatus(codes.Error, "twilio responded with failure")
			return sent, errors.New("twilio responded with failure")
		}
	}
	return sent, nil
}

func generateHash(payload string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(payload)))
}
