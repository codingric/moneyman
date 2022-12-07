package models

import (
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primarykey"`
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `json:"name"`
	Email     string         `json:"email"`
	Hash      string         `json:"-"`
}

type UserModelFactory struct {
}

type UserCreator struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Creator  uint   `json:"-"`
}

type UserReader struct{}

type UserUpdater struct{}

type UserDeleter struct{}

type Users []User

func (c *UserCreator) SetOwner(u *User) {
	c.Creator = u.ID
}

func (f *UserModelFactory) Creator() CreatorRequest {
	return new(UserCreator)
}
func (f *UserModelFactory) Reader() interface{} {
	return new(UserReader)
}
func (f *UserModelFactory) Updater() interface{} {
	return new(UserUpdater)
}
func (f *UserModelFactory) Deleter() interface{} {
	return new(UserDeleter)
}

func (f *UserModelFactory) New(b interface{}) Model {
	bb := b.(*UserCreator)
	var v = &User{}
	v.Name = bb.Name
	v.Email = bb.Email
	bytes, _ := bcrypt.GenerateFromPassword([]byte(bb.Password), 14)
	v.Hash = string(bytes)
	return v
}

func (f *UserModelFactory) All() Models {
	var vs *Users
	db.Find(&vs)
	return vs
}

func (v *User) validate() error {
	if v.Name == "" {
		return fmt.Errorf("name is required")
	}
	if v.Email == "" {
		return fmt.Errorf("email is required")
	}
	return nil
}

func (v *User) Add() error {
	if err := v.validate(); err != nil {
		return err
	}

	db.Create(&v)
	return nil
}

func (vs *Users) Hydrate() {
	db.Find(&vs)
}

func (vs *Users) Count() (i int64) {
	db.Find(&vs).Count(&i)
	return
}

func (v *User) Save() {

	db.Save(&v)
}

func (v *User) Get() {
	db.Find(&v)
}
