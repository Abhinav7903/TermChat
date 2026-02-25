package client

import (
	"fmt"
	"net"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type appState int

const (
	stateConnecting appState = iota
	stateAuth
	stateRegister
	stateMenu
	stateChat    // /tempchat — ephemeral, no history
	stateHistory // /chat    — persistent, with history + live
	stateSearch  // showing search results
)

type serverLineMsg string
type connectedMsg struct{ conn net.Conn }
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

// ChatMessage holds a parsed chat message for display
type ChatMessage struct {
	sender    string
	timestamp string
	content   string
	isSelf    bool
	isSystem  bool
	isHistory bool // came from HIST (dimmed display)
}

// Notification shown in the sidebar / banner
type Notification struct {
	from     string
	chatType string // "chat" or "tempchat" or "msg"
}

type Model struct {
	host string
	port string
	conn net.Conn
	inCh chan string

	state       appState
	currentUser string

	// Auth fields
	authInputs  [3]textinput.Model // 0=email, 1=password, 2=username
	activeInput int
	isRegister  bool

	// Main command input
	msgInput textinput.Model
	viewport viewport.Model
	spinner  spinner.Model

	// Data
	messages      []ChatMessage
	rooms         []string
	searchResult  []string
	notifications []Notification // incoming chat/msg notifications
	chatPartner   string         // active chat or tempchat partner
	chatReady     bool           // true after OK CHAT READY received

	width    int
	height   int
	banner   string
	bannerOK bool

	// Command history
	history    []string
	historyIdx int

	// Set to true for one frame when a notification arrives → emits terminal bell
	needsBell bool
}

func InitialModel(host, port string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	email := textinput.New()
	email.Placeholder = "you@example.com"
	email.CharLimit = 100
	email.Focus()

	password := textinput.New()
	password.Placeholder = "password"
	password.EchoMode = textinput.EchoPassword
	password.CharLimit = 100

	username := textinput.New()
	username.Placeholder = "username"
	username.CharLimit = 50

	cmdInput := textinput.New()
	cmdInput.Placeholder = "/help for commands"
	cmdInput.CharLimit = 500

	return Model{
		host:       host,
		port:       port,
		state:      stateConnecting,
		spinner:    s,
		authInputs: [3]textinput.Model{email, password, username},
		msgInput:   cmdInput,
		historyIdx: -1,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		connectCmd(m.host, m.port),
		textinput.Blink,
	)
}

func connectCmd(host, port string) tea.Cmd {
	return func() tea.Msg {
		conn, err := Connect(host, port)
		if err != nil {
			return errMsg{err}
		}
		return connectedMsg{conn}
	}
}

// bellCmd writes ASCII BEL to the terminal for notification sound.
func bellCmd() tea.Cmd {
	return func() tea.Msg {
		fmt.Print("\a")
		return nil
	}
}

func waitForServerLine(ch chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return errMsg{fmt.Errorf("server closed connection")}
		}
		return serverLineMsg(line)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport = viewport.New(m.width-38, m.height-12)
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case connectedMsg:
		m.conn = msg.conn
		m.inCh = make(chan string, 128)
		go ReadLoop(m.conn, m.inCh)
		m.state = stateAuth
		return m, tea.Batch(waitForServerLine(m.inCh), textinput.Blink)

	case serverLineMsg:
		m.needsBell = false
		m = m.handleServerLine(string(msg))
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		cmds := []tea.Cmd{waitForServerLine(m.inCh)}
		if m.needsBell {
			cmds = append(cmds, bellCmd())
			m.needsBell = false
		}
		return m, tea.Batch(cmds...)

	case errMsg:
		m.banner = "✗ " + msg.Error()
		m.bannerOK = false
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleServerLine(raw string) Model {
	line := strings.TrimSpace(raw)

	if strings.HasPrefix(line, "> ") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "> "))
	}
	if line == "" || line == ">" {
		return m
	}

	parts := strings.Fields(line)
	if len(parts) == 0 {
		return m
	}

	switch parts[0] {

	// ── OK ───────────────────────────────────────────────────────────────────
	case "OK":
		if len(parts) < 2 {
			return m
		}
		switch parts[1] {

		case "LOGIN":
			if len(parts) >= 3 {
				m.currentUser = parts[2]
				m.state = stateMenu
				m.banner = "✓ Logged in as " + m.currentUser
				m.bannerOK = true
				m.rooms = []string{}
				m.messages = []ChatMessage{}
				m.notifications = []Notification{}
				for i := range m.authInputs {
					m.authInputs[i].Blur()
				}
				m.msgInput.Focus()
				go Write(m.conn, "/room")
			}

		case "REGISTER":
			m.banner = "✓ Registered! You can now log in."
			m.bannerOK = true
			m.isRegister = false
			m.state = stateAuth
			m.authInputs[0].Reset()
			m.authInputs[1].Reset()
			m.authInputs[2].Reset()
			m.authInputs[0].Focus()
			m.activeInput = 0

		case "SEND":
			m.banner = "✓ Message sent"
			m.bannerOK = true

		case "EXIT":
			m.banner = "Disconnected"
			m.bannerOK = false

		case "CHAT":
			if len(parts) < 3 {
				return m
			}
			switch parts[2] {
			case "READY":
				m.chatReady = true
				m.messages = append(m.messages, ChatMessage{
					isSystem: true,
					content:  fmt.Sprintf("── live with %s ──────────────────────────────", m.chatPartner),
				})
				m.banner = fmt.Sprintf("✓ Chat with %s — /exit to leave", m.chatPartner)
				m.bannerOK = true
			case "EXIT":
				m.messages = append(m.messages, ChatMessage{
					isSystem: true,
					content:  fmt.Sprintf("Chat with %s ended", m.chatPartner),
				})
				m.state = stateMenu
				m.chatPartner = ""
				m.chatReady = false
				m.msgInput.Focus()
			default:
				// "OK CHAT <partner>" — entering chat mode
				partner := parts[2]
				m.chatPartner = partner
				m.state = stateHistory
				m.chatReady = false
				m.messages = []ChatMessage{}
				m.msgInput.Focus()
				m.banner = fmt.Sprintf("Loading history with %s...", partner)
				m.bannerOK = false
				// Dismiss the notification for this partner since we're now in chat
				m.dismissNotification(partner)
			}

		case "TEMPCHAT":
			if len(parts) < 3 {
				return m
			}
			partner := parts[2]
			if partner == "EXIT" {
				m.messages = append(m.messages, ChatMessage{
					isSystem: true,
					content:  "Temp chat ended",
				})
				m.state = stateMenu
				m.chatPartner = ""
				m.msgInput.Focus()
			} else {
				m.chatPartner = partner
				m.state = stateChat
				m.messages = []ChatMessage{}
				m.banner = fmt.Sprintf("✓ Temp chatting with %s — /exit to leave", partner)
				m.bannerOK = true
				m.msgInput.Focus()
				m.dismissNotification(partner)
			}
		}

	// ── ERR ──────────────────────────────────────────────────────────────────
	case "ERR":
		m.banner = "✗ " + strings.Join(parts[1:], " ")
		m.bannerOK = false

	// ── ROOM ─────────────────────────────────────────────────────────────────
	case "ROOM":
		if len(parts) == 2 && parts[1] != "NONE" {
			m.rooms = append(m.rooms, parts[1])
		}

	// ── HIST — chat history line ──────────────────────────────────────────────
	// Format: HIST <timestamp>|<sender>|<content>
	case "HIST":
		payload := strings.Join(parts[1:], " ")
		segs := strings.SplitN(payload, "|", 3)
		if len(segs) == 3 {
			ts, sender, content := segs[0], segs[1], segs[2]
			m.messages = append(m.messages, ChatMessage{
				sender:    sender,
				timestamp: ts,
				content:   content,
				isSelf:    sender == m.currentUser,
				isHistory: true,
			})
		}

	// ── SEARCH ───────────────────────────────────────────────────────────────
	case "SEARCH":
		if len(parts) >= 2 && parts[1] != "NONE" {
			m.searchResult = append(m.searchResult, strings.Join(parts[1:], " "))
		}

	// ── MSG — live message ────────────────────────────────────────────────────
	// Format: MSG <sender>|<timestamp>|<content>
	//    or:  MSG <sender>|/close  (tempchat partner left)
	case "MSG":
		payload := strings.Join(parts[1:], " ")
		segs := strings.SplitN(payload, "|", 3)
		switch len(segs) {
		case 3:
			if segs[2] == "/close" {
				m.messages = append(m.messages, ChatMessage{
					content:  segs[0] + " left the chat",
					isSystem: true,
				})
				m.state = stateMenu
				m.chatPartner = ""
				m.chatReady = false
				m.msgInput.Focus()
			} else {
				m.messages = append(m.messages, ChatMessage{
					sender:    segs[0],
					timestamp: segs[1],
					content:   segs[2],
					isSelf:    segs[0] == m.currentUser,
				})
			}
		case 2:
			if segs[1] == "/close" {
				m.messages = append(m.messages, ChatMessage{
					content:  segs[0] + " left the chat",
					isSystem: true,
				})
				m.state = stateMenu
				m.chatPartner = ""
				m.chatReady = false
				m.msgInput.Focus()
			} else {
				m.messages = append(m.messages, ChatMessage{
					sender:  segs[0],
					content: segs[1],
					isSelf:  segs[0] == m.currentUser,
				})
			}
		default:
			m.messages = append(m.messages, ChatMessage{
				content:  payload,
				isSystem: true,
			})
		}

	// ── NOTIFY — someone wants to chat with you ───────────────────────────────
	// Format: NOTIFY CHAT <sender>
	//         NOTIFY TEMPCHAT <sender>
	//         NOTIFY MSG <sender>
	case "NOTIFY":
		if len(parts) < 3 {
			return m
		}
		notifType := parts[1] // CHAT, TEMPCHAT, or MSG
		from := parts[2]

		// Don't notify about your own actions
		if from == m.currentUser {
			return m
		}

		// Keep at most 5 notifications
		notif := Notification{from: from, chatType: strings.ToLower(notifType)}
		// Avoid duplicate notifications from the same person for the same type
		for _, n := range m.notifications {
			if n.from == from && n.chatType == notif.chatType {
				return m
			}
		}
		m.notifications = append(m.notifications, notif)
		if len(m.notifications) > 5 {
			m.notifications = m.notifications[1:]
		}
		// Ring the terminal bell on next frame
		m.needsBell = true
	}

	return m
}

