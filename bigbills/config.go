package main

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type AppConfig struct {
	Sid              string   `yaml:"sid"`
	Token            string   `yaml:"token"`
	Mobiles          []string `yaml:"mobiles"`
	Credentials      string   `yaml:"credentials"`
	SpreadsheetId    string   `yaml:"spreadsheet_id"`
	SpreadsheetRange string   `yaml:"spreadsheet_range"`
}

func LoadConfig(path string) AppConfig {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	m := AppConfig{}
	yaml.Unmarshal(data, &m)
	return m
}
