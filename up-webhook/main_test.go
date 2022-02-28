package main

import (
	"errors"
	"os"
	"testing"

	"bou.ke/monkey"
	"github.com/ian-kent/go-log/levels"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logger.SetLevel(levels.FATAL)
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

			monkey.Patch(CrashAndBurn, func(m string) {
				assert.Equal(tt, test.expected.logs, m)
				assert.True(tt, true, test.expected.fatal)
				panic(m)
			})

			monkey.Patch(Configure, func() error {
				if test.setup.configure {
					return errors.New("mock error")
				}
				return nil
			})

			monkey.Patch(RunWebhook, func() error {
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
