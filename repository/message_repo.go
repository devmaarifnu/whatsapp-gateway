package repository

import (
	"database/sql"
	"time"

	"whatsapp-gateway/model"
)

type MessageRepo struct {
	db *sql.DB
}

func NewMessageRepo(db *sql.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

func (r *MessageRepo) Insert(msg *model.Message) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO messages (type, template_id, to_number, body, status, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		msg.Type, msg.TemplateID, msg.ToNumber, msg.Body, model.StatusPending, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *MessageRepo) UpdateStatus(id int64, status model.MessageStatus, errMsg *string) error {
	var sentAt *time.Time
	if status == model.StatusSent {
		now := time.Now()
		sentAt = &now
	}
	_, err := r.db.Exec(
		`UPDATE messages SET status = ?, error_msg = ?, sent_at = ? WHERE id = ?`,
		status, errMsg, sentAt, id,
	)
	return err
}

func (r *MessageRepo) FindByID(id int64) (*model.Message, error) {
	row := r.db.QueryRow(
		`SELECT id, type, template_id, to_number, body, status, error_msg, sent_at, created_at FROM messages WHERE id = ?`,
		id,
	)
	return scanMessage(row)
}

func (r *MessageRepo) List(toNumber string, status model.MessageStatus, limit int) ([]*model.Message, error) {
	query := `SELECT id, type, template_id, to_number, body, status, error_msg, sent_at, created_at FROM messages WHERE 1=1`
	args := []any{}

	if toNumber != "" {
		query += " AND to_number = ?"
		args = append(args, toNumber)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*model.Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func scanMessage(s scanner) (*model.Message, error) {
	var m model.Message
	var templateID sql.NullInt32
	var errMsg sql.NullString
	var sentAt sql.NullTime

	err := s.Scan(&m.ID, &m.Type, &templateID, &m.ToNumber, &m.Body, &m.Status, &errMsg, &sentAt, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if templateID.Valid {
		v := int(templateID.Int32)
		m.TemplateID = &v
	}
	if errMsg.Valid {
		m.ErrorMsg = &errMsg.String
	}
	if sentAt.Valid {
		m.SentAt = &sentAt.Time
	}
	return &m, nil
}

