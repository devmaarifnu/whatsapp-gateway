package model

import "time"

type MessageStatus string

const (
	StatusPending MessageStatus = "pending"
	StatusSent    MessageStatus = "sent"
	StatusFailed  MessageStatus = "failed"
)

type MessageType string

const (
	TypeSingle   MessageType = "single"
	TypeTemplate MessageType = "template"
)

type Message struct {
	ID         int64         `json:"id"`
	Type       MessageType   `json:"type"`
	TemplateID *int          `json:"template_id,omitempty"`
	ToNumber   string        `json:"to_number"`
	Body       string        `json:"body"`
	Status     MessageStatus `json:"status"`
	ErrorMsg   *string       `json:"error_msg,omitempty"`
	SentAt     *time.Time    `json:"sent_at,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
}
