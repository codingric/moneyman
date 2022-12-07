package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type CollectionModelFactory struct {
}

type CollectionCreator struct {
	Name    string `json:"name"`
	Notes   string `json:"notes"`
	Amount  int64  `json:"amount"`
	Creator uint   `json:"-"`
}

type CollectionReader struct{}

type CollectionUpdater struct{}

type CollectionDeleter struct{}

type Collection struct {
	gorm.Model
	Name          string    `json:"name"`
	Notes         string    `json:"notes"`
	Enabled       bool      `json:"enabled"`
	Creator       string    `json:"creator"`
	FromAccountID int       `json:"from_account_id"`
	FromAccount   Account   `json:"-"`
	ToAccountID   int       `json:"to_account_id"`
	ToAccount     Account   `json:"-"`
	Start         time.Time `json:"start"`
	End           time.Time `json:"end"`
	Period        string    `json:"period"`
}

type Collections []Collection

func (c *CollectionCreator) SetOwner(u *User) {
	c.Creator = u.ID
}

func (f *CollectionModelFactory) Creator() CreatorRequest {
	return new(CollectionCreator)
}
func (f *CollectionModelFactory) Reader() interface{} {
	return new(CollectionReader)
}
func (f *CollectionModelFactory) Updater() interface{} {
	return new(CollectionUpdater)
}
func (f *CollectionModelFactory) Deleter() interface{} {
	return new(CollectionDeleter)
}

func (f *CollectionModelFactory) New(b interface{}) Model {
	bb := b.(*CollectionCreator)
	var v = &Collection{}
	v.Name = bb.Name
	v.Notes = bb.Notes
	return v
}

func (f *CollectionModelFactory) All() Models {
	var vs *Collections
	db.Find(&vs)
	return vs
}

func (v *Collection) validate() error {
	if v.Notes == "" {
		return fmt.Errorf("notes are required")
	}
	return nil
}

func (v *Collection) Add() error {
	if err := v.validate(); err != nil {
		return err
	}

	db.Create(&v)
	return nil
}

func (vs *Collections) Hydrate() {
	db.Find(&vs)
}

func (vs *Collections) Count() (i int64) {
	db.Find(&vs).Count(&i)
	return
}

func (v *Collection) Save() {

	db.Save(&v)
}

func (v *Collection) Get() {
	db.Find(&v)
}
