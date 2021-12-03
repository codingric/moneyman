package main

import (
	"fmt"
	"log"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	config_path = kingpin.Flag("config", "config.yaml").Default("config.yaml").Short('c').ExistingFile()
	creds_path  = kingpin.Flag("creds", "credentials.json").Default("credentials.json").Short('g').ExistingFile()
	verbose     = kingpin.Flag("verbose", "Verbose mode.").Short('v').Bool()
)

func main() {

	kingpin.Parse()

	var bills BigBills
	config := LoadConfig(*config_path)
	bills.Hydrate(*creds_path)
	checkLate(bills, config)
}

func checkLate(b BigBills, c map[interface{}]interface{}) {
	late := b.GetLate()

	if len(late) > 0 {
		if *verbose {
			log.Printf("%d overdue BigBills detected.\n", len(late))
		}
		message := "Need to move BigBills:"
		for _, detail := range late {
			message = fmt.Sprintf("%s\n%s from %s ago", message, detail.Amount, detail.Days)
		}
		log.Print(message)
		_, err := Notify(message, c)
		if err != nil {
			panic(err)
		}
	} else {
		if *verbose {
			log.Println("No overdue BigBills detected.")
		}
	}
}
