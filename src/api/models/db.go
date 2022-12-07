package models

import (
	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitDB() (err error) {
	dsn := viper.GetString("database.dsn")
	db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return
	}

	// Migrate the schema
	db.AutoMigrate(&Budget{})
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Account{})
	db.AutoMigrate(&Collection{})
	return
}
