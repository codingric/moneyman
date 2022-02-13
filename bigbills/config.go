package main

import (
	"flag"
	"log"

	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func LoadConfig() {
	flag.Bool("v", false, "Verbose")
	flag.String("c", "config.yaml", "Config.yaml")

	viper.RegisterAlias("verbose", "v")
	viper.RegisterAlias("config", "c")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(filepath.Dir(viper.GetString("config")))
	viper.AddConfigPath("/etc/bigbills/")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Fatalf("%v\n", err)
	}

	if viper.GetBool("verbose") {
		log.Println("Verbose: ON")
		log.Printf("Config: `%s`\n", viper.ConfigFileUsed())
	} else {
		log.Println("Verbose: OFF")
	}
}
