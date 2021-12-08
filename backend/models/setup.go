package models

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase(path string) {
	database, err := gorm.Open(sqlite.Open(path))

	if err != nil {
		panic("Failed to connect to database!")
	}

	database.AutoMigrate(&Account{})
	database.AutoMigrate(&Transaction{})

	DB = database.Debug()
}
