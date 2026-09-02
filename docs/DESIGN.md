# WhatsApp Gateway — Design

## Stack

| Layer | Tech |
|---|---|
| Service | Go |
| WA Client | whatsmeow |
| WA Session | SQLite |
| Auth / Template / History | MySQL |
| HTTP | Gin / Echo |
| Config | config.yaml (via viper) |
| Alert | SMTP email + Telegram Bot |

---

## Arsitektur

```
┌─────────────────────────────────────────────────────┐
│                   HTTP API Layer                     │
│  POST /send/message    POST /send/template          │
│  GET  /qr              GET  /status                 │
│  Middleware: Static Token Auth (from MySQL)         │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│                 Message Service                      │
│  - Validate request                                  │
│  - Resolve template (jika template)                  │
│  - Insert ke messages (status=pending)               │
│  - Return message_id immediately (async)             │
│  - Worker goroutine: send + update status            │
└──────┬───────────────────────────┬──────────────────┘
       │                           │
┌──────▼──────┐           ┌───────▼────────┐
│  whatsmeow  │           │     MySQL      │
│  (SQLite)   │           │  - access_token│
│  WA session │           │  - templates   │
└──────┬──────┘           │  - messages    │
       │ disconnect event └────────────────┘
┌──────▼──────────────────────┐
│       Alert Service         │
│  - detect disconnect        │
│  - cooldown check           │
│  - kirim email SMTP         │
│  - kirim Telegram message   │
└─────────────────────────────┘
```

---

## MySQL Schema

### Auth — tabel existing (tidak dibuat ulang)

```sql
CREATE TABLE `access_token` (
    `id`         BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `name`       VARCHAR(255) NOT NULL,
    `token`      VARCHAR(255) NOT NULL,
    `hashtype`   VARCHAR(10) NULL DEFAULT NULL,  -- NULL=plain, 'bcrypt', 'sha256'
    `expires_at` TIMESTAMP NULL DEFAULT NULL,    -- NULL = tidak expired
    `created_at` TIMESTAMP NULL DEFAULT NULL,
    `updated_at` TIMESTAMP NULL DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `access_token_token_unique` (`token`)
);
```

**Validasi token di middleware:**
- `hashtype IS NULL` → plain compare
- `hashtype = 'bcrypt'` → bcrypt.CompareHashAndPassword
- `hashtype = 'sha256'` → sha256(requestToken) lalu compare
- `expires_at IS NOT NULL AND expires_at < NOW()` → 401

### Template + History (dibuat baru)

```sql
CREATE TABLE templates (
  id         INT AUTO_INCREMENT PRIMARY KEY,
  name       VARCHAR(100) UNIQUE NOT NULL,
  body       TEXT NOT NULL,          -- "Halo {{name}}, pesanan {{order_id}} siap"
  is_active  BOOLEAN DEFAULT TRUE,
  created_at DATETIME,
  updated_at DATETIME
);

CREATE TABLE messages (
  id           BIGINT AUTO_INCREMENT PRIMARY KEY,
  type         ENUM('single','template') NOT NULL,
  template_id  INT NULL,
  to_number    VARCHAR(20) NOT NULL,
  body         TEXT NOT NULL,        -- resolved body setelah substitusi
  status       ENUM('pending','sent','failed') DEFAULT 'pending',
  error_msg    TEXT NULL,
  sent_at      DATETIME NULL,
  created_at   DATETIME,
  FOREIGN KEY (template_id) REFERENCES templates(id)
);
```

---

## API Endpoints

### Auth Middleware

```
Header: Authorization: Bearer <token>
→ SELECT * FROM access_token WHERE token = ?
→ cek hashtype → verify
→ cek expires_at
→ 401 jika gagal
```

### `POST /send/message`

```json
Request:
{
  "to": "628123456789",
  "message": "Halo, ini pesan langsung"
}

Response 202 Accepted:
{
  "message_id": 42,
  "status": "pending"
}
```

### `POST /send/template`

```json
Request:
{
  "to": "628123456789",
  "template": "order_ready",
  "variables": {
    "name": "Budi",
    "order_id": "ORD-001"
  }
}

Response 202 Accepted:
{
  "message_id": 43,
  "status": "pending"
}
```

### `GET /messages/:id`
Cek status pesan (polling setelah async submit).

```json
{
  "id": 42,
  "status": "sent",
  "sent_at": "2026-09-02T10:00:00Z"
}
```

### `GET /messages?to=628xxx&status=sent`
Query history.

### `GET /qr`
Return QR code untuk link WA session.

```json
{
  "qr": "data:image/png;base64,..."
}
```
Atau return raw PNG via `Accept: image/png`.

### `GET /status`
Cek apakah WA client connected / disconnected.

