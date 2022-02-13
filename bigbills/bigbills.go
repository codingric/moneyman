package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/spf13/viper"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
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

func GetBigBillsRange() (*sheets.ValueRange, error) {
	ctx := context.Background()
	srv, err := sheets.NewService(
		ctx,
		option.WithCredentialsJSON(
			[]byte(viper.GetString("credentials")),
		),
		option.WithScopes(sheets.SpreadsheetsScope),
	)

	if err != nil {
		log.Printf("Sheets client error: %v", err)
		return nil, err
	}

	spreadsheetId := viper.GetString("spreadsheet_id")
	readRange := viper.GetString("spreadsheet_range")
	r, e := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	return r, e
}

func (b *BigBills) Hydrate() error {

	resp, err := GetBigBillsRange()
	if err != nil {
		return err
	}

	if len(resp.Values) == 0 {
		if viper.GetBool("verbose") {
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
		if viper.GetBool("verbose") {
			log.Printf("%d rows loaded from spreadsheet", len(resp.Values))
		}
	}
	return nil
}

func (b *BigBills) GetLate() (result []LateBigBill, err error) {
	t := time.Now()
	for _, date := range b.Dates {
		p, err := time.Parse("2006-01-02", date.Date)
		if err != nil {
			// Issue parsing the date, log and skip
			log.Println(err)
			continue
		}
		if date.Paid != "" {
			// Already paid, skip!
			continue
		}
		if p.After(t) {
			// Future payments, we can stop now
			break
		}
		// If we are here this payment is late
		days := fmt.Sprintf("%d days", int(math.Round(t.Sub(p).Hours()/24)))
		result = append(result, LateBigBill{p.Format("02 Jan 06"), date.Amount, days})
	}
	return result, err
}
