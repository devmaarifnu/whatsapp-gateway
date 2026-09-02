package repository

import (
	"database/sql"

	"whatsapp-gateway/model"
)

type TokenRepo struct {
	db *sql.DB
}

func NewTokenRepo(db *sql.DB) *TokenRepo {
	return &TokenRepo{db: db}
}

func (r *TokenRepo) FindByToken(token string) (*model.AccessToken, error) {
	row := r.db.QueryRow(
		`SELECT id, name, token, hashtype, expires_at FROM access_token WHERE token = ?`,
		token,
	)
	return scanToken(row)
}

func (r *TokenRepo) FindByHashType(hashType string) ([]*model.AccessToken, error) {
	rows, err := r.db.Query(
		`SELECT id, name, token, hashtype, expires_at FROM access_token WHERE hashtype = ?`,
		hashType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*model.AccessToken
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanToken(s scanner) (*model.AccessToken, error) {
	var t model.AccessToken
	var hashType sql.NullString
	var expiresAt sql.NullTime

	err := s.Scan(&t.ID, &t.Name, &t.Token, &hashType, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if hashType.Valid {
		t.HashType = &hashType.String
	}
	if expiresAt.Valid {
		t.ExpiresAt = &expiresAt.Time
	}
	return &t, nil
}

