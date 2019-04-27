package main

import (
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
	"github.com/pkg/errors"
)

// conn коннект к базе данных
var pgxConn *pgx.ConnPool

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

// MessageModel Message model for users table
type MessageModel struct {
	ID      pgtype.Int8
	Message pgtype.Text
	Author  pgtype.Text
	ConvID  pgtype.Int8
}

func (ms *AccessObject) Create(m *MessageModel) error {
	tx, err := pgxConn.Begin()
	if err != nil {
		return errors.Wrap(err, "can not open message create transaction")
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO messages (message, author, conv_id) VALUES($1, $2, $3);`, &m.Message, &m.Author, &m.ConvID)
	if err != nil {
		return errors.Wrap(err, "message create error")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "message create transaction commit error")
	}

	return nil

}

func (ms *AccessObject) GetMessagesByConvID(id int64, limit int, offset int) ([]*MessageModel, error) {
	tx, err := pgxConn.Begin()
	if err != nil {
		return nil, errors.Wrap(err, "can not open messages get transaction")
	}
	defer tx.Rollback()

	rows, err := tx.Query(`SELECT m.id, m.message, m.author, m.conv_id FROM messages m WHERE conv_id = $1 LIMIT $2 OFFSET $3 ORDER BY m.id DESC;`, id, limit, offset)
	msgs := make([]*MessageModel, 0, 0)

	for rows.Next() {
		m := &MessageModel{}
		err := rows.Scan(&m.ID, &m.Message, &m.Author, &m.ConvID)
		if err != nil {
			return nil, errors.Wrap(err, "faild to get messages")
		}

		msgs = append(msgs, m)
	}

	return msgs, nil
}
