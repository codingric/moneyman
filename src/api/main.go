package main

import (
	"moneyman/config"
	"moneyman/handlers"
	"moneyman/models"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	config.Init()
	models.InitDB()
}

func main() {
	http.HandleFunc("/healthz/ready", handlers.ReadyHandler)
	http.HandleFunc("/authenticate", handlers.AuthenticateHandler)
	http.HandleFunc("/api/v1/budget", handlers.ModelHandler[*models.BudgetModelFactory])
	http.HandleFunc("/api/v1/account", handlers.ModelHandler[*models.AccountModelFactory])
	http.HandleFunc("/api/v1/user", handlers.ModelHandler[*models.UserModelFactory])
	http.HandleFunc("/api/v1/collection", handlers.ModelHandler[*models.CollectionModelFactory])
	log.Infof("Listening on port %s", viper.GetString("server.listen_port"))
	log.Fatal(http.ListenAndServe(":"+viper.GetString("server.listen_port"), nil))
}
