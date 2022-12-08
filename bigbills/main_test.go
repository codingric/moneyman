package main

import (
	"bigbills/bigbills"
	"bigbills/config"
	"bigbills/notify"
	"errors"
	"reflect"
	"testing"

	"bou.ke/monkey"
)

func TestCheckLate(t *testing.T) {
	tests := []struct {
		name         string
		CheckLate    func(*bigbills.BigBills) (string, error)
		Notify       func(string) (string, error)
		expect_panic bool
	}{
		{
			"CheckLate error",
			func(bb *bigbills.BigBills) (string, error) { return "", errors.New("CheckLate error") },
			nil,
			true,
		},
		{
			"No Late",
			func(bb *bigbills.BigBills) (string, error) { return "", nil },
			nil,
			false,
		},
		{
			"Notify failed",
			func(bb *bigbills.BigBills) (string, error) { return "Late", nil },
			func(s string) (string, error) { return "No credit", errors.New("no credit") },
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(st *testing.T) {
			defer monkey.UnpatchAll()
			monkey.PatchInstanceMethod(reflect.TypeOf((*bigbills.BigBills)(nil)), "CheckLate", test.CheckLate)
			monkey.Patch(notify.Notify, test.Notify)
			monkey.Patch(config.Init, func() {})

			if test.expect_panic {
				defer func() {
					if r := recover(); r == nil {
						st.Errorf("Panic expected but didn't occur")
					}
				}()
			}

			main()

		})
	}
}
