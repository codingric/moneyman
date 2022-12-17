package main

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB
var Debug bool

/*
type Interface interface {
  LogMode(LogLevel) Interface
  Info(context.Context, string, ...interface{})
  Warn(context.Context, string, ...interface{})
  Error(context.Context, string, ...interface{})
  Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error)
}
*/

type zerologger struct {
	Logger *zerolog.Logger
}

func (z zerologger) LogMode(logger.LogLevel) logger.Interface            { return nil }
func (z zerologger) Info(c context.Context, m string, x ...interface{})  { z.Logger.Info().Msg(m) }
func (z zerologger) Warn(c context.Context, m string, x ...interface{})  { z.Logger.Warn().Msg(m) }
func (z zerologger) Error(c context.Context, m string, x ...interface{}) { z.Logger.Error().Msg(m) }
func (z zerologger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	s, r := fc()
	z.Logger.Trace().Int64("rows", r).Dur("duration_ms", time.Since(begin)).Msg(s)
}

func ConnectDatabase(path string, debug bool) {

	//var database *gorm.DB
	var err error

	Debug = debug

	DB, err = gorm.Open(
		sqlite.Open(path),
		&gorm.Config{
			Logger: zerologger{
				Logger: &log.Logger,
			},
		},
	)
	if err != nil {
		log.Panic().Msg("Failed to connect to database")
	}

	DB.AutoMigrate(&Account{})
	DB.AutoMigrate(&Transaction{})

	if debug {
		DB = DB.Debug()
	}
	log.Info().Msgf("Database connected: %s", path)
}
