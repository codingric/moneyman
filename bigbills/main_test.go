package main

import (
	"bigbills/bigbills"
	"bigbills/config"
	"context"
	"errors"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/codingric/moneyman/pkg/notify"
)

func Test(t *testing.T) {
	tests := []struct {
		name         string
		CheckLate    func(*bigbills.BigBills, context.Context) (string, error)
		Notify       func(string, context.Context) (int, error)
		expect_panic bool
	}{
		{
			"CheckLate error",
			func(bb *bigbills.BigBills, c context.Context) (string, error) {
				return "", errors.New("CheckLate error")
			},
			nil,
			true,
		},
		{
			"No Late",
			func(bb *bigbills.BigBills, c context.Context) (string, error) { return "", nil },
			nil,
			false,
		},
		{
			"Notify failed",
			func(bb *bigbills.BigBills, c context.Context) (string, error) { return "Late", nil },
			func(s string, c context.Context) (int, error) { return 0, errors.New("no credit") },
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(st *testing.T) {
			defer monkey.UnpatchAll()
			monkey.PatchInstanceMethod(reflect.TypeOf((*bigbills.BigBills)(nil)), "CheckLate", test.CheckLate)
			monkey.Patch(notify.Notify, test.Notify)
			monkey.Patch(config.Init, func(c context.Context) {})

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
