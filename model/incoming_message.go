package model

import "time"

// IncomingMessage merepresentasikan pesan WhatsApp masuk (inbound).
// Disimpan terpisah dari tabel `messages` (outbound history).
type IncomingMessage struct {
	ID           int64     `json:"id"`
	WAMessageID  string    `json:"wa_message_id"` // ID unik dari WhatsApp
	Sender       string    `json:"sender"`        // JID pengirim (nomor@pengirim)
	Chat         string    `json:"chat"`          // JID chat tempat pesan masuk
	Body         string    `json:"body"`
	IsGroup      bool      `json:"is_group"`
	ReceivedAt   time.Time `json:"received_at"`
}
