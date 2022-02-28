package main

import (
	"testing"

	"github.com/ian-kent/go-log/levels"
	"github.com/spf13/viper"
)

func Test_UpTransactionGet(t *testing.T) {
	logger.SetLevel(levels.ERROR)
	viper.Set("bearer", "up:yeah:demo")
	var trans UpTransaction
	trans.Get("4922b484-49da-4a51-89f6-9c26cf1b7116")
}
