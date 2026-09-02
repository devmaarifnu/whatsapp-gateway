package db

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"whatsapp-gateway/config"
)

func NewMySQL(cfg config.MySQLConfig) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

