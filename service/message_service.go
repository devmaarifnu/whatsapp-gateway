package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"whatsapp-gateway/model"
	"whatsapp-gateway/repository"
)

type sendJob struct {
	messageID int64
	to        string
	body      string
}

type MessageService struct {
	msgRepo *repository.MessageRepo
	tmplSvc *TemplateService
	client  *whatsmeow.Client
	queue   chan sendJob
	logger  *zap.Logger
}

func NewMessageService(
	msgRepo *repository.MessageRepo,
	tmplSvc *TemplateService,
	client *whatsmeow.Client,
	workerCount int,
	logger *zap.Logger,
) *MessageService {
	svc := &MessageService{
		msgRepo: msgRepo,
		tmplSvc: tmplSvc,
		client:  client,
		queue:   make(chan sendJob, 100),
		logger:  logger,
	}
	for i := 0; i < workerCount; i++ {
		go svc.worker()
	}
	return svc
}

var nonDigit = regexp.MustCompile(`\D`)

func normalizePhone(phone string) (string, error) {
	// strip semua non-digit (spasi, dash, +, kurung, dll)
	digits := nonDigit.ReplaceAllString(phone, "")

	if digits == "" {
		return "", fmt.Errorf("nomor tidak valid: %q", phone)
	}

	switch {
	case strings.HasPrefix(digits, "62"):
		// sudah benar: 628xxx
	case strings.HasPrefix(digits, "0"):
		// 08xxx → 628xxx
		digits = "62" + digits[1:]
	default:
		// 8xxx → 628xxx
		digits = "62" + digits
	}

	return digits, nil
}

func (s *MessageService) SendSingle(to, body string) (int64, error) {
	normalized, err := normalizePhone(to)
	if err != nil {
		return 0, err
	}
	to = normalized

	msg := &model.Message{
		Type:     model.TypeSingle,
		ToNumber: to,
		Body:     body,
	}
	id, err := s.msgRepo.Insert(msg)
	if err != nil {
		return 0, err
	}
	s.queue <- sendJob{messageID: id, to: to, body: body}
	return id, nil
}

func (s *MessageService) SendTemplate(to, templateName string, vars map[string]string) (int64, error) {
	normalized, err := normalizePhone(to)
	if err != nil {
		return 0, err
	}
	to = normalized

	body, tmpl, err := s.tmplSvc.Resolve(templateName, vars)
	if err != nil {
		return 0, err
	}

	msg := &model.Message{
		Type:       model.TypeTemplate,
		TemplateID: &tmpl.ID,
		ToNumber:   to,
		Body:       body,
	}
	id, err := s.msgRepo.Insert(msg)
	if err != nil {
		return 0, err
	}
	s.queue <- sendJob{messageID: id, to: to, body: body}
	return id, nil
}

func (s *MessageService) worker() {
	for job := range s.queue {
		s.process(job)
	}
}

func (s *MessageService) process(job sendJob) {
	jid, err := types.ParseJID(job.to + "@s.whatsapp.net")
	if err != nil {
		msg := fmt.Sprintf("invalid JID: %v", err)
		s.msgRepo.UpdateStatus(job.messageID, model.StatusFailed, &msg)
		return
	}

	_, err = s.client.SendMessage(context.Background(), jid, &waE2E.Message{
		Conversation: proto.String(job.body),
	})
	if err != nil {
		msg := err.Error()
		s.logger.Error("send failed", zap.Int64("id", job.messageID), zap.Error(err))
		s.msgRepo.UpdateStatus(job.messageID, model.StatusFailed, &msg)
		return
	}

	s.msgRepo.UpdateStatus(job.messageID, model.StatusSent, nil)
	s.logger.Info("message sent", zap.Int64("id", job.messageID), zap.String("to", job.to))
}

