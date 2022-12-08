package main

import (
	"bigbills/bigbills"
	"bigbills/config"
	"bigbills/notify"
)

func main() {
	config.Init()
	bills := &bigbills.BigBills{}

	message, err := bills.CheckLate()
	if err != nil {
		panic(err.Error())
	}

	if message != "" {
		if resp, e := notify.Notify(message); e != nil {
			panic(resp)
		}
	}
}
