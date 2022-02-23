package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type BigBillDate struct {
	Date   time.Time  `json:"date"`
	Amount float64    `json:"amount"`
	Paid   *time.Time `json:"paid"`
	Row    int
}

type BigBills struct {
	Dates []BigBillDate `json:"dates"`
}

type LateBigBill struct {
	Date   time.Time `json:"date"`
	Amount float64   `json:"amount"`
	Days   int       `json:"days"`
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
		for n, row := range resp.Values {

			datum := BigBillDate{}
			date, err := time.Parse("2006-01-02", strings.Trim(row[0].(string), " "))
			if err != nil {
				log.Printf("Unable to parse date: `%s`", err.Error())
				continue
			}
			datum.Date = date
			amount, _ := strconv.ParseFloat(strings.Trim(row[1].(string), "$"), 64)
			datum.Amount = amount
			datum.Row = n

			if len(row) > 3 {
				paid, _ := time.Parse("2006-01-02", strings.Trim(row[3].(string), " "))
				datum.Paid = &paid
			}
			b.Dates = append(b.Dates, datum)
		}
		if viper.GetBool("verbose") {
			log.Printf("%d rows loaded from spreadsheet", len(resp.Values))
		}
	}
	return nil
}

func (b *BigBills) GetLate() (result []LateBigBill, err error) {
	for _, payment := range b.Dates {

		future, err := payment.InFuture()
		if err != nil {
			log.Println(err)
			continue
		}

		if future {
			// Future payments, we can stop now
			break
		}

		paid, err := payment.Repaid()
		if err != nil {
			log.Println(err)
			continue
		}

		if paid {
			// Already paid, skip!
			continue
		}

		// If we are here this payment is late
		late := LateBigBill{
			payment.Date,
			payment.Amount,
			payment.DaysLate()}

		result = append(result, late)
	}
	return result, err
}

func (p *BigBillDate) Repaid() (paid bool, err error) {
	if p.Paid != nil {
		paid = true
		return
	}
	paid, err = p.CheckRepayments()
	return
}

type APIResponse struct {
	Data []APITransaction `json:"data"`
}

type APITransaction struct {
	Id          int64     `json:"id"`
	Description string    `json:"description"`
	Amount      float32   `json:"amount"`
	Account     int64     `json:"account"`
	Created     time.Time `json:"created"`
}

func (p *BigBillDate) CheckRepayments() (paid bool, err error) {
	url := fmt.Sprintf(
		"%s?amount=-%0.2f&account=%s&created__gt=%s",
		viper.GetString("backend"),
		p.Amount,
		viper.GetString("account_id"),
		p.Date.Format("2006-01-02T00:00:00"),
	)
	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	rb, _ := io.ReadAll(resp.Body)
	var result APIResponse
	json.Unmarshal(rb, &result)
	paid = len(result.Data) == 1
	p.UpdatePaid(result.Data[0].Created)
	return
}

func (p *BigBillDate) UpdatePaid(paid time.Time) (err error) {
	p.Paid = &paid
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
		return err
	}

	finder := regexp.MustCompile(`^(.*)[!]([A-Z]+)([\d]+)[:]([A-Z]+)$`)
	matches := finder.FindAllStringSubmatch(viper.GetString("spreadsheet_range"), -1)
	start, _ := strconv.Atoi(matches[0][3])

	spreadsheetId := viper.GetString("spreadsheet_id")
	updateRange := fmt.Sprintf("%s!%s%d", matches[0][1], matches[0][4], start+p.Row)
	values := [][]interface{}{
		{paid.Format("02/01/2006")},
	}

	_, e := srv.Spreadsheets.Values.Update(spreadsheetId, updateRange, &sheets.ValueRange{Values: values}).ValueInputOption("USER_ENTERED").Do()
	return e
}

func (p *BigBillDate) DaysLate() int {
	return int(math.Round(time.Since(p.Date).Hours() / 24))
}

func (p *BigBillDate) InFuture() (future bool, err error) {
	future = p.Date.After(time.Now())
	return
}
