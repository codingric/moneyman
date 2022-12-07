package models

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	viper.Set("database.dsn", ":memory:")
	InitDB()
}

func Test_Budget_validate(t *testing.T) {
	b := &Budget{Name: "Missing Notes"}
	err := b.validate()
	if assert.NotNil(t, err) {
		assert.Error(t, err, "Notes are required")
	}

	b.Notes = "Happy path"
	err = b.validate()
	assert.Nil(t, err)
}
