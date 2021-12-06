package models

import "time"

type Transaction struct {
	ID          uint   `json:"id" gorm:"primary_key"`
	Description string `json:"description"`
	Amount      int64  `json:"amount"`
	CreatedAt   time.Time
}
