# TermChat 🖥️💬
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/Abhinav7903/TermChat)
[![Release](https://img.shields.io/github/v/release/Abhinav7903/TermChat)](https://github.com/Abhinav7903/TermChat/releases)
[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**TermChat** is a terminal-based chat application with a fully interactive **Bubble Tea TUI client** and a robust **TCP/Telnet server**. It supports real-time messaging, persistent chat history, ephemeral chats, live notifications, and encrypted message storage — all backed by **PostgreSQL** and **Redis**.

![Fast Demo](https://github.com/Abhinav7903/TermChat/blob/main/demo.gif)

---

## ✨ Features

- 🎨 **Rich Terminal UI**
  - Full Bubble Tea TUI client with Lip Gloss styling
  - Dark theme with colored panels, badges, and live indicators
  - Keyboard-driven with command history navigation
  - **Vim-like scrolling** (Ctrl+K/J) and **Dynamic Themes**
- 👥 **Advanced Communities**
  - **Group Chats** with `/create`, `/join`, and `/leave`
  - **Global Room** (`/global`) for all users to connect instantly
  - **Owner Controls** for kicking and inviting members
- 🔐 **Security**
  - Register & login with email + password
  - Password hashing with **bcrypt**
  - **AES-256-GCM** message storage at rest
- ⚡ **Real-Time Communication**
  - Persistent chat with history via **PostgreSQL** + **Redis pub/sub**
  - **Rich Notifications**: Native terminal bell (`\a`) and real-time popups
  - Message **Reactions** using `/react <emoji>`
- ⌨️ **Shell Integration**
  - Pipe terminal output directly to chat using `--mode send`
- 🔎 **Search & Discovery**
  - Search users by name prefix
  - Room/partner list on login

---

## 📂 Project Structure

```text
/
├── cmd/                # Entry point (main.go)
├── client/             # Bubble Tea TUI client & CLI mode
├── server/             # HTTP & TCP server logic
├── db/                 # Database migrations, Postgres & Redis drivers
├── pkg/                # Repository interfaces & domain logic
├── factory/            # Data models
├── scripts/            # Helper scripts (key generation, etc.)
├── themes/             # JSON theme files
└── utils/              # Encryption & encoding utilities
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
   TermChat looks for a `term_chat_dev.json` file in the root or `$HOME/.sck/`.

   **Step A: Create config**
   ```sh
   cp term_chat_dev.json.example term_chat_dev.json
   ```

   **Step B: Generate encryption key**
   ```sh
   # Run the provided generator script
   go run scripts/gen_key.go
   # Paste the output into your term_chat_dev.json
   ```

   **Alternative: Environment Variables**
   ```sh
   export POSTGRES_URL="postgres://user:pass@localhost:5432/term?sslmode=disable"
   export REDIS_URL="redis://localhost:6379"
   export ENCRYPTION_KEY="your-generated-base64-key"
   ```

3. **Run database migrations**
   ```sh
   migrate -database "$POSTGRES_URL" -path db/migrations up
   ```

4. **Build**
   ```sh
   go build -o termchat ./cmd/main.go
   ```

---

## 🖥️ Running

### Server
```sh
./termchat --env dev
```

### TUI Client (interactive)
```sh
./termchat --mode client --host localhost --port 9000
```

### CLI Send Mode (pipe output)
```sh
echo "Server logs: $(date)" | ./termchat --mode send --email user@ex.com --pass 123 --to @admin
```

---

## 📋 Command Reference (Inside TUI)

| Command | Description |
|---------|-------------|
| `/create <name>` | Create a new group chat |
| `/join <name>` | Join an existing group |
| `/group <name>` | Switch to a group chat |
| `/global` | Jump into the global community room |
| `/chat <user>` | Open private chat with history |
| `/tempchat <user>` | Ephemeral chat (no history) |
| `/react <emoji>` | React to the last message in current chat |
| `/theme <path>` | Load a `.json` theme file |
| `/invite <grp> <usr>` | (Owner) Invite user to group |
| `/kick <grp> <usr>` | (Owner) Kick user from group |
| `/clear` | Clear dashboard and notifications |
| `/exit` | Leave current chat or disconnect |
| `Ctrl+K/J` | Scroll chat history |

---

## 💡 Tips & Tricks

- **Message Reactions**: Use `/react 👍` while inside a chat to attach an emoji to the most recent message. These are saved and visible to everyone in the history.

---

## 🛠️ Troubleshooting

### "pq: password authentication failed for user ..."
Ensure `POSTGRES_URL` is set correctly in your JSON config or environment variables. By default, Postgres attempts to connect using your OS username.

### "Config File Not Found"
Copy the example file: `cp term_chat_dev.json.example term_chat_dev.json`.

---

## 📜 License
MIT License © 2025 [Abhinav Ashish](https://github.com/Abhinav7903)
