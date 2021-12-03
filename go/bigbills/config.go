package main

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type Config struct {
	sid   string `yaml:"sid"`
	token string `yaml:"token"`
	//mobiles []string `yaml:"mobiles"`
}

func LoadConfig() map[interface{}]interface{} {
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	m := make(map[interface{}]interface{})
	yaml.Unmarshal(data, &m)
	return m
}
