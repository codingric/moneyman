package config

import (
	"bytes"
	"context"
	"encoding/base64"
	"flag"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"filippo.io/age"
	"github.com/codingric/moneyman/pkg/tracing"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	ageIdentity *age.X25519Identity
)

func Init(ctx context.Context) {
	_, span := tracing.NewSpan("load.config", ctx)
	defer span.End()
	flag.Bool("v", false, "Verbose")
	flag.String("c", "config.toml", "Config TOML")
	flag.String("a", "age.key", "Age private key")
	flag.Int("log-level", int(zerolog.ErrorLevel), "Log level")

	viper.RegisterAlias("verbose", "v")
	viper.RegisterAlias("config", "c")
	viper.RegisterAlias("agekey", "a")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	loadAgeIdentity()

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(filepath.Dir(viper.GetString("config")))
	viper.AddConfigPath("/etc/bigbills/")

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Fatal().Msg(err.Error())
	}

	log.Level(zerolog.InfoLevel)
	if viper.GetBool("verbose") {
		log.Level(zerolog.DebugLevel)
	}
	if viper.IsSet("log-level") {
		log.Level(zerolog.Level(viper.GetInt("log-level")))
	}
	log.Debug().Msgf("Config: `%s`", viper.ConfigFileUsed())
	// viper.Debug()
}

type Config struct {
	Notify struct {
		Sid     string
		Token   string
		Mobiles []string
	}
	Google struct {
		SpreadsheetId    string
		SpreadsheetRange string
		Credentials      string
		AccountId        string
	}
	Backend struct {
		Endpoint string
	}
}

func Unmarshal(key string, target interface{}) (err error) {
	err = viper.UnmarshalKey(key, &target, viper.DecodeHook(ageHookFunc()))
	return
}

func decodeAge(s string) string {
	enc := strings.TrimPrefix(s, "age:")
	eb, _ := base64.StdEncoding.DecodeString(enc)
	r := bytes.NewReader(eb)
	d, _ := age.Decrypt(r, ageIdentity)
	b := &bytes.Buffer{}
	io.Copy(b, d)
	return b.String()
}

func loadAgeIdentity() {
	log.Debug().Msgf("Loading age key: %s", viper.GetString("agekey"))
	b, err := os.ReadFile(viper.GetString("agekey")) // just pass the file name
	if err != nil {
		log.Fatal().Msgf("Failed to read age key: %s", err.Error())
	}
	re := regexp.MustCompile(`(s?)#.*\n`)
	c := re.ReplaceAll(b, nil)
	str := string(c) // convert content to a 'string'
	ageIdentity, err = age.ParseX25519Identity(strings.Trim(str, "\n"))
	if err != nil {
		log.Fatal().Msgf("Failed to load age key: %s", err.Error())
	}

}

func ageHookFunc() mapstructure.DecodeHookFuncType {
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

		return decodeAge(data.(string)), nil
	}
}
