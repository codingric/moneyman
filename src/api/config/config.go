package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func Init() {
	viper.SetConfigName("config.toml") // name of config file (without extension)
	viper.SetConfigType("toml")        // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")           // optionally look for config in the working directory
	err := viper.ReadInConfig()        // Find and read the config file
	if err != nil {                    // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	//viper.Debug()
}
