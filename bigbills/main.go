package main

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

func main() {

	LoadConfig()
	var bills BigBills

	if err := bills.Hydrate(); err != nil {
		log.Fatalf("Failure: %s", err.Error())
	}

	if err := CheckLate(bills); err != nil {
		log.Fatalf("Failure: %s", err.Error())
	}
}

func CheckLate(b BigBills) error {
	late, err := b.GetLate()
	if err != nil {
		return err
	}

	if len(late) > 0 {
		if viper.GetBool("verbose") {
			log.Printf("%d overdue BigBills detected.\n", len(late))
		}
		message := "Need to move BigBills:"
		for _, detail := range late {
			message = fmt.Sprintf("%s\n$%0.2f from %d days ago", message, detail.Amount, detail.Days)
		}
		log.Print(message)
		_, err := Notify(message)
		if err != nil {
			return err
		}
	} else {
		if viper.GetBool("verbose") {
			log.Println("No overdue BigBills detected.")
		}
	}
	return nil
}