```json
{
  "connected": true,
  "phone": "628123456789"
}
```

---

## Send Flow (Async)

### Single Message

```
POST /send/message
→ Auth middleware
→ Validate to + message
→ INSERT messages (status=pending)
→ Return 202 {message_id, status:"pending"}
                    ↓ (goroutine worker)
→ Send via whatsmeow
→ UPDATE messages SET status=sent/failed, sent_at, error_msg
```

### Template Message

```
POST /send/template
→ Auth middleware
→ Load template dari MySQL by name
→ Substitusi variabel: strings.Replace("{{name}}", value)
→ INSERT messages (status=pending, template_id)
→ Return 202 {message_id, status:"pending"}
                    ↓ (goroutine worker)
→ Send via whatsmeow
→ UPDATE messages SET status=sent/failed
```

**Worker pattern:** channel-based queue. Message service push ke channel, worker goroutine consume dan send. Jumlah worker configurable.

---

## QR Scan Flow (via API)

```
GET /qr
→ cek apakah sudah connected → return 200 {connected:true}
→ jika belum: whatsmeow.GetQRChannel()
→ encode QR ke PNG / base64
→ return ke client
→ client scan via HP
→ whatsmeow emit events.Connected
→ session tersimpan di SQLite
```

---

## Alert Service — Disconnect Flow

```
whatsmeow event: events.Disconnected / events.LoggedOut
→ alert_service.Notify(reason)
→ cek cooldown (lastSentAt + cooldown_minutes > now → skip)
→ compose pesan (timestamp, reason, hostname)
→ kirim email via SMTP (jika enabled)
→ kirim Telegram message via Bot API (jika enabled)
→ update lastSentAt
```

**Cooldown penting** — whatsmeow emit disconnect berkali-kali saat reconnect loop.

Event yang di-watch:

| Event | Trigger |
|---|---|
| `events.Disconnected` | koneksi WA putus (network, timeout) |
| `events.LoggedOut` | session expired / logout dari HP |

---

## Struktur Folder

```
whatsapp-gateway/
├── main.go
├── config.yaml
├── config/
│   └── config.go            -- load config.yaml via viper, struct Config
├── db/
│   ├── mysql.go             -- MySQL connection
│   └── sqlite.go            -- whatsmeow SQLite store
├── handler/
│   ├── send.go              -- POST /send/message, POST /send/template
│   ├── history.go           -- GET /messages
│   └── whatsapp.go          -- GET /qr, GET /status
├── middleware/
│   └── auth.go              -- token verify (plain/bcrypt/sha256 + expires_at)
├── model/
│   ├── token.go
│   ├── template.go
│   └── message.go
├── repository/
│   ├── token_repo.go
│   ├── template_repo.go
│   └── message_repo.go
├── service/
│   ├── message_service.go   -- enqueue, worker goroutine
│   ├── template_service.go  -- load + substitusi variabel
│   └── alert_service.go     -- email + telegram notify
└── whatsapp/
    └── client.go            -- init whatsmeow, event handler, QR channel
```

---

## config.yaml

```yaml
server:
  port: 8080

database:
  mysql:
    host: "127.0.0.1"
    port: 3306
    user: "root"
    password: "secret"
    name: "wagateway"
  sqlite:
    path: "./whatsapp.db"

worker:
  count: 3                    # jumlah goroutine worker pengirim pesan

alert:
  enabled: true
  cooldown_minutes: 30

  email:
    enabled: true
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "bot@example.com"
    password: "secret"
    from: "bot@example.com"
    to:
      - "admin@example.com"
      - "ops@example.com"
    subject: "[ALERT] WhatsApp Gateway Disconnected"

  telegram:
    enabled: true
    bot_token: "123456:ABC-DEF..."
    chat_ids:
      - "-100123456789"       # group/channel ID
      - "987654321"           # personal ID
    message_template: "⚠️ WA Gateway disconnected\nHost: {{hostname}}\nTime: {{time}}\nReason: {{reason}}"
```

---

## Keputusan Desain

| Aspek | Keputusan |
|---|---|
| WA session | SQLite via whatsmeow store |
| Token | Tabel `access_token` existing — validasi hashtype + expires_at |
| Template var | Simple `{{key}}` substitution |
| Send mode | **Async** — return 202 + message_id, worker update status |
| Worker | Channel-based goroutine pool, count configurable |
| QR scan | Via `GET /qr` endpoint |
| Multi-device | 1 WA account |
| Config source | `config.yaml` via viper |
| Alert channel | Email (SMTP) + Telegram Bot — masing-masing bisa enable/disable |
| Alert cooldown | Configurable, default 30 menit |
| MySQL config | Host, port, user, password, name — bukan DSN string |
