package config

import (
	"context"
	"flag"
	"path/filepath"

	"github.com/codingric/moneyman/pkg/age"
	"github.com/codingric/moneyman/pkg/tracing"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func Init(ctx context.Context) {
	_, span := tracing.NewSpan("load.config", ctx)
	defer span.End()
	flag.Bool("v", false, "Verbose")
	flag.String("c", "config.toml", "Config TOML")
	flag.String("a", "age.key", "Age private key")
	flag.String("s", "store.db", "KV store")
	flag.Int("log-level", int(zerolog.ErrorLevel), "Log level")

	viper.RegisterAlias("verbose", "v")
	viper.RegisterAlias("config", "c")
	viper.RegisterAlias("agekey", "a")
	viper.RegisterAlias("store", "s")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	age.Init(viper.GetString("agekey"))

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
	err = viper.UnmarshalKey(key, &target, viper.DecodeHook(age.AgeHookFunc(age.AgeKey)))
	return
}
