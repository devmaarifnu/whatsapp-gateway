package handler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type MessageHandler struct {
	logger *zap.Logger
	db     *sql.DB
}

func NewMessageHandler(logger *zap.Logger) *MessageHandler {
	return &MessageHandler{logger: logger}
}

func (h *MessageHandler) SetDB(db *sql.DB) {
	h.db = db
}

func (h *MessageHandler) Handle(client *whatsmeow.Client, evt *events.Message) {
	// Ambil teks pesan
	body := extractText(evt)
	if body == "" {
		return
	}

	sender := evt.Info.Sender.String()
	chat := evt.Info.Chat.String()

	h.logger.Info("pesan masuk",
		zap.String("dari", sender),
		zap.String("chat", chat),
		zap.String("isi", body),
	)

	// Simpan ke DB
	if h.db != nil {
		h.saveMessage(evt.Info.ID, sender, chat, body)
	}

	// Proses command
	reply := h.processCommand(body)
	if reply == "" {
		return
	}

	// Kirim balasan
	_, err := client.SendMessage(context.Background(), evt.Info.Chat, &waE2E.Message{
		Conversation: proto.String(reply),
	})
	if err != nil {
		h.logger.Error("gagal kirim pesan", zap.Error(err))
	}
}

func (h *MessageHandler) processCommand(text string) string {
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
			"!stat  — statistik pesan"

	case strings.HasPrefix(lower, "!echo "):
		return strings.TrimPrefix(text, "!echo ")

	case lower == "!stat":
		return h.getStats()

	default:
		return ""
	}
}

func (h *MessageHandler) saveMessage(id, sender, chat, body string) {
	_, err := h.db.Exec(
		`INSERT OR IGNORE INTO messages (id, sender, chat, body, received_at) VALUES (?, ?, ?, ?, ?)`,
		id, sender, chat, body, time.Now().Unix(),
	)
	if err != nil {
		h.logger.Error("gagal simpan pesan", zap.Error(err))
	}
}

func (h *MessageHandler) getStats() string {
	if h.db == nil {
		return "DB tidak tersedia."
	}

	var total int
	h.db.QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&total)

	var uniqueSenders int
	h.db.QueryRow(`SELECT COUNT(DISTINCT sender) FROM messages`).Scan(&uniqueSenders)

	var last string
	h.db.QueryRow(`SELECT body FROM messages ORDER BY received_at DESC LIMIT 1`).Scan(&last)

	return fmt.Sprintf(
		"Statistik pesan:\nTotal: %d\nPengirim unik: %d\nPesan terakhir: %s",
		total, uniqueSenders, last,
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
