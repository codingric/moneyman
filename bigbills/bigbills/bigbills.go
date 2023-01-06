package bigbills

import (
	"bigbills/config"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/codingric/moneyman/pkg/tracing"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Settings struct {
	SpreadsheetId    string `mapstructure:"spreadsheet_id"`
	SpreadsheetRange string `mapstructure:"spreadsheet_range"`
	Credentials      string `mapstructure:"credentials"`
	AccountId        string `mapstructure:"account_id"`
	Transactions     string `mapstructure:"transactions"`
}

var (
	sheetsService *sheets.Service
	httpClient    *http.Client
	settings      *Settings
)

func initAll() {
	if httpClient != nil {
		return
	}
	httpClient = &http.Client{}
	if err := config.Unmarshal("bigbills", &settings); err != nil {
		log.Fatal().Err(err)
	}
	var err error
	sheetsService, err = sheets.NewService(
		context.Background(),
		option.WithCredentialsJSON(
			[]byte(settings.Credentials),
		),
		option.WithScopes(sheets.SpreadsheetsScope),
	)
	if err != nil {
		log.Error().Msgf("Sheets client error: %v", err)
	}
}

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

func (b *BigBills) CheckLate(ctx context.Context) (string, error) {
	ctx, span := tracing.NewSpan("bigbills.checklate", ctx)
	defer span.End()
	if err := b.Hydrate(ctx); err != nil {
		return "", err
	}

	late, err := b.GetLate(ctx)
	if err != nil {
		return "", err
	}

	if len(late) > 0 {
		log.Debug().Msgf("%d overdue BigBills detected.\n", len(late))
		message := "Need to move BigBills:"
		for _, detail := range late {
			message = fmt.Sprintf("%s\n$%0.2f from %d days ago", message, detail.Amount, detail.Days)
		}
		log.Info().Msg(message)
		return message, nil
	} else {
		log.Debug().Msgf("No overdue BigBills detected.")
	}
	return "", nil

}

func (b *BigBills) Hydrate(ctx context.Context) error {
	ctx, span := tracing.NewSpan("bigbills.hydrate", ctx)
	defer span.End()
	if sheetsService == nil {
		initAll()
	}

	ctx, gspan := tracing.NewSpan("Spreadsheets.Values.Get", ctx)
	resp, err := sheetsService.Spreadsheets.Values.Get(
		settings.SpreadsheetId,
		settings.SpreadsheetRange,
	).Do()
	gspan.End()

	if err != nil {
		log.Error().Msgf("Unable to get GetBigBillsRange: `%s`", err.Error())
		return err
	}

	if len(resp.Values) == 0 {
		log.Debug().Msgf("No data loaded from spreadsheet")
	} else {
		for n, row := range resp.Values {

			datum := BigBillDate{}
			date, err := time.Parse("2006-01-02", strings.Trim(row[0].(string), " "))
			if err != nil {
				log.Error().Str("range", findcell(resp.Range, n, true)).Msgf("Unable to parse date: `%s`", err.Error())
				continue
			}
			datum.Date = date
			amount, _ := strconv.ParseFloat(strings.Trim(row[1].(string), "$"), 64)
			datum.Amount = amount
			datum.Row = n

			if len(row) > 3 {
				paid, err := time.Parse("2006-01-02", strings.Trim(row[3].(string), " "))
				if err != nil {
					log.Error().Str("range", findcell(resp.Range, n, true)).Msgf("Unable to parse paid date: `%s`", err.Error())
				}
				datum.Paid = &paid
			}
			b.Dates = append(b.Dates, datum)
		}
		log.Debug().Msgf("%d rows loaded from spreadsheet", len(resp.Values))
		span.SetAttributes(attribute.Int("loaded.rows", len(resp.Values)))
	}
	return nil
}

func (b *BigBills) GetLate(ctx context.Context) (result []LateBigBill, err error) {
	ctx, span := tracing.NewSpan("bigbills.getlate", ctx)
	defer span.End()
	for _, payment := range b.Dates {

		future, err := payment.InFuture()
		if err != nil {
			log.Error().Err(err).Msgf("Unable to determine if %s is in the future", payment.Date.String())
			continue
		}

		if future {
			// Future payments, we can stop now
			break
		}

		paid, err := payment.Repaid(ctx)
		if err != nil {
			log.Error().Err(err).Msgf("Unable to determine if %s has been paid", payment.Date.String())
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

func (p *BigBillDate) Repaid(ctx context.Context) (paid bool, err error) {
	ctx, span := tracing.NewSpan("bigbilldate.repaid", ctx)
	defer span.End()
	if p.Paid != nil {
		paid = true
		return
	}
	paid, err = p.CheckRepayments(ctx)
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

func (p *BigBillDate) CheckRepayments(ctx context.Context) (paid bool, err error) {
	ctx, span := tracing.NewSpan("bigbillsdate.checkrepayments", ctx)
	defer span.End()
	var (
		req  *http.Request
		resp *http.Response
	)
	url := fmt.Sprintf(
		"%s?amount=-%0.2f&account=%s&created__gt=%s",
		settings.Transactions,
		p.Amount,
		settings.AccountId,
		p.Date.Format("2006-01-02T00:00:00"),
	)

	req, _ = http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err = httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Str("Url", url)
		return false, err
	}
	rb, _ := io.ReadAll(resp.Body)

	log.Trace().Str("url", url).Str("method", "GET").Int("status_code", resp.StatusCode)

	var result APIResponse
	json.Unmarshal(rb, &result)
	paid = len(result.Data) == 1
	if paid {
		p.UpdatePaid(result.Data[0].Created, ctx)
	}
	return
}

func (p *BigBillDate) UpdatePaid(paid time.Time, ctx context.Context) (err error) {
	ctx, span := tracing.NewSpan("bigbilldate.updatepaid", ctx)
	defer span.End()
	p.Paid = &paid

	spreadsheetId := settings.SpreadsheetId
	values := [][]interface{}{
		{paid.Format("02/01/2006")},
	}
	updateRange := findcell(settings.SpreadsheetRange, p.Row, false)
	ctx, gspan := tracing.NewSpan("Spreadsheets.Values.Update", ctx)
	gspan.SetAttributes(
		attribute.String("range", updateRange),
		attribute.String("value", paid.Format("02/01/2006")),
	)
	_, e := sheetsService.Spreadsheets.Values.Update(spreadsheetId, updateRange, &sheets.ValueRange{Values: values}).ValueInputOption("USER_ENTERED").Do()
	gspan.End()
	return e
}

func findcell(_range string, index int, row bool) string {
	finder := regexp.MustCompile(`(.*[!])?([A-Z]+)([\d]+)[:]?([A-Z]+)?`)
	matches := finder.FindAllStringSubmatch(_range, -1)[0]
	start, _ := strconv.Atoi(matches[3])
	if row && matches[4] != "" {
		return fmt.Sprintf(
			"%s%s%d:%s%d",
			matches[1],
			matches[2],
			start+index,
			matches[4],
			start+index,
		)
	}
	if matches[4] != "" {
		return fmt.Sprintf(
			"%s%s%d",
			matches[1],
			matches[4],
			start+index,
		)
	}
	return fmt.Sprintf(
		"%s%s%d",
		matches[1],
		matches[2],
		start+index,
	)

}

func (p *BigBillDate) DaysLate() int {
	return int(math.Round(time.Since(p.Date).Hours() / 24))
}

func (p *BigBillDate) InFuture() (future bool, err error) {
	future = p.Date.After(time.Now())
	return
}
