package models

import (
	"fmt"

	"gorm.io/gorm"
)

type AccountModelFactory struct {
}

type AccountCreator struct {
	Name    string `json:"name"`
	Notes   string `json:"notes"`
	Amount  int64  `json:"amount"`
	Creator uint   `json:"-"`
}

type AccountReader struct{}

type AccountUpdater struct{}

type AccountDeleter struct{}

type Account struct {
	gorm.Model
	Name      string `json:"name"`
	Notes     string `json:"notes"`
	Amount    int64  `json:"amount"`
	Enabled   bool   `json:"enabled"`
	CreatorID string `json:"creator_id"`
	Creator   User
}

type Accounts []Account

func (c *AccountCreator) SetOwner(u *User) {
	c.Creator = u.ID
}

func (f *AccountModelFactory) Creator() CreatorRequest {
	return new(AccountCreator)
}
func (f *AccountModelFactory) Reader() interface{} {
	return new(AccountReader)
}
func (f *AccountModelFactory) Updater() interface{} {
	return new(AccountUpdater)
}
func (f *AccountModelFactory) Deleter() interface{} {
	return new(AccountDeleter)
}

func (f *AccountModelFactory) New(b interface{}) Model {
	bb := b.(*AccountCreator)
	var v = &Account{}
	v.Name = bb.Name
	v.Notes = bb.Notes
	v.Amount = bb.Amount
	return v
}

func (f *AccountModelFactory) All() Models {
	var vs *Accounts
	db.Find(&vs)
	return vs
}

func (v *Account) validate() error {
	if v.Notes == "" {
		return fmt.Errorf("notes are required")
	}
	return nil
}

func (v *Account) Add() error {
	if err := v.validate(); err != nil {
		return err
	}

	db.Create(&v)
	return nil
}

func (vs *Accounts) Hydrate() {
	db.Find(&vs)
}

func (vs *Accounts) Count() (i int64) {
	db.Find(&vs).Count(&i)
	return
}

func (v *Account) Save() {

	db.Save(&v)
}
func (v *Account) Get() {
	db.Find(&v)
}
