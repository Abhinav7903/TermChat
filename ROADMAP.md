# TermChat Roadmap 🚀

This document outlines the planned improvements and new features for TermChat.

## ✅ Completed Features
- [x] TUI Client (Bubble Tea)
- [x] Persistent 1-to-1 Chat (PostgreSQL)
- [x] Ephemeral 1-to-1 Chat (TempChat via Redis)
- [x] Real-time Notifications (Redis Pub/Sub)
- [x] Web Terminal (React + xterm.js)
- [x] User Authentication (bcrypt)
- [x] Encrypted Message Storage (AES-256-GCM at rest)

## 🚧 Upcoming Features (Planned)

### 👥 Group Chats & Communities
- [ ] **Rooms**: Implementation of `/create <name>`, `/join <name>`, and `/leave`.
- [ ] **Global Room**: A default room where all users can chat upon login.
- [ ] **Room Membership**: Role-based access control (owner, member).

### ⌨️ Terminal Integration
- [ ] **Shell Pipes**: Allow piping local shell output directly to TermChat (e.g., `ls | termchat send @admin`).
- [ ] **Shared Snippets**: Dedicated rooms or a `/snippet` command for sharing command-line code blocks.

### 🤖 Intelligence & Bots
- [ ] **AI Assistant**: A `/ask` command to interact with an AI agent (GPT/Gemini) within the chat.
- [ ] **System Bots**: Automated bots for `/help`, `/weather`, `/time`, and server monitoring.

### 🎨 TUI & UX Enhancements
- [ ] **Sidebars & Panels**: Update the Bubble Tea UI to show persistent room lists and online status.
- [ ] **Vim-Mode**: Optional Vim keybindings for navigation and message editing.
- [ ] **Dynamic Themes**: Support for loading `.json` color themes using Lip Gloss.
- [ ] **Typing Indicators**: Show `[...] is typing` in the status bar.
- [ ] **Message Reactions**: Simple emoji-based reactions (e.g., `/react 👍`).

### 🔧 Developer Experience
- [ ] **AGENTS.md**: Context and coding standards for AI agents (Implemented).
- [ ] **Better Testing**: Increased coverage for core messaging and storage logic.

---
*Last Updated: April 2026*
