package main

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
)

func TestCheckLate(t *testing.T) {
	tests := []struct {
		name       string
		bills      BigBills
		getlate    string
		notify_msg string
		notify_err string
		err        string
	}{
		{
			"Empty",
			BigBills{},
			"",
			"",
			"",
			"",
		},
		{
			"GetLateErr",
			BigBills{},
			"Failure in GetLate",
			"",
			"",
			"Failure in GetLate",
		},
		{
			"Notify",
			BigBills{[]BigBillDate{{"2000-01-02", "100.00", ""}}},
			"",
			"Need to move BigBills:\n100.00 from 2 days ago",
			"",
			"",
		},
		{
			"NotifyErr",
			BigBills{[]BigBillDate{{"2000-01-02", "100.00", ""}}},
			"",
			"Need to move BigBills:\n100.00 from 2 days ago",
			"Notify failure",
			"Notify failure",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(st *testing.T) {
			monkey.Patch(time.Now, func() time.Time {
				return time.Date(2000, time.January, 4, 0, 0, 0, 0, time.UTC)
			})
			defer monkey.Unpatch(time.Now)

			if test.getlate != "" {
				var b *BigBills
				monkey.PatchInstanceMethod(reflect.TypeOf(b), "GetLate", func(*BigBills) ([]LateBigBill, error) {
					return nil, errors.New(test.getlate)
				})
				defer monkey.UnpatchInstanceMethod(reflect.TypeOf(b), "GetLate")
			}

			monkey.Patch(Notify, func(m string) (s string, e error) {
				assert.Equal(st, test.notify_msg, m)
				if test.notify_err != "" {
					e = errors.New(test.notify_err)
				}
				return
			})
			defer monkey.Unpatch(Notify)

			err := CheckLate(test.bills)

			if test.err == "" {
				assert.Nil(st, err)
			} else {
				assert.NotNil(st, err)
				if err != nil {
					assert.Equal(st, test.err, err.Error())
				}
			}
		})
	}
}
