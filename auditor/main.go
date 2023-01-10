package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/codingric/moneyman/pkg/age"
	"github.com/codingric/moneyman/pkg/notify"
	"github.com/codingric/moneyman/pkg/tracing"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/teambition/rrule-go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var (
	httpClient *http.Client
)

func init() {
	httpClient = &http.Client{}
	httpClient.Transport = otelhttp.NewTransport(http.DefaultTransport)
}

func LoadConf(ctx context.Context) error {
	_, span := tracing.NewSpan("LoadConf", ctx)
	defer span.End()
	flag.Bool("dryrun", false, "Log instead of SMS")
	flag.String("a", "/etc/auditor/age.key", "Age key")
	flag.String("c", "/etc/auditor/config.yaml", "Config yaml")
	flag.String("l", zerolog.InfoLevel.String(), "trace|debug|info|error|fatal|panic|disabled")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	viper.RegisterAlias("loglevel", "l")
	viper.RegisterAlias("config", "c")
	viper.RegisterAlias("agekey", "a")

	cpath := viper.GetString("config")

	fpath, fname := filepath.Split(cpath)
	fex := filepath.Ext(fname)
	fname = strings.TrimSuffix(fname, fex)

	viper.SetConfigName(fname)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath(fpath)
	err := viper.ReadInConfig()
	if err != nil {
		log.Error().Err(err).Msgf("Unable to read config: `%s`", cpath)
		span.RecordError(err)
		return fmt.Errorf("unable to load config: `%s`", cpath)
	}

	lvl := zerolog.InfoLevel
	lvl, err = zerolog.ParseLevel(viper.GetString("loglevel"))
	if err != nil {
		log.Error().Err(err).Str("span_id", span.SpanContext().SpanID().String()).Msgf("Ignoring --loglevel")
		span.RecordError(err)
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	log.Info().Str("span_id", span.SpanContext().SpanID().String()).Msgf("Config loaded: `%s`", viper.ConfigFileUsed())

	if err := age.Init(viper.GetString("agekey")); err != nil {
		span.RecordError(err)
		return fmt.Errorf("unable to load agekey: `%s`", viper.GetString("agekey"))
	}

	return nil
}

func main() {
	ctx := context.Background()
	shutdown, tpErr := tracing.InitTraceProvider("auditor")
	if tpErr != nil {
		log.Fatal().Err(tpErr)
	}
	defer shutdown()

	ctx, span := tracing.NewSpan("main", ctx)
	log.Logger = log.Logger.With().Str("trace_id", span.SpanContext().TraceID().String()).Logger()
	defer span.End()

	if err := LoadConf(ctx); err != nil {
		log.Fatal().Msg("Failed to load configuration")
	}

	if err := RunChecks(ctx); err != nil {
		log.Fatal().Msgf("Failure: %s", err.Error())
	}
}

type Checks []Check

type Check struct {
	Name string `mapstructure:"name"`
	Type string `mapstructure:"type"`
}

type Repay struct {
	Name  string `mapstructure:"name"`
	Match string `mapstructure:"match"`
	Days  int    `mapstructure:"days"`
	From  string `mapstructure:"from"`
	To    string `mapstructure:"to"`
}

type Amount struct {
	Name      string  `mapstructure:"name"`
	Match     string  `mapstructure:"match"`
	Days      int     `mapstructure:"days"`
	Expected  float64 `mapstructure:"expected"`
	Threshold string  `mapstructure:"threshold"`
	Rrule     string  `mapstructure:"rrule"`
}

func RunChecks(ctx context.Context) error {
	ctx, span := tracing.NewSpan("RunChecks", ctx)

	defer span.End()
	var checks Checks

	viper.UnmarshalKey("checks", &checks)
	for i, check := range checks {

		log.Info().Msgf("Checking: %s (%s)", check.Name, check.Type)
		var message string
		switch check.Type {
		// case "repay":
		// 	var err error
		// 	var c Repay
		// 	viper.UnmarshalKey(fmt.Sprintf("checks.%d", i), &c)
		// 	message, err = CheckRepay(c)
		// 	if err != nil {
		// 		return err
		// 	}
		case "amount":
			var e error
			var c Amount
			viper.UnmarshalKey(fmt.Sprintf("checks.%d", i), &c)
			message, e = CheckAmount(c, ctx)
			if e != nil {
				return e
			}
		default:
			return errors.New("Invalid check type: " + check.Type)
		}
		if message != "" {
			_, err := notify.Notify(message, ctx)
			if err != nil {
				log.Error().Err(err).Msgf("Notify error: %s", err.Error())
				return err
			}
		}

	}
	return nil
}

type APIResponse struct {
	Data []APITransaction `json:"data"`
}

type APITransaction struct {
	Id          int64     `json:"id"`
	Description string    `json:"description"`
	Amount      float32   `json:"amount"`
	Account     int64     `json:"account"`
	Created     time.Time `json:"created"`
}

func QueryBackend(params map[string]string, ctx context.Context) (result APIResponse, err error) {
	ctx, span := tracing.NewSpan("QueryBackend", ctx)
	defer span.End()
	p := url.Values{}
	for k, v := range params {
		p.Set(k, v)
		span.SetAttributes(attribute.String(k, v))
	}
	url_ := fmt.Sprintf("%s?%s", viper.GetString("backend"), p.Encode())
	req, _ := http.NewRequestWithContext(ctx, "GET", url_, nil)
	log.Trace().Str("params", p.Encode()).Msg("Query backend")

	resp, err := httpClient.Do(req)
	//resp, err := otelhttp.NewTransport(http.DefaultTransport).RoundTrip(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query backend")
		return
	}
	defer resp.Body.Close()
	var resp_body []byte
	resp_body, _ = io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		err = errors.New(strings.ToLower(string(resp_body)))
		log.Error().Err(err).Int("status", resp.StatusCode).Msg("Query returned non 200")
		return
	}

	err = json.Unmarshal(resp_body, &result)
	if err != nil {
		log.Error().Err(err).Msg("Failed to pase query result")
		err = errors.New("failed not parse result")
		return
	}
	return
}

