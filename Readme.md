# TermChat 🖥️💬

**TermChat** is a terminal-based chat application supporting **real-time messaging** via **Telnet (TCP)** and **HTTP APIs**.  
It features **user registration, login, encrypted messaging, chat rooms, and temporary chats**, all backed by **PostgreSQL** and **Redis**.

---

## ✨ Features

- 🔐 **Secure authentication**
  - Register & login users
  - Password hashing with **bcrypt**
- 🔒 **Encrypted messages**
  - AES-256-GCM message storage
- ⚡ **Real-time communication**
  - Chat via **Telnet (TCP)**
  - RESTful **HTTP API**
- 👥 **Chat types**
  - Personal one-to-one chats
  - Group chat rooms
  - Temporary chats via Redis pub/sub
- 🔎 **Search & history**
  - Search users
  - Retrieve past conversations

---

## 📂 Project Structure

```text
cmd/                 # Entry point (main.go)
db/
  migrations/        # SQL migrations
  postgres/          # PostgreSQL logic
  redis/             # Redis logic
factory/             # Data models (User, Message)
pkg/
  message/           # Message repository interface
  users/             # User repository interface & utils
server/              # HTTP & TCP server logic
utils/               # Encryption utilities
```

---

## 🚀 Getting Started

### ✅ Prerequisites

- [Go 1.23+](https://go.dev/dl/)
- [PostgreSQL](https://www.postgresql.org/)
- [Redis](https://redis.io/)

---

### ⚙️ Setup

1. **Clone the repo**

   ```sh
   git clone https://github.com/Abhinav7903/TermChat.git
   cd TermChat
   ```

2. **Configure environment**

   - Place config files in:

     ```
     $HOME/.sck/
       ├── term_chat_dev.json
       ├── term_chat_staging.json
       └── term_chat_prod.json
     ```

   - Example `term_chat_dev.json`:

     ```json
     {
       "postgres_url": "postgres://user:password@localhost:5432/term?sslmode=disable",
       "redis_url": "localhost:6379",
       "encryption_key": "base64-encoded-32-byte-key"
     }
     ```

3. **Run database migrations**

   ```
   migrate -database "postgres://user:pass@localhost:5432/term?sslmode=disable" -path db/migrations up
   ```

4. **Start dependencies**

   ```sh
   # Start PostgreSQL & Redis
   systemctl start postgresql
   systemctl start redis
   ```

5. **Build and run the server**

   ```sh
   go build -o termchat ./cmd/main.go
   ./termchat --env dev
   ```

   - HTTP server runs on **:8080** (dev) / **:8194** (prod)
   - TCP chat server runs on **:9000**

---

## 💬 Usage

### Telnet CLI (TCP)

Connect to the chat server:

```sh
telnet localhost 9000
```

Available commands:

- `/register <email> <username> <password>`
- `/login <email> <password>`
- `/chat <username>` → Start personal chat
- `/send <username> <message>` → Send message
- `/room` → Join group chat room
- `/temp <username>` → Start temporary chat
- `/search <username_prefix>` → Search users
- `/last <username>` → Show last chat
- `/exit` → Quit
- `/help` → Show available commands

---

### HTTP API

Basic health check:

```sh
curl http://localhost:8080/ping
```

(More endpoints coming soon: registration, login, messages, etc.)

---

## 🔐 Security

- Passwords: **bcrypt** hashing
- Messages: **AES-256-GCM encryption**
- Session tokens: stored in **Redis**

---

## 📜 License

MIT License © 2025 [Abhinav Ashish](https://github.com/Abhinav7903)

---
### 🌐 Tailscale Deployment (Recommended)

You can run TermChat securely over [Tailscale](https://tailscale.com), a mesh VPN that gives each machine a private IP (e.g., `100.x.x.x`).

#### 1. Install Tailscale

On both server and client machines:

```sh
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up
```

#### 2. Start TermChat on Server

```sh
./termchat --env dev
```

By default:

- **TCP Chat:** `:9000`
- **HTTP API:** `:8080`

#### 3. Find Your Server’s Tailscale IP

```sh
tailscale ip -4
```

Example output: `100.72.55.34`

#### 4. Connect from Another Device

From any client on the same Tailnet:

- **Chat via Telnet:**

    ```sh
    telnet 100.72.55.34 9000
    ```

- **Check HTTP health:**

    ```sh
    curl http://100.72.55.34:8080/ping
    ```

#### 5. (Optional) Use MagicDNS

If enabled in Tailscale admin:

```sh
telnet myserver.tailnet-name.ts.net 9000
```

✅ TermChat works securely without opening ports to the public internet.
