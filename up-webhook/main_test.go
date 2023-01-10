package main

import (
	"context"
	"errors"
	"os"
	"testing"

	"bou.ke/monkey"
	"github.com/codingric/moneyman/up-webhook/webhook"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.PanicLevel)
	os.Exit(m.Run())
}

func Test_Main(t *testing.T) {
	type setup struct {
		configure bool
		runserver bool
	}
	type expected struct {
		fatal bool
		logs  string
	}
	tests := []struct {
		name     string
		setup    setup
		expected expected
	}{
		{
			"FailedConfigure",
			setup{configure: true},
			expected{fatal: true, logs: "mock error"},
		},
		{
			"FailedRunServer",
			setup{runserver: true},
			expected{fatal: true, logs: "mock error"},
		},
		{
			"OK",
			setup{},
			expected{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fatal := false
			defer func() {
				if r := recover(); r != nil {
					fatal = true
					return
				}
			}()

			monkey.Patch(Configure, func() error {
				if test.setup.configure {
					return errors.New("mock error")
				}
				return nil
			})

			monkey.Patch(webhook.RunWebhook, func(context.Context) error {
				if test.setup.runserver {
					return errors.New("mock error")
				}
				return nil
			})

			main()

			assert.Equal(tt, test.expected.fatal, fatal)
			monkey.UnpatchAll()
		})
	}
}
