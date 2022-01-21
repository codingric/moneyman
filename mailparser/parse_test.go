package main

import (
	"bufio"
	"bytes"
	"github.com/spf13/viper"
	"log"
	"os"
	"testing"
)

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
	for k, _ := range a {
		if a[k] != b[k] {
			return false
		}
	}
	for k, _ := range b {
		if b[k] != a[k] {
			return false
		}
	}

	return true
}

func TestMain(m *testing.M) {
	Configure()
	viper.Set("verbose", true)
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

func TestDepositedFrom(t *testing.T) {
	RunValidation(
		"recently_spent",
		Dict{"created": "2022-01-21T08:33:13+11:00", "amount": "-5.00", "description": "TOSSD PL DO"},
		t,
	)
}

func TestRecentlySpent(t *testing.T) {
	RunValidation(
		"deposited_from",
		Dict{"created": "2022-01-21T01:58:02+11:00", "amount": "3.04", "description": "deposited into your Bills from your Spending"},
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