// dismissNotification removes any notification from the given user
func (m *Model) dismissNotification(from string) {
	filtered := m.notifications[:0]
	for _, n := range m.notifications {
		if n.from != from {
			filtered = append(filtered, n)
		}
	}
	m.notifications = filtered
}

// renderMessages builds raw string content for the viewport
func (m Model) renderMessages() string {
	if len(m.messages) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, msg := range m.messages {
		sb.WriteString(formatChatMessage(msg, m.currentUser))
		sb.WriteString("\n")
	}
	return sb.String()
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch m.state {

	// ── AUTH / REGISTER ──────────────────────────────────────────────────────
	case stateAuth, stateRegister:
		numFields := 2
		if m.isRegister {
			numFields = 3
		}

		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyTab, tea.KeyDown:
			m.authInputs[m.activeInput].Blur()
			m.activeInput = (m.activeInput + 1) % numFields
			m.authInputs[m.activeInput].Focus()
			return m, textinput.Blink

		case tea.KeyShiftTab, tea.KeyUp:
			m.authInputs[m.activeInput].Blur()
			m.activeInput = (m.activeInput - 1 + numFields) % numFields
			m.authInputs[m.activeInput].Focus()
			return m, textinput.Blink

		case tea.KeyF1:
			m.isRegister = !m.isRegister
			if m.isRegister {
				m.state = stateRegister
			} else {
				m.state = stateAuth
			}
			m.activeInput = 0
			for i := range m.authInputs {
				m.authInputs[i].Blur()
			}
			m.authInputs[0].Focus()
			m.banner = ""
			return m, textinput.Blink

		case tea.KeyEnter:
			if m.isRegister {
				email := strings.TrimSpace(m.authInputs[0].Value())
				username := strings.TrimSpace(m.authInputs[2].Value())
				pass := strings.TrimSpace(m.authInputs[1].Value())
				if email != "" && pass != "" && username != "" {
					go Write(m.conn, fmt.Sprintf("/register %s %s %s", email, username, pass))
					m.banner = "Registering..."
					m.bannerOK = false
				} else {
					m.banner = "✗ Fill in all fields"
					m.bannerOK = false
				}
			} else {
				email := strings.TrimSpace(m.authInputs[0].Value())
				pass := strings.TrimSpace(m.authInputs[1].Value())
				if email != "" && pass != "" {
					go Write(m.conn, fmt.Sprintf("/login %s %s", email, pass))
					m.banner = "Authenticating..."
					m.bannerOK = false
				} else {
					m.banner = "✗ Email and password required"
					m.bannerOK = false
				}
			}
			return m, nil

		default:
			var cmd tea.Cmd
			m.authInputs[m.activeInput], cmd = m.authInputs[m.activeInput].Update(msg)
			return m, cmd
		}

	// ── MENU / SEARCH ────────────────────────────────────────────────────────
	case stateMenu, stateSearch:
		switch msg.Type {
		case tea.KeyCtrlC:
			go Write(m.conn, "/exit")
			return m, tea.Quit

		case tea.KeyUp:
			if len(m.history) > 0 {
				m.historyIdx++
				if m.historyIdx >= len(m.history) {
					m.historyIdx = len(m.history) - 1
				}
				m.msgInput.SetValue(m.history[len(m.history)-1-m.historyIdx])
				m.msgInput.CursorEnd()
			}
			return m, nil

		case tea.KeyDown:
			if m.historyIdx > 0 {
				m.historyIdx--
				m.msgInput.SetValue(m.history[len(m.history)-1-m.historyIdx])
				m.msgInput.CursorEnd()
			} else {
				m.historyIdx = -1
				m.msgInput.Reset()
			}
			return m, nil

		case tea.KeyEnter:
			raw := strings.TrimSpace(m.msgInput.Value())
			if raw == "" {
				return m, nil
			}
			m.history = append(m.history, raw)
			m.historyIdx = -1
			m.searchResult = []string{}
			m.msgInput.Reset()

			fields := strings.Fields(raw)
			switch fields[0] {
			case "/search":
				m.state = stateSearch
				go Write(m.conn, raw)
			case "/room":
				m.rooms = []string{}
				go Write(m.conn, raw)
			case "/tempchat":
				go Write(m.conn, raw)
			case "/chat":
				go Write(m.conn, raw)
			case "/send":
				if len(fields) >= 3 {
					m.messages = append(m.messages, ChatMessage{
						sender:  m.currentUser,
						content: strings.Join(fields[2:], " "),
						isSelf:  true,
					})
				}
				go Write(m.conn, raw)
			case "/help":
				m.messages = append(m.messages, ChatMessage{
					isSystem: true,
					content:  helpText(),
				})
			case "/clear":
				m.messages = []ChatMessage{}
				m.searchResult = []string{}
				m.notifications = []Notification{}
			default:
				go Write(m.conn, raw)
			}

			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
			return m, nil

		default:
			var cmd tea.Cmd
			m.msgInput, cmd = m.msgInput.Update(msg)
			return m, cmd
		}

	// ── PERSISTENT CHAT (/chat) ──────────────────────────────────────────────
	case stateHistory:
		switch msg.Type {
		case tea.KeyCtrlC:
			go Write(m.conn, "/exit")
			m.state = stateMenu
			m.chatPartner = ""
			m.chatReady = false
			m.msgInput.Focus()
			return m, nil

		case tea.KeyUp:
			if len(m.history) > 0 {
				m.historyIdx++
				if m.historyIdx >= len(m.history) {
					m.historyIdx = len(m.history) - 1
				}
				m.msgInput.SetValue(m.history[len(m.history)-1-m.historyIdx])
				m.msgInput.CursorEnd()
			}
			return m, nil

		case tea.KeyDown:
			if m.historyIdx > 0 {
				m.historyIdx--
				m.msgInput.SetValue(m.history[len(m.history)-1-m.historyIdx])
				m.msgInput.CursorEnd()
			} else {
				m.historyIdx = -1
				m.msgInput.Reset()
			}
			return m, nil

		case tea.KeyEnter:
			raw := strings.TrimSpace(m.msgInput.Value())
			if raw == "" {
				return m, nil
			}
			m.history = append(m.history, raw)
			m.historyIdx = -1
			m.msgInput.Reset()

			if raw == "/exit" {
				go Write(m.conn, "/exit")
				return m, nil
			}

			if !m.chatReady {
				m.banner = "⏳ Loading history, please wait..."
				m.bannerOK = false
				return m, nil
			}

			// Optimistic echo — server will NOT echo this back
			m.messages = append(m.messages, ChatMessage{
				sender:  m.currentUser,
				content: raw,
				isSelf:  true,
			})
			go Write(m.conn, raw)
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
			return m, nil

		default:
			var cmd tea.Cmd
			m.msgInput, cmd = m.msgInput.Update(msg)
			return m, cmd
		}

	// ──  TEMPCHAT ────────────────────────────────────────────────────
	case stateChat:
		switch msg.Type {
		case tea.KeyCtrlC:
			go Write(m.conn, "/exit")
			m.state = stateMenu
			m.chatPartner = ""
			m.msgInput.Focus()
			return m, nil

		case tea.KeyUp:
			if len(m.history) > 0 {
				m.historyIdx++
				if m.historyIdx >= len(m.history) {
					m.historyIdx = len(m.history) - 1
				}
				m.msgInput.SetValue(m.history[len(m.history)-1-m.historyIdx])
				m.msgInput.CursorEnd()
			}
			return m, nil

		case tea.KeyDown:
			if m.historyIdx > 0 {
				m.historyIdx--
				m.msgInput.SetValue(m.history[len(m.history)-1-m.historyIdx])
				m.msgInput.CursorEnd()
			} else {
				m.historyIdx = -1
				m.msgInput.Reset()
			}
			return m, nil

		case tea.KeyEnter:
			raw := strings.TrimSpace(m.msgInput.Value())
			if raw == "" {
				return m, nil
			}
			m.history = append(m.history, raw)
			m.historyIdx = -1
			m.msgInput.Reset()

			if raw == "/exit" {
				go Write(m.conn, "/exit")
				m.state = stateMenu
				m.chatPartner = ""
				m.msgInput.Focus()
				return m, nil
			}

			// Optimistic echo — server will NOT echo this back
			m.messages = append(m.messages, ChatMessage{
				sender:  m.currentUser,
				content: raw,
				isSelf:  true,
			})
			go Write(m.conn, raw)
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
			return m, nil

		default:
			var cmd tea.Cmd
			m.msgInput, cmd = m.msgInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) View() string {
	return Render(m)
}

func helpText() string {
	return `Available commands:
  /register <email> <username> <password>
  /login <email> <password>
  /room                    — list your chat partners
  /chat <username>         — open chat with history + live messages
  /tempchat <username>     — ephemeral real-time chat (no history saved)
  /send <username> <msg>   — send a one-off message
  /search <prefix>         — search users by name
  /clear                   — clear messages and notifications
  /exit                    — disconnect (or leave chat)
  [↑/↓]                   — navigate command history`
}
