package main

import (
	"bigbills/bigbills"
	"bigbills/config"
	"context"

	"github.com/codingric/moneyman/pkg/notify"
	"github.com/codingric/moneyman/pkg/tracing"
	"github.com/rs/zerolog/log"
)

func main() {
	shutdown, err := tracing.InitTraceProvider("bigbills")
	if err != nil {
		log.Fatal().Err(err)
	}
	defer shutdown()

	ctx := context.Background()

	ctx, span := tracing.NewSpan("main", ctx)
	defer span.End()

	config.Init(ctx)
	bills := &bigbills.BigBills{}

	message, err := bills.CheckLate(ctx)
	if err != nil {
		panic(err.Error())
	}

	if message != "" {
		if resp, e := notify.Notify(message, ctx); e != nil {
			panic(resp)
		}
	}
}
