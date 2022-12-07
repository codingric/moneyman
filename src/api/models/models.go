package models

type Model interface {
	Add() error
	Save()
	Get()
}

type Models interface {
	Hydrate()
	Count() int64
}

type Factory interface {
	New(interface{}) Model
	All() Models

	Creator() CreatorRequest
	Reader() interface{}
	Updater() interface{}
	Deleter() interface{}
}

type CreatorRequest interface {
	SetOwner(*User)
}

// type Creator interface{}
// type Reader interface{}
// type Updater interface{}
// type Deleter interface{}