func CheckAmount(c Amount, ctx context.Context) (msg string, err error) {
	ctx, span := tracing.NewSpan("CheckAmount", ctx)
	defer span.End()
	var s strings.Builder
	json.NewEncoder(&s).Encode(c)
	span.SetAttributes(attribute.String("check", s.String()))

	past := time.Now().AddDate(0, 0, -c.Days)
	var expected time.Time
	if c.Rrule != "" {
		rr, err := rrule.StrToRRule(c.Rrule)
		if err != nil {
			log.Error().Err(err).Msgf("Unable to parse rrule '%s'", c.Rrule)
			span.RecordError(err)
			span.SetStatus(codes.Error, fmt.Sprintf("Unable to parse rrule '%s'", c.Rrule))
			return "", fmt.Errorf("rrule invalid")
		}

		expected = rr.Before(time.Now(), false)
		past = expected.AddDate(0, 0, -c.Days)

		if time.Now().After(past.AddDate(0, 0, c.Days*2)) {
			log.Info().Msgf("Skipping '%s' as outside of dates", c.Name)
			return "", nil
		}
	}

	thresh := 0.0
	params := map[string]string{
		"description__like": c.Match,
		"created__gt":       past.Format("2006-01-02T15:04:05"),
	}
	if c.Threshold == "" {
		params["amount__ne"] = fmt.Sprintf("%0.2f", c.Expected)
	} else if strings.Contains(c.Threshold, "%") {
		t, _ := strconv.ParseFloat(strings.Replace(c.Threshold, "%", "", 1), 64)
		thresh = math.Abs(c.Expected * (t / 100.0))
		params["amount__lt"] = fmt.Sprintf("%0.2f", c.Expected+thresh)
		params["amount__gt"] = fmt.Sprintf("%0.2f", c.Expected-thresh)
	} else {
		thresh, _ = strconv.ParseFloat(strings.Replace(c.Threshold, "$", "", 1), 64)
		params["amount__gt"] = fmt.Sprintf("%0.2f", c.Expected-math.Abs(thresh))
		params["amount__lt"] = fmt.Sprintf("%0.2f", c.Expected+math.Abs(thresh))
	}

	response, err := QueryBackend(params, ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query backend")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to query backend")
		return msg, err
	}

	if len(response.Data) > 0 {
		msg = "Unexpected amounts:"
		for _, t := range response.Data {
			msg = fmt.Sprintf(
				"%s\n%s %s for $%0.2f expecting $%0.2f",
				msg,
				t.Created.Format("Mon 2 Jan"),
				t.Description,
				math.Abs(float64(t.Amount)),
				math.Abs(c.Expected),
			)
		}
		return
	}

	if c.Rrule != "" && time.Now().After(expected) {
		msg = fmt.Sprintf(
			"Payment for %s ($%0.2f) overdue %d days",
			c.Name,
			math.Abs(c.Expected),
			int(time.Since(expected).Hours()/24),
		)
	}

	return
}

// past := time.Now().AddDate(0, 0, c.Days*-1)
// params := map[string]string{
// 	"description__like": c.Match,
// 	"created__gt":       past.Format("2006-01-02T15:04:05"),
// 	"amount__lt":        "0.00",
// }
// var result []string

// response, err := QueryBackend(params)
// if err != nil {
// 	return
// }

// for _, transaction := range response.Data {
// 	p := map[string]string{
// 		"created__gt":       transaction.Created.Format("2006-01-02T15:04:05"),
// 		"description__like": c.From,
// 		"amount":            fmt.Sprintf("%0.2f", transaction.Amount*-1),
// 	}

// 	rr, err := QueryBackend(p)
// 	if err != nil {
// 		return msg, err
// 	}

// 	if len(rr.Data) == 0 {
// 		m := fmt.Sprintf("$%0.2f from %s", transaction.Amount*-1, transaction.Created.Format("Mon 2 Jan"))
// 		result = append(result, m)
// 	}
// }

//	if len(response.Data) > 0 {
//		unpaid := len(result)
//		found := len(response.Data)
//		log.Info().Msgf("Transactions: %d/%d repaid", found-unpaid, found)
//	}
//
//	if len(result) > 0 {
//		msg = fmt.Sprintf("Move money from %s to %s:", c.From, c.To)
//		for _, row := range result {
//			msg = fmt.Sprintf("%s\n%s", msg, row)
//		}
//	}
func CheckRepay(c Repay) (msg string, err error) {
	return
}
