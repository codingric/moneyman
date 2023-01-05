package main

import (
	"context"
	"time"

	"github.com/codingric/moneyman/backend/tracing"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/gin-gonic/gin"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	database_path = kingpin.Flag("database", "Backend database").Default("backend.db").Short('d').String()
	verbose       = kingpin.Flag("verbose", "Verbosity").Short('v').Bool()
	port          = kingpin.Flag("port", "Port").Short('p').Default("8080").String()
)

func main() {
	ctx := context.Background()
	tp, tpErr := tracing.AspectoTraceProvider()
	defer tp.Shutdown(ctx)

	if tpErr != nil {
		log.Fatal().Err(tpErr)
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	tracer := otel.Tracer("gorm.io/plugin/opentelemetry")

	_, span := tracer.Start(ctx, "root")
	defer span.End()

	kingpin.Parse()
	zerolog.DurationFieldUnit = time.Millisecond
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	ConnectDatabase(ctx, *database_path, *verbose)
	if *verbose {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}

	// Connect to database

	// Run the server
	log.Info().Msgf("Server running on port %s", *port)
	if err := setupServer(*verbose).Run(":" + *port); err != nil {
		log.Fatal().Err(err).Msg("Server Error")
	}
}

func setupServer(debug bool) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(otelgin.Middleware("backend"))

	// Routes
	r.GET("/accounts", FindAccounts)
	r.GET("/accounts/:id", FindAccount)
	r.POST("/accounts", CreateAccount)
	r.PATCH("/accounts/:id", UpdateAccount)
	r.GET("/transactions", FindTransactions)
	r.POST("/transactions", CreateTransaction)

	return r
}
