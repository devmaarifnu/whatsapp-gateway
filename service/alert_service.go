package service

import (
	"fmt"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"whatsapp-gateway/config"
)

type AlertService struct {
	cfg    config.AlertConfig
	logger *zap.Logger
	mu     sync.Mutex
	lastAt time.Time
}

func NewAlertService(cfg config.AlertConfig, logger *zap.Logger) *AlertService {
	return &AlertService{cfg: cfg, logger: logger}
}

func (s *AlertService) Notify(reason string) {
	if !s.cfg.Enabled {
		return
	}

	s.mu.Lock()
	cooldown := time.Duration(s.cfg.CooldownMinutes) * time.Minute
	if !s.lastAt.IsZero() && time.Since(s.lastAt) < cooldown {
		s.mu.Unlock()
		return
	}
	s.lastAt = time.Now()
	s.mu.Unlock()

	hostname, _ := os.Hostname()
	now := time.Now().Format("2006-01-02 15:04:05")

	if s.cfg.Email.Enabled {
		go s.sendEmail(hostname, now, reason)
	}
	if s.cfg.Telegram.Enabled {
		go s.sendTelegram(hostname, now, reason)
	}
}

func (s *AlertService) sendEmail(hostname, now, reason string) {
	cfg := s.cfg.Email
	body := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\nHost: %s\nTime: %s\nReason: %s",
		cfg.From,
		strings.Join(cfg.To, ", "),
		cfg.Subject,
		hostname, now, reason,
	)
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	if err := smtp.SendMail(addr, auth, cfg.From, cfg.To, []byte(body)); err != nil {
		s.logger.Error("email alert failed", zap.Error(err))
	}
}

func (s *AlertService) sendTelegram(hostname, now, reason string) {
	cfg := s.cfg.Telegram
	text := cfg.MessageTemplate
	text = strings.ReplaceAll(text, "{{hostname}}", hostname)
	text = strings.ReplaceAll(text, "{{time}}", now)
	text = strings.ReplaceAll(text, "{{reason}}", reason)

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	for _, chatID := range cfg.ChatIDs {
		params := url.Values{"chat_id": {chatID}, "text": {text}}
		resp, err := http.PostForm(apiURL, params)
		if err != nil {
			s.logger.Error("telegram alert failed", zap.String("chat_id", chatID), zap.Error(err))
			continue
		}
		resp.Body.Close()
	}
}

