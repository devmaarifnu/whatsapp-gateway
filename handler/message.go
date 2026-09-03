package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"whatsapp-gateway/model"
	"whatsapp-gateway/repository"
)

type MessageHandler struct {
	logger *zap.Logger
	repo   *repository.IncomingRepo
}

func NewMessageHandler(logger *zap.Logger, repo *repository.IncomingRepo) *MessageHandler {
	return &MessageHandler{logger: logger, repo: repo}
}

// Handle dipanggil untuk setiap pesan masuk (events.Message).
func (h *MessageHandler) Handle(client *whatsmeow.Client, evt *events.Message) {
	// Abaikan pesan yang dikirim nomor ini sendiri (echo) dan pesan tanpa teks.
	if evt.Info.IsFromMe {
		return
	}

	body := extractText(evt)
	if body == "" {
		return
	}

	sender := evt.Info.Sender.String()
	chat := evt.Info.Chat.String()
	isGroup := evt.Info.IsGroup

	// Simpan ke MySQL (tabel incoming_messages)
	_, err := h.repo.Insert(&model.IncomingMessage{
		WAMessageID: evt.Info.ID,
		Sender:      sender,
		Chat:        chat,
		Body:        body,
		IsGroup:     isGroup,
	})
	if err != nil {
		h.logger.Error("gagal simpan pesan masuk", zap.String("id", evt.Info.ID), zap.Error(err))
	} else {
		h.logger.Info("pesan masuk tersimpan",
			zap.String("id", evt.Info.ID),
			zap.String("dari", sender),
			zap.String("chat", chat),
		)
	}

	// Balas perintah hanya pada chat 1-on-1; pesan grup cukup disimpan.
	if isGroup {
		return
	}

	reply := h.processCommand(body, sender)
	if reply == "" {
		return
	}

	_, err = client.SendMessage(context.Background(), evt.Info.Chat, &waE2E.Message{
		Conversation: proto.String(reply),
	})
	if err != nil {
		h.logger.Error("gagal kirim pesan balasan", zap.Error(err))
	}
}

func (h *MessageHandler) processCommand(text, sender string) string {
	lower := strings.ToLower(strings.TrimSpace(text))

	switch {
	case lower == "!ping":
		return "Pong! Bot aktif."

	case lower == "!waktu":
		return fmt.Sprintf("Waktu sekarang: %s", time.Now().Format("02 Jan 2006, 15:04:05"))

	case lower == "!help":
		return "Perintah tersedia:\n" +
			"!ping  — cek bot aktif\n" +
			"!waktu — lihat waktu server\n" +
			"!echo <teks> — balik teks\n" +
			"!stat  — statistik pesan kamu"

	case strings.HasPrefix(lower, "!echo "):
		return strings.TrimPrefix(text, "!echo ")

	case lower == "!stat":
		return h.getStats(sender)

	default:
		return ""
	}
}

func (h *MessageHandler) getStats(sender string) string {
	total, err := h.repo.CountBySender(sender)
	if err != nil {
		return "Gagal membaca statistik."
	}
	lastBody, err := h.repo.LastBodyBySender(sender)
	if err != nil {
		lastBody = "—"
	}

	return fmt.Sprintf(
		"Statistik pesan kamu:\nTotal: %d\nPesan terakhir: %s",
		total, lastBody,
	)
}

func extractText(evt *events.Message) string {
	msg := evt.Message
	if msg == nil {
		return ""
	}
	if msg.Conversation != nil {
		return msg.GetConversation()
	}
	if msg.ExtendedTextMessage != nil {
		return msg.ExtendedTextMessage.GetText()
	}
	return ""
}
