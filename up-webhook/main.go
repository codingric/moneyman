package main

import (
	"fmt"
	"os"
	"path"

	"github.com/ian-kent/go-log/layout"
	"github.com/ian-kent/go-log/levels"
	"github.com/ian-kent/go-log/log"

	"github.com/spf13/viper"
)

var logger = log.Logger()

func init() {
	layout.DefaultTimeLayout = "2006-01-02 15:04:05"
	logger.Appender().SetLayout(layout.Pattern("%d %p %m"))
	logger.SetLevel(levels.INFO)
}

func main() {
	if err := Configure(); err != nil {
		CrashAndBurn(err.Error())
	}
	if err := RunWebhook(); err != nil {
		CrashAndBurn(err.Error())
	}

}

func CrashAndBurn(message string) {
	logger.Fatal("Fatal: %s", message)
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
			logger.Warn("Config not found - using defaults")
			return nil
		} else {
			logger.Error("Failed to load config")
		}
		return err
	}
	logger.SetLevel(log.Stol(viper.GetString("log_level")))
	logger.Debug("Config loaded `%s`", viper.ConfigFileUsed())

	return nil
}
