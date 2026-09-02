package model

import "time"

type AccessToken struct {
	ID        uint64
	Name      string
	Token     string
	HashType  *string
	ExpiresAt *time.Time
}
