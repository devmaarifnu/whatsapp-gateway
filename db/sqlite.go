package db

import (
	"context"

	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"
	"whatsapp-gateway/config"
)

func NewSQLiteStore(ctx context.Context, cfg config.SQLiteConfig) (*sqlstore.Container, error) {
	dsn := "file:" + cfg.Path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"
	return sqlstore.New(ctx, "sqlite", dsn, waLog.Noop)
}

