package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"google.golang.org/api/option"
	sheets "google.golang.org/api/sheets/v4"
)

type BigBillDate struct {
	Date   string `json:"date"`
	Amount string `json:"amount"`
	Paid   string `json:"paid"`
}

type BigBills struct {
	Dates []BigBillDate `json:"dates"`
}

type LateBigBill struct {
	Date   string `json:"date"`
	Amount string `json:"amount"`
	Days   string `json:"days"`
}

func (b *BigBills) Hydrate(config AppConfig) {
	ctx := context.Background()

	srv, err := sheets.NewService(ctx, option.WithCredentialsJSON([]byte(config.Credentials)), option.WithScopes(sheets.SpreadsheetsScope))

	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	// Prints the names and majors of students in a sample spreadsheet:
	// https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit
	spreadsheetId := config.SpreadsheetId
	readRange := config.SpreadsheetRange
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	if len(resp.Values) == 0 {
		if *verbose {
			log.Printf("No data loaded from spreadsheet")
		}
	} else {
		for _, row := range resp.Values {
			// Print columns A and E, which correspond to indices 0 and 4.
			if len(row) > 3 {
				b.Dates = append(b.Dates, BigBillDate{strings.Trim(row[0].(string), " "), row[1].(string), row[3].(string)})
			} else {
				b.Dates = append(b.Dates, BigBillDate{strings.Trim(row[0].(string), " "), row[1].(string), ""})
			}
		}
		if *verbose {
			log.Printf("%d rows loaded from spreadsheet", len(resp.Values))
		}
	}
}

func (b *BigBills) GetLate() (result []LateBigBill) {
	t := time.Now()
	for _, date := range b.Dates {
		p, err := time.Parse("2006-01-02", date.Date)
		if err != nil {
			log.Println(err)
			continue
		}
		if date.Paid != "" {
			continue
		}
		if p.After(t) {
			break
		}
		//log.Printf("%v %v\n", p.Format("02 Jan 06"), t.Sub(p))
		days := fmt.Sprintf("%d days", int(math.Round(t.Sub(p).Hours()/24)))
		result = append(result, LateBigBill{p.Format("02 Jan 06"), date.Amount, days})
	}
	return result
}
