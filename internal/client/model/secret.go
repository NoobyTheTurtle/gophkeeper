package model

import "time"

type SecretList struct {
	Id   int
	Name string
}

type LoginPassSecret struct {
	Id         int
	Name       string
	RecordType int
	Login      string
	Password   string
	UpdatedAt  time.Time
}

type TextSecret struct {
	Id         int `json:"id"`
	Name       string
	RecordType int
	Text       string
	UpdatedAt  time.Time
}

type FileSecret struct {
	Id         int
	Name       string
	RecordType int
	Path       string
	Binary     []byte
	UpdatedAt  time.Time
}

type CardSecret struct {
	Id         int
	Name       string
	RecordType int
	CardNumber string
	CVV        string
	Due        string
	UpdatedAt  time.Time
}
