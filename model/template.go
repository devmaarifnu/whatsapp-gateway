package model

import "time"

type Template struct {
	ID        int
	Name      string
	Body      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
