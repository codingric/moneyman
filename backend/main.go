package main

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	database_path = kingpin.Flag("database", "Backend database").Default("backend.db").Short('d').String()
	verbose       = kingpin.Flag("verbose", "Verbosity").Short('v').Bool()
	port          = kingpin.Flag("port", "Port").Short('p').Default("8080").String()
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	kingpin.Parse()
	zerolog.DurationFieldUnit = time.Millisecond
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	ConnectDatabase(*database_path, *verbose)
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

	// Routes
	r.GET("/accounts", FindAccounts)
	r.GET("/accounts/:id", FindAccount)
	r.POST("/accounts", CreateAccount)
	r.PATCH("/accounts/:id", UpdateAccount)
	r.GET("/transactions", FindTransactions)
	r.POST("/transactions", CreateTransaction)

	return r
}
