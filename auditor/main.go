package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/codingric/moneyman/pkg/tracing"
	"github.com/mitchellh/mapstructure"
	"github.com/rapidloop/skv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/teambition/rrule-go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

var (
	ageKey     *age.X25519Identity
	httpClient *http.Client
	skvStore   *skv.KVStore
)

func init() {
	httpClient = &http.Client{}
}

func initStore() {
	var err error
	skvStore, err = skv.Open(viper.GetString("store"))
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to init KV store")
	}
}

func LoadConf(ctx context.Context) {
	ctx, span := otel.Tracer("auditor").Start(ctx, "LoadConf")
	defer span.End()
	flag.Bool("dryrun", false, "Log instead of SMS")
	flag.String("agekey", "/etc/auditor/age.key", "Age key")
	flag.String("config", "/etc/auditor/config.yaml", "Config yaml")
	flag.String("loglevel", zerolog.ErrorLevel.String(), "trace|debug|info|error|fatal|panic|disabled")
	flag.String("store", "/var/run/auditor/notifications.db", "Location for KV store")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

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
		log.Fatal().Err(err).Msgf("Unable to read config: `%s`", cpath)
	}

	lvl := zerolog.InfoLevel
	lvl, err = zerolog.ParseLevel(viper.GetString("loglevel"))
	if err != nil {
		log.Error().Err(err).Msgf("Ignoring --loglevel")
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	log.Info().Msgf("Config loaded: `%s`", viper.ConfigFileUsed())

	b, err := os.ReadFile(viper.GetString("agekey")) // just pass the file name
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to open age key: `%s`", viper.GetString("agekey"))
	}
	ageKey, err = loadAgeKey(b)
	if err != nil {
		log.Fatal().Msgf("Unable to load age key: %s", err.Error())
	}
}

func loadAgeKey(b []byte) (a *age.X25519Identity, e error) {
	// Remove comments
	re := regexp.MustCompile(`(s?)#.*\n`)
	c := re.ReplaceAll(b, nil)
	str := string(c) // convert content to a 'string'

	a, e = age.ParseX25519Identity(strings.Trim(str, "\n"))
	return
}

func decodeAge(s string, a *age.X25519Identity) string {
	enc := strings.TrimPrefix(s, "age:")
	eb, _ := base64.StdEncoding.DecodeString(enc)
	r := bytes.NewReader(eb)
	d, _ := age.Decrypt(r, a)
	b := &bytes.Buffer{}
	io.Copy(b, d)
	return b.String()
}

func ageHookFunc(a *age.X25519Identity) mapstructure.DecodeHookFuncType {
	// Wrapped in a function call to add optional input parameters (eg. separator)
	return func(
		f reflect.Type, // data type
		t reflect.Type, // target data type
		data interface{}, // raw data
	) (interface{}, error) {

		// Check if the data type matches the expected one
		if f.Kind() != reflect.String {
			return data, nil
		}

		// Check if the target type matches the expected one
		if t.Kind() != reflect.String {
			return data, nil
		}

		if !strings.HasPrefix(data.(string), "age:") {
			return data, nil
		}

		return decodeAge(data.(string), a), nil
	}
}

func main() {
	ctx := context.Background()
	tp, tpErr := tracing.AspectoTraceProvider("auditor")
	if tpErr != nil {
		log.Fatal().Err(tpErr)
	}
	defer tp.Shutdown(ctx)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	tracer := otel.Tracer("auditor")

	ctx, span := tracer.Start(ctx, "main")
	defer span.End()

	LoadConf(ctx)

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
	ctx, span := otel.Tracer("auditor").Start(ctx, "RunChecks")
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
			_, err := Notify(message)
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
	ctx, span := otel.Tracer("auditor").Start(ctx, "QueryBackend")
	defer span.End()
	p := url.Values{}
	for k, v := range params {
		p.Set(k, v)
	}
	url_ := fmt.Sprintf("%s?%s", viper.GetString("backend"), p.Encode())
	req, _ := http.NewRequestWithContext(ctx, "GET", url_, nil)
	log.Trace().Str("params", p.Encode()).Msg("Query backend")

	//resp, err := httpClient.Do(req)
	resp, err := otelhttp.NewTransport(http.DefaultTransport).RoundTrip(req)
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
		err = errors.New("Failed not parse result")
		return
	}
	return
}

func CheckAmount(c Amount, ctx context.Context) (msg string, err error) {
	ctx, span := otel.Tracer("auditor").Start(ctx, "CheckAmount")
	defer span.End()

	past := time.Now().AddDate(0, 0, -c.Days)
	var expected time.Time
	if c.Rrule != "" {
		rr, err := rrule.StrToRRule(c.Rrule)
		if err != nil {
			log.Error().Err(err).Msgf("Unable to parse rrule '%s'", c.Rrule)
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

func Notify(message string) (sent int, err error) {
	var settings *Settings

	if err := viper.UnmarshalKey("notify", &settings, viper.DecodeHook(ageHookFunc(ageKey))); err != nil || settings == nil {
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

		req, _ := http.NewRequest("POST", endpoint, strings.NewReader(body.Encode()))
		req.SetBasicAuth(settings.Sid, settings.Token)

		if viper.GetBool("dryrun") {
			log.Info().Msgf("(DRYRUN) SMS : %s", message)
			continue
		}
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
