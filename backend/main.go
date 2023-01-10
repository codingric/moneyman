package main

import (
	"context"
	"time"

	"github.com/codingric/moneyman/pkg/tracing"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"

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
	shutdown, tpErr := tracing.InitTraceProvider("backend")
	defer shutdown()

	if tpErr != nil {
		log.Fatal().Err(tpErr)
	}

	tracer := otel.Tracer("")

	_, span := tracer.Start(ctx, "main")
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
	r.GET("/transaction/:id", FindTransaction)
	r.POST("/transactions", CreateTransaction)

	return r
}
