package main

import (
	"log"

	"github.com/codingric/moneyman/go/backend/controllers"
	"github.com/codingric/moneyman/go/backend/models"

	"github.com/gin-gonic/gin"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	database_path = kingpin.Flag("database", "Backend database").Default("backend.db").Short('d').String()
	verbose       = kingpin.Flag("verbose", "Verbosity").Short('v').Bool()
)

func main() {
	kingpin.Parse()
	// gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	if *verbose {
		log.Printf("Database: %s", *database_path)
	}

	// Connect to database
	models.ConnectDatabase(*database_path)

	// Routes
	r.GET("/accounts", controllers.FindAccounts)
	r.GET("/accounts/:id", controllers.FindAccount)
	r.POST("/accounts", controllers.CreateAccount)
	r.PATCH("/accounts/:id", controllers.UpdateAccount)
	r.DELETE("/accounts/:id", controllers.DeleteAccount)

	r.GET("/transactions", controllers.FindTransactions)
	r.GET("/transactions/:id", controllers.FindTransaction)
	r.POST("/transactions", controllers.CreateTransaction)

	// Run the server
	r.Run()
}
