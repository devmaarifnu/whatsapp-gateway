package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Worker   WorkerConfig   `mapstructure:"worker"`
	Alert    AlertConfig    `mapstructure:"alert"`
	Log      LogConfig      `mapstructure:"log"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`       // debug, info, warn, error
	File       string `mapstructure:"file"`        // path to log file, empty = stdout only
	MaxSizeMB  int    `mapstructure:"max_size_mb"` // MB before rotate
	MaxBackups int    `mapstructure:"max_backups"` // jumlah file lama disimpan
	MaxAgeDays int    `mapstructure:"max_age_days"`// hari sebelum dihapus
	Compress   bool   `mapstructure:"compress"`    // gzip rotated files
}

type ServerConfig struct {
	Port     int    `mapstructure:"port"`
	BasePath string `mapstructure:"base_path"`
}

type DatabaseConfig struct {
	MySQL  MySQLConfig  `mapstructure:"mysql"`
	SQLite SQLiteConfig `mapstructure:"sqlite"`
}

type MySQLConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
}

func (m MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		m.User, m.Password, m.Host, m.Port, m.Name)
}

type SQLiteConfig struct {
	Path string `mapstructure:"path"`
}

type WorkerConfig struct {
	Count int `mapstructure:"count"`
}

type AlertConfig struct {
	Enabled         bool           `mapstructure:"enabled"`
	CooldownMinutes int            `mapstructure:"cooldown_minutes"`
	Email           EmailConfig    `mapstructure:"email"`
	Telegram        TelegramConfig `mapstructure:"telegram"`
}

type EmailConfig struct {
	Enabled  bool     `mapstructure:"enabled"`
	SMTPHost string   `mapstructure:"smtp_host"`
	SMTPPort int      `mapstructure:"smtp_port"`
	Username string   `mapstructure:"username"`
	Password string   `mapstructure:"password"`
	From     string   `mapstructure:"from"`
	To       []string `mapstructure:"to"`
	Subject  string   `mapstructure:"subject"`
}

type TelegramConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	BotToken        string   `mapstructure:"bot_token"`
	ChatIDs         []string `mapstructure:"chat_ids"`
	MessageTemplate string   `mapstructure:"message_template"`
}

func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")

	viper.SetDefault("server.port", 8080)
	viper.SetDefault("worker.count", 3)
	viper.SetDefault("alert.cooldown_minutes", 30)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.max_size_mb", 100)
	viper.SetDefault("log.max_backups", 7)
	viper.SetDefault("log.max_age_days", 30)
	viper.SetDefault("log.compress", true)

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
