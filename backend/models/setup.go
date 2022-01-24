package models

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB
var Debug bool

func ConnectDatabase(path string, debug bool) {

	var database *gorm.DB
	var err error

	Debug = debug

	if debug {
		database, err = gorm.Open(sqlite.Open(path))
	} else {
		database, err = gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	}

	if err != nil {
		panic("Failed to connect to database!")
	}

	database.AutoMigrate(&Account{})
	database.AutoMigrate(&Transaction{})

	if debug {
		DB = database.Debug()
	} else {
		DB = database
	}
}
