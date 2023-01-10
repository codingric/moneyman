package main

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/codingric/moneyman/pkg/tracing"
	"github.com/codingric/moneyman/up-webhook/webhook"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func main() {
	ctx := context.Background()
	shutdown, tpErr := tracing.InitTraceProvider(path.Base(os.Args[0]))
	if tpErr != nil {
		log.Fatal().Err(tpErr)
	}
	defer shutdown()
	ctx, span := tracing.NewSpan("main", ctx)
	log.Logger = log.Logger.With().Str("trace_id", span.SpanContext().TraceID().String()).Logger()
	defer span.End()

	if err := Configure(); err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}
	if err := webhook.RunWebhook(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}
}

func Configure() error {
	viper.SetDefault("log_level", "INFO")
	viper.SetDefault("port", 8080)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath(fmt.Sprintf("/etc/%s/", path.Base(os.Args[0])))
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Warn().Msg("Config not found - using defaults")
			return nil
		} else {
			log.Error().Msg("Failed to load config")
		}
		return err
	}
	//log.SetLevel().Msg(log.Stol(viper.GetString("log_level")))
	log.Debug().Msgf("Config loaded `%s`", viper.ConfigFileUsed())

	return nil
}
