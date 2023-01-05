package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"path/filepath"
	"regexp"
	"time"

	"github.com/codingric/moneyman/pkg/tracing"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Dict map[string]string

type Json map[string]interface{}

func main() {

	shutdown, err := tracing.InitTraceProvider("mailparser")
	if err != nil {
		log.Fatal().Err(err)
	}
	defer shutdown()

	Configure()

	http.Handle("/", otelhttp.NewHandler(http.HandlerFunc(Handler), "incoming.mail"))
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("OK")) })
	log.Info().Msg("Server started on port " + viper.GetString("port"))
	log.Fatal().Err(http.ListenAndServe(":"+viper.GetString("port"), nil)).Send()
}

func Configure() {
	viper.SetDefault("logging", map[string]string{"path": ".", "type": "none"})
	viper.SetDefault("port", "8081")

	flag.Bool("v", false, "Verbose")

	viper.RegisterAlias("verbose", "v")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/mailparser/")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Fatal().Err(err).Send()
	}

	log.Info().Msgf("Loaded config: %s", viper.ConfigFileUsed())
}

func Handler(w http.ResponseWriter, r *http.Request) {
	// ctx, span := tracing.NewSpan("http.incoming", r.Context())
	// span.SetAttributes()
	// defer span.End()

	dump, _ := httputil.DumpRequest(r, true)
	body, _ := ioutil.ReadAll(r.Body)
	hash := md5.Sum(dump)

	log.Debug().Msgf("Received request - %x\n", hash)

	data, err := ParseMessage(body, r.Context())
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(fmt.Sprintf("Error: %v", err)))
		log.Error().Err(err).Msgf("Failed to parse body")
		fn := filepath.Join(viper.GetString("logging.path"), fmt.Sprintf("%x.dump", hash))
		err := ioutil.WriteFile(fn, dump, 0644)
		if err != nil {
			log.Error().Err(err).Str("file", fn).Msg("Failed to save request dump")
		}
		log.Info().Msgf("Saved request: %s", fn)
		return
	}
	data["account"] = r.RequestURI[1:]

	log.Debug().Msgf("Parsed - %v\n", data)

	jb, _ := json.Marshal(data)

	if viper.GetString("webhook") != "" {
		req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, viper.GetString("webhook"), bytes.NewReader(jb))
		req.Header.Add("Content-Type", "application/json")
		resp, post_err := otelhttp.NewTransport(http.DefaultTransport).RoundTrip(req)

		if post_err != nil {
			log.Error().Err(post_err).Msgf("Webhook failed: %s", viper.GetString("webhook"))
			w.WriteHeader(503)
			w.Write([]byte("Error"))
			return
		}

		rb, _ := io.ReadAll(resp.Body)
		log.Debug().Msgf("Webhook response: %s", string(rb))

	}
	w.Write([]byte("OK"))
	log.Info().Msg("Successfully processed")
	if viper.GetString("logging.type") == "all" {
		fn := filepath.Join(viper.GetString("logging.path"), fmt.Sprintf("%x.dump", hash))
		err := ioutil.WriteFile(fn, dump, 0644)
		if err != nil {
			log.Error().Err(err).Str("file", fn).Msg("Failed to save request dump")
			return
		}
		log.Info().Msgf("Saved request %s", fn)
	}
}

const remove = `<.*?>|\n+`
const spaces = `[ \t]+`

// This method uses a regular expresion to remove HTML tags.
func stripHtml(str string) string {
	r := regexp.MustCompile(remove)
	s := regexp.MustCompile(spaces)
	str = r.ReplaceAllString(str, "")
	return s.ReplaceAllString(str, " ")
}

func ParseMessage(body []byte, c context.Context) (data Dict, err error) {
	_, span := tracing.NewSpan("ParseMessage", c)
	defer span.End()

	if data == nil {
		data = make(Dict)
	}
	decoder := json.NewDecoder(bytes.NewBuffer(body))
	decoder.UseNumber()

	json_ := make(map[string]interface{})
	err = decoder.Decode(&json_)
	if err != nil {
		return
	}

	if _, ok := json_["html"]; !ok {
		err = errors.New("missing html")
		return
	}

	wording := stripHtml(json_["html"].(string))

	if _, ok := json_["headers"]; !ok {
		err = errors.New("missing headers")
		return
	}

	dt, _ := time.Parse(time.RFC1123Z, json_["headers"].(map[string]interface{})["date"].(string))
	data["created"] = dt.Format("2006-01-02T15:04:05Z07:00")

	for _, p := range viper.GetStringSlice("patterns") {
		r := regexp.MustCompile(p)
		n := r.SubexpNames()
		s := ""
		for i, m := range r.FindStringSubmatch(wording) {
			switch n[i] {
			case "":
				continue
			case "negative":
				s = "-"
				continue
			case "amount":
				data[n[i]] = s + m
				continue
			}
			data[n[i]] = m
		}
	}

	if _, k := data["amount"]; !k {
		log.Debug().Msgf("No matches: %s", wording)
		err = errors.New("no matches found")
		if viper.GetString("logging") == "failure" {
			hash := json_["envelope"].(map[string]interface{})["md5"].(string)
			if err := ioutil.WriteFile(hash+".json", body, 0644); err != nil {
				log.Error().Err(err).Msgf("Unable to save %s", hash+".json")
				return data, err
			}
			log.Printf("Saved failed request %s.json", hash)
		}
		return
	}

	return
}
