package models

import "time"

type Transaction struct {
	ID          uint      `json:"id" gorm:"primary_key"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	Account     int64     `json:"account"`
	Created     time.Time `json:"created"`
}
