package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	database_path = kingpin.Flag("database", "Backend database").Default("backend.db").Short('d').String()
	verbose       = kingpin.Flag("verbose", "Verbosity").Short('v').Bool()
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	kingpin.Parse()

	if *verbose {
		log.Printf("Database: %s", *database_path)
	}

	// Connect to database
	ConnectDatabase(*database_path, *verbose)

	// Run the server
	setupServer(true).Run()
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
	return r
}
