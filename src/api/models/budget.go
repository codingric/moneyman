package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type BudgetModelFactory struct {
}

type BudgetCreator struct {
	Name          string    `json:"name"`
	Notes         string    `json:"notes"`
	Amount        int64     `json:"amount"`
	FromAccountID int       `json:"from_account_id"`
	ToAccountID   int       `json:"to_account_id"`
	Start         time.Time `json:"start"`
	End           time.Time `json:"end"`
	Period        string    `json:"period"`
	Creator       uint      `json:"-"`
}

type BudgetReader struct{}

type BudgetUpdater struct{}

type BudgetDeleter struct{}

type Budget struct {
	gorm.Model
	Name          string    `json:"name"`
	Notes         string    `json:"notes"`
	Amount        int64     `json:"amount"`
	Enabled       bool      `json:"enabled"`
	CreatorID     uint      `json:"creator_id"`
	Creator       User      `json:"-"`
	FromAccountID int       `json:"from_account_id"`
	FromAccount   Account   `json:"-"`
	ToAccountID   int       `json:"to_account_id"`
	ToAccount     Account   `json:"-"`
	Start         time.Time `json:"start"`
	End           time.Time `json:"end"`
	Period        string    `json:"period"`
}

type Budgets []Budget

func (c *BudgetCreator) SetOwner(u *User) {
	c.Creator = u.ID
}

func (f *BudgetModelFactory) Creator() CreatorRequest {
	return new(BudgetCreator)
}
func (f *BudgetModelFactory) Reader() interface{} {
	return new(BudgetReader)
}
func (f *BudgetModelFactory) Updater() interface{} {
	return new(BudgetUpdater)
}
func (f *BudgetModelFactory) Deleter() interface{} {
	return new(BudgetDeleter)
}

func (f *BudgetModelFactory) New(b interface{}) Model {
	bb := b.(*BudgetCreator)
	var v = &Budget{}
	v.Name = bb.Name
	v.Notes = bb.Notes
	v.Amount = bb.Amount
	v.Enabled = false
	v.FromAccountID = bb.FromAccountID
	v.ToAccountID = bb.ToAccountID
	v.Start = bb.Start
	v.End = bb.End
	v.Period = bb.Period
	return v
}

func (f *BudgetModelFactory) All() Models {
	var vs *Budgets
	db.Find(&vs)
	return vs
}

func (v *Budget) validate() error {
	if v.Notes == "" {
		return fmt.Errorf("notes are required")
	}
	return nil
}

func (v *Budget) Add() error {
	if err := v.validate(); err != nil {
		return err
	}

	db.Create(&v)
	return nil
}

func (vs *Budgets) Hydrate() {
	db.Find(&vs)
}

func (vs *Budgets) Count() (i int64) {
	db.Find(&vs).Count(&i)
	return
}

func (v *Budget) Save() {

	db.Save(&v)
}

func (v *Budget) Get() {
	db.Find(&v)
}
