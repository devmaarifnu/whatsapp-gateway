package repository

import (
	"database/sql"

	"whatsapp-gateway/model"
)

type TemplateRepo struct {
	db *sql.DB
}

func NewTemplateRepo(db *sql.DB) *TemplateRepo {
	return &TemplateRepo{db: db}
}

func (r *TemplateRepo) FindByName(name string) (*model.Template, error) {
	row := r.db.QueryRow(
		`SELECT id, name, body, is_active FROM templates WHERE name = ? AND is_active = 1`,
		name,
	)
	var t model.Template
	err := row.Scan(&t.ID, &t.Name, &t.Body, &t.IsActive)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

