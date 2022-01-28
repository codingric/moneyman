package main

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"testing"

	"github.com/spf13/viper"
)

func LoadConfig() {
	var config = []byte(`patterns:
- '(?P<negative>spent) \$(?P<amount>\d{1,}\.\d{2}) at (?P<description>[^.]+)'
- '(?P<negative>payment) of \$(?P<amount>\d{1,}\.\d{2}) to (?P<description>[^.]+)'
- '(?P<negative> )\$(?P<amount>\d{1,}\.\d{2}) was moved between your ING accounts, (?P<description>[^.]+)'
- '(?P<description>withdrawal) of(?P<negative> )\$(?P<amount>\d{1,}\.\d{2}) was made from your'
- '\$(?P<amount>\d{1,}\.\d{2}) (?P<description>deposit) has been made into your'
- 'that (?P<description>.*) deposited \$(?P<amount>\d{1,}\.\d{2}) into your'
- '\$(?P<amount>\d{1,}.\d{2}) has been (?P<description>deposited into your \w+ from your \w+)'`)

	viper.SetConfigType("yaml") // or viper.SetConfigType("YAML")

	viper.ReadConfig(bytes.NewBuffer(config))
	viper.RegisterAlias("verbose", "v")
	viper.Set("verbose", false)
}

func LoadFixture(name string) []byte {

	body := new(bytes.Buffer)
	path := "fixtures/" + name + ".dump"

	buf, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err = buf.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	snl := bufio.NewScanner(buf)
	for snl.Scan() {
		if snl.Text() == "" {
			break
		}
	}
	err = snl.Err()
	if err != nil {
		log.Fatal(err)
	}
	for snl.Scan() {
		body.Write([]byte(snl.Text()))
	}
	return body.Bytes()
}

func (a Dict) Equal(b Dict) bool {
	for k := range a {
		if a[k] != b[k] {
			return false
		}
	}
	for k := range b {
		if b[k] != a[k] {
			return false
		}
	}

	return true
}

func TestMain(m *testing.M) {
	LoadConfig()
	retCode := m.Run()
	os.Exit(retCode)
}

func RunValidation(name string, expected Dict, t *testing.T) {
	input := LoadFixture(name)
	result, _ := ParseMessage(input)

	if !expected.Equal(result) {
		t.Fatalf("Validation failed.\nExpected:\n%v\nResult:\n%v", expected, result)
	}
}

func TestMovedBetween(t *testing.T) {
	RunValidation(
		"moved_between",
		Dict{"created": "2022-01-21T01:58:02+11:00", "amount": "-3.04", "description": "from Spending to Bills"},
		t,
	)
}

func TestRecentlySpent(t *testing.T) {
	RunValidation(
		"recently_spent",
		Dict{"created": "2022-01-21T08:33:13+11:00", "amount": "-5.00", "description": "TOSSD PL DO"},
		t,
	)
}
func TestDepositedFrom(t *testing.T) {
	RunValidation(
		"deposited_from",
		Dict{"created": "2022-01-24T08:14:15+11:00", "amount": "71.34", "description": "deposited into your Spending from your Food"},
		t,
	)
}

func TestDepositNamed(t *testing.T) {
	RunValidation(
		"deposit_named",
		Dict{"created": "2022-01-21T11:25:21+11:00", "amount": "1.50", "description": "R O SALINAS"},
		t,
	)
}
