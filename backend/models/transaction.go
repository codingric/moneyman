package models

import "time"

type Transaction struct {
	ID          uint      `json:"id" gorm:"primary_key"`
	Md5			string	  `json:"-" gorm:"unique"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	Account     int64     `json:"account"`
	Created     time.Time `json:"created"`
}
