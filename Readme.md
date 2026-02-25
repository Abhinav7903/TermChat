# TermChat 🖥️💬
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/Abhinav7903/TermChat)
[![Release](https://img.shields.io/github/v/release/Abhinav7903/TermChat)](https://github.com/Abhinav7903/TermChat/releases)
[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**TermChat** is a terminal-based chat application with a fully interactive **Bubble Tea TUI client** and a robust **TCP/Telnet server**. It supports real-time messaging, persistent chat history, ephemeral chats, live notifications, and encrypted message storage — all backed by **PostgreSQL** and **Redis**.

![UI-DEMO](https://github.com/Abhinav7903/TermChat/blob/main/UIDEMo.gif)

![Fast Demo](https://github.com/Abhinav7903/TermChat/blob/main/demo.gif)

---

## ✨ Features

- 🎨 **Rich Terminal UI**
  - Full Bubble Tea TUI client with Lip Gloss styling
  - Dark theme with colored panels, badges, and live indicators
  - Keyboard-driven with command history navigation
- 🔐 **Secure Authentication**
  - Register & login with email + password
  - Password hashing with **bcrypt**
- 🔒 **Encrypted Messages**
  - AES-256-GCM message storage
- ⚡ **Real-Time Communication**
  - Persistent chat with history via **PostgreSQL** + **Redis pub/sub**
  - Ephemeral temp chats with no DB storage
  - One-off direct message sending
- 🔔 **Live Notifications**
  - Instant alerts when someone starts a `/chat` or `/tempchat` with you
  - Terminal bell sound on notification
  - Per-user Redis notification channels — zero polling
- 👥 **Chat Types**
  - `/chat` — persistent one-to-one with scrollable history
  - `/tempchat` — ephemeral real-time, nothing saved
  - `/send` — fire-and-forget direct message
- 🔎 **Search & Discovery**
  - Search users by name prefix
  - Room/partner list on login

---

## 📂 Project Structure

```text
cmd/                 # Entry point (main.go)
client/              # Bubble Tea TUI client
  client.go          #   Program entrypoint
  model.go           #   State machine & server message parsing
  views.go           #   Lip Gloss layouts & rendering
  tcp.go             #   TCP connect / read / write helpers
db/
  migrations/        # SQL migrations
  postgres/          # PostgreSQL connection & queries
  redis/             # Redis client
factory/             # Data models (User, Message)
pkg/
  message/           # Message repository interface
  users/             # User repository interface & utils
server/              # HTTP & TCP server logic
  telnet.go          #   TCP command handler & Redis pub/sub
utils/               # Encryption utilities
```

---

## 🚀 Getting Started

### ✅ Prerequisites

- [Go 1.23+](https://go.dev/dl/)
- [PostgreSQL](https://www.postgresql.org/)
- [Redis](https://redis.io/)

### ⚙️ Setup

1. **Clone the repo**

   ```sh
   git clone https://github.com/Abhinav7903/TermChat.git
   cd TermChat
   ```

2. **Configure environment**

   Place config files in `$HOME/.sck/`:

   ```
   $HOME/.sck/
     ├── term_chat_dev.json
     ├── term_chat_staging.json
     └── term_chat_prod.json
   ```

   Example `term_chat_dev.json`:

   ```json
   {
     "postgres_url": "postgres://user:password@localhost:5432/term?sslmode=disable",
     "redis_url": "localhost:6379",
     "encryption_key": "base64-encoded-32-byte-key"
   }
   ```

3. **Run database migrations**

   ```sh
   migrate -database "postgres://user:pass@localhost:5432/term?sslmode=disable" \
           -path db/migrations up
   ```

4. **Start dependencies**

   ```sh
   systemctl start postgresql
   systemctl start redis
   ```

5. **Build**

   ```sh
   go build -o termchat ./cmd/main.go
   ```

---

## 🖥️ Running

### Server

```sh
./termchat --env dev
```

- TCP chat server: **:9000**
- HTTP API: **:8080** (dev) / **:8194** (prod)

### TUI Client (recommended)

```sh
./termchat --mode client --host localhost --port 9000
```

Or with `go run`:

```sh
go run ./cmd/main.go --mode client --host localhost --port 9000
```

### Raw Telnet (manual testing)

```sh
telnet localhost 9000
```

---

## 💬 Client Usage

### Auth Screen

On launch you'll see the login form. Use `[Tab]` or `[↑/↓]` to move between fields, `[Enter]` to submit, and `[F1]` to toggle between **Sign In** and **Create Account**.

### Main Dashboard

After login the dashboard shows:
- **Sidebar** — your chat partners (rooms), incoming notifications
- **Activity panel** — scrollable message/command output
- **Command bar** — type any command and press `[Enter]`

### Chat Screen (`/chat`)

Opens a persistent chat with full history. History messages appear dimmed; a `── live ──` divider marks where live messages begin. Orange border indicates messages are saved to the database.

### TempChat Screen (`/tempchat`)

Ephemeral real-time chat. Purple border, `● live` indicator. Nothing is saved — messages disappear when the session ends.

---

## 📋 Command Reference

| Command | Description |
|---------|-------------|
| `/register <email> <username> <password>` | Create a new account |
| `/login <email> <password>` | Sign in |
| `/room` | Refresh your chat partners list |
| `/chat <username>` | Open persistent chat with history + live messages |
| `/tempchat <username>` | Start ephemeral real-time chat (not saved) |
| `/send <username> <message>` | Send a one-off direct message |
| `/search <prefix>` | Search users by name |
| `/clear` | Clear messages and notifications |
| `/exit` | Leave current chat or disconnect |
| `/help` | Show command list in the activity panel |
| `[↑/↓]` | Navigate command history |
| `[F1]` | Toggle login / register on auth screen |
| `[Ctrl+C]` | Quit |

---

## 🔔 Notification System

TermChat notifies you in real time when:
- Another user opens a `/chat` with you → `◆ @user wants to /chat`
- Another user opens a `/tempchat` with you → `◆ @user /tempchat`
- Someone sends you a `/send` message → `◆ msg from @user`

Notifications appear in the sidebar and the top bar shows a `🔔 N` badge. Your terminal will also ring the bell (`\a`) so you notice even if the window is in the background.

Notifications are delivered via a per-user Redis channel (`notify:<username>`) subscribed immediately on login.

---

## 🔐 Security

- Passwords: **bcrypt** hashing
- Messages: **AES-256-GCM** encryption at rest
- Session deduplication: unique session ID per TCP connection prevents message echo across multiple open tabs

---

## 🌐 Tailscale Deployment (Recommended)

Run TermChat securely across machines using [Tailscale](https://tailscale.com).

```sh
# Install Tailscale on both machines
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up

# Start the server
./termchat --env dev

# Find your server's Tailscale IP
tailscale ip -4
# → e.g. 100.72.55.34

# Connect from another machine
./termchat --mode client --host 100.72.55.34 --port 9000
```

With MagicDNS enabled:

```sh
./termchat --mode client --host myserver.tailnet-name.ts.net --port 9000
```

✅ No open ports. No public exposure. Works over any network.

---

## 📜 License

MIT License © 2025 [Abhinav Ashish](https://github.com/Abhinav7903)
