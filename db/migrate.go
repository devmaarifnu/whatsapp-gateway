package db

import "database/sql"

func MigrateMySQL(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS templates (
			id         INT AUTO_INCREMENT PRIMARY KEY,
			name       VARCHAR(100) UNIQUE NOT NULL,
			body       TEXT NOT NULL,
			is_active  BOOLEAN DEFAULT TRUE,
			created_at DATETIME,
			updated_at DATETIME
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id           BIGINT AUTO_INCREMENT PRIMARY KEY,
			type         ENUM('single','template') NOT NULL,
			template_id  INT NULL,
			to_number    VARCHAR(20) NOT NULL,
			body         TEXT NOT NULL,
			status       ENUM('pending','sent','failed') DEFAULT 'pending',
			error_msg    TEXT NULL,
			sent_at      DATETIME NULL,
			created_at   DATETIME,
			FOREIGN KEY (template_id) REFERENCES templates(id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	return err
}
