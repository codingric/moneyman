package main

import (
	"fmt"
	"log"
)

func main() {
	var bills BigBills
	config := LoadConfig()
	bills.Hydrate()
	checkLate(bills, config)
}

func checkLate(b BigBills, c map[interface{}]interface{}) {
	late := b.GetLate()

	if len(late) > 0 {
		message := "Need to move BigBills:"
		for _, detail := range late {
			message = fmt.Sprintf("%s\n%s from %s ago", message, detail.Amount, detail.Days)
		}
		log.Print(message)
		_, err := Notify(message, c)
		if err != nil {
			panic(err)
		}
	}
}
