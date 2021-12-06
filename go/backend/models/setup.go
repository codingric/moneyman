package models

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var DB *gorm.DB

func ConnectDatabase(path string) {
	database, err := gorm.Open("sqlite3", path)

	if err != nil {
		panic("Failed to connect to database!")
	}

	database.AutoMigrate(&Account{})
	database.AutoMigrate(&Transaction{})

	DB = database
}
