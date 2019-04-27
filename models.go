package main

import (
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
)

// conn коннект к базе данных
var pgxConn *pgx.ConnPool

type Queryer interface {
	QueryRow(string, ...interface{}) *pgx.Row
}

// MessageAccessObject DAO for User model
type MessageAccessObject interface {
	GetMessagesByConvID(id int64, limit int, offset int) ([]*MessageModel, error)

	Create(u *MessageModel) error
}

// AccessObject implementation of UserAccessObject
type AccessObject struct{}

var Messages MessageAccessObject

func init() {
	Messages = &AccessObject{}
}

// User model for users table
type MessageModel struct {
	ID       pgtype.Int8
	Message  pgtype.Text
	AuthorID pgtype.Int8
	ConvID   pgtype.Int8
}

func (ms *AccessObject) GetMessagesByConvID(id int64, limit int, offset int) ([]*MessageModel, error) {
	return nil, nil
}

func (ms *AccessObject) Create(u *MessageModel) error {
	return nil
}
