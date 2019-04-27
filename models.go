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
	Messages = &MessageAccessObject{}
}

// User model for users table
type MessageModel struct {
	ID       pgtype.Int8
	message  pgtype.Text
	authorID pgtype.Int8
	convID   pgtype.Int8
}
