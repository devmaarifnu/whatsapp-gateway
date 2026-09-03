package repository

import (
	"database/sql"
	"time"

	"whatsapp-gateway/model"
)

// IncomingRepo menyimpan pesan WA masuk ke tabel incoming_messages.
type IncomingRepo struct {
	db *sql.DB
}

func NewIncomingRepo(db *sql.DB) *IncomingRepo {
	return &IncomingRepo{db: db}
}

// Insert menyimpan satu pesan masuk. wa_message_id UNIQUE dipakai sebagai
// kunci idempotensi — jika pesan yang sama sudah pernah tersimpan (mis. dari
// multi-device / event berulang), insert dibungkam via ON DUPLICATE KEY.
func (r *IncomingRepo) Insert(msg *model.IncomingMessage) (int64, error) {
	now := time.Now()
	msg.ReceivedAt = now

	res, err := r.db.Exec(
		`INSERT INTO incoming_messages (wa_message_id, sender, chat, body, is_group, received_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE id = id`, // wa_message_id sudah ada → abaikan duplikat
		msg.WAMessageID, msg.Sender, msg.Chat, msg.Body, msg.IsGroup, now,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	msg.ID = id
	return id, nil
}

// CountBySender menghitung jumlah pesan masuk dari satu pengirim (JID).
func (r *IncomingRepo) CountBySender(senderJID string) (int, error) {
	var total int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM incoming_messages WHERE sender = ?`, senderJID,
	).Scan(&total)
	return total, err
}

// LastBodyBySender mengambil isi pesan masuk terbaru dari satu pengirim.
func (r *IncomingRepo) LastBodyBySender(senderJID string) (string, error) {
	var body string
	err := r.db.QueryRow(
		`SELECT body FROM incoming_messages WHERE sender = ? ORDER BY received_at DESC LIMIT 1`,
		senderJID,
	).Scan(&body)
	return body, err
}
