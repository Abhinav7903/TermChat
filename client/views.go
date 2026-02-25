package client

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// ─── Color Palette ────────────────────────────────────────────────────────────
var (
	colorBg      = lipgloss.Color("#0d1117")
	colorSurface = lipgloss.Color("#161b22")
	colorBorder  = lipgloss.Color("#30363d")
	colorAccent  = lipgloss.Color("#58a6ff")
	colorAccent2 = lipgloss.Color("#3fb950")
	colorMuted   = lipgloss.Color("#8b949e")
	colorDanger  = lipgloss.Color("#f85149")
	colorSelf    = lipgloss.Color("#79c0ff")
	colorOther   = lipgloss.Color("#56d364")
	colorSystem  = lipgloss.Color("#6e7681")
	colorWhite   = lipgloss.Color("#e6edf3")
	colorPurple  = lipgloss.Color("#bc8cff")
	colorRoomDot = lipgloss.Color("#3fb950")
	colorHistory = lipgloss.Color("#484f58")
	colorOrange  = lipgloss.Color("#f0883e")
	colorNotif   = lipgloss.Color("#e3b341") // yellow for notifications
)

// ─── Styles ───────────────────────────────────────────────────────────────────
var (
	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			PaddingBottom(1)

	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	styleBorderActive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent)

	styleBorderChat = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorOrange)

	styleMuted    = lipgloss.NewStyle().Foreground(colorMuted)
	styleAccent   = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	styleDanger   = lipgloss.NewStyle().Foreground(colorDanger)
	styleOK       = lipgloss.NewStyle().Foreground(colorAccent2)
	styleWhite    = lipgloss.NewStyle().Foreground(colorWhite)
	styleOrange   = lipgloss.NewStyle().Foreground(colorOrange)
	styleNotif    = lipgloss.NewStyle().Foreground(colorNotif).Bold(true)
	styleNotifDim = lipgloss.NewStyle().Foreground(colorNotif)

	styleSelfMsg   = lipgloss.NewStyle().Foreground(colorSelf).Bold(true)
	styleOtherMsg  = lipgloss.NewStyle().Foreground(colorOther).Bold(true)
	styleHistSelf  = lipgloss.NewStyle().Foreground(colorHistory).Bold(true)
	styleHistOther = lipgloss.NewStyle().Foreground(colorHistory)
	styleSystemMsg = lipgloss.NewStyle().Foreground(colorSystem).Italic(true)
	styleTimestamp = lipgloss.NewStyle().Foreground(colorMuted)
	styleHistTs    = lipgloss.NewStyle().Foreground(colorHistory)
	stylePurple    = lipgloss.NewStyle().Foreground(colorPurple)

	styleLogo = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)
)

// ─── Main Render Dispatcher ───────────────────────────────────────────────────
func Render(m Model) string {
	if m.width == 0 {
		return "\n\n   Loading..."
	}

	switch m.state {
	case stateConnecting:
		return renderConnecting(m)
	case stateAuth, stateRegister:
		return renderAuth(m)
	case stateMenu, stateSearch:
		return renderMenu(m)
	case stateChat:
		return renderChatScreen(m, "tempchat", false)
	case stateHistory:
		return renderChatScreen(m, "chat", true)
	}
	return ""
}

// ─── Connecting ───────────────────────────────────────────────────────────────
func renderConnecting(m Model) string {
	content := fmt.Sprintf(
		"\n%s\n\n%s Connecting to %s:%s...\n",
		styleLogo.Render(logo()),
		m.spinner.View(),
		styleAccent.Render(m.host),
		m.port,
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		styleBorder.Padding(2, 6).Render(content))
}

// ─── Auth ─────────────────────────────────────────────────────────────────────
func renderAuth(m Model) string {
	title := "Sign In"
	if m.isRegister {
		title = "Create Account"
	}

	var fields strings.Builder

	emailLabel := styleMuted.Render("Email")
	if m.activeInput == 0 {
		emailLabel = styleAccent.Render("Email")
	}
	fields.WriteString(emailLabel + "\n")
	fields.WriteString(m.authInputs[0].View() + "\n\n")

	passLabel := styleMuted.Render("Password")
	if m.activeInput == 1 {
		passLabel = styleAccent.Render("Password")
	}
	fields.WriteString(passLabel + "\n")
	fields.WriteString(m.authInputs[1].View() + "\n\n")

	if m.isRegister {
		userLabel := styleMuted.Render("Username")
		if m.activeInput == 2 {
			userLabel = styleAccent.Render("Username")
		}
		fields.WriteString(userLabel + "\n")
		fields.WriteString(m.authInputs[2].View() + "\n\n")
	}

	fields.WriteString(renderBanner(m) + "\n\n")

	submitLabel := "  [Enter] Sign In  "
	if m.isRegister {
		submitLabel = "  [Enter] Register  "
	}
	toggle := "  [F1] Create account  "
	if m.isRegister {
		toggle = "  [F1] Sign in instead  "
	}

	actions := lipgloss.JoinHorizontal(lipgloss.Center,
		lipgloss.NewStyle().
			Background(colorAccent).
			Foreground(colorBg).
			Bold(true).
			Padding(0, 1).
			Render(submitLabel),
		"  ",
		styleMuted.Render(toggle),
	)
	fields.WriteString(actions)

	box := styleBorderActive.
		Width(50).
		Padding(1, 3).
		Render(
			styleHeader.Render("⬡ TermChat  /  "+title) +
				"\n" +
				fields.String(),
		)

	full := lipgloss.JoinVertical(lipgloss.Center,
		styleLogo.Render(logo()),
		"",
		box,
		"",
		styleMuted.Render("[Tab/↑↓] navigate fields   [F1] toggle mode   [Ctrl+C] quit"),
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, full)
}

// ─── Menu ─────────────────────────────────────────────────────────────────────
func renderMenu(m Model) string {
	sideW := 30
	sidebar := renderSidebar(m, sideW)

	mainW := m.width - sideW - 6
	if mainW < 20 {
		mainW = 20
	}
	mainPanel := renderMainPanel(m, mainW)

	top := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, "  ", mainPanel)
	inputBar := renderInputBar(m, m.width-4)
	hdr := renderTopBar(m)

	return lipgloss.JoinVertical(lipgloss.Left, hdr, "", top, "", inputBar)
}

func renderTopBar(m Model) string {
	left := styleAccent.Render("⬡ TermChat")

	// Show notification bell if there are pending notifications
	notifIndicator := ""
	if len(m.notifications) > 0 {
		notifIndicator = " " + styleNotif.Render(fmt.Sprintf("🔔 %d", len(m.notifications)))
	}

	right := fmt.Sprintf("%s  %s%s",
		styleMuted.Render(time.Now().Format("15:04")),
		stylePurple.Render("@"+m.currentUser),
		notifIndicator,
	)
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 0 {
		gap = 0
	}
	return lipgloss.NewStyle().
		Background(colorSurface).
		Foreground(colorWhite).
		Width(m.width).
		Padding(0, 1).
		Render(left + strings.Repeat(" ", gap) + right)
}

func renderSidebar(m Model, w int) string {
	var sb strings.Builder

	// User info
	sb.WriteString(styleAccent.Render("YOU") + "\n")
	sb.WriteString(stylePurple.Render("@"+m.currentUser) + "\n")
	sb.WriteString(styleMuted.Render("● connected") + "\n\n")

	// ── Notifications ──────────────────────────────────────────
	if len(m.notifications) > 0 {
		sb.WriteString(styleNotif.Render("🔔 INCOMING") + "\n")
		for _, n := range m.notifications {
			var icon, label string
			switch n.chatType {
			case "chat":
				icon = styleOrange.Render("◆")
				label = styleNotifDim.Render(fmt.Sprintf(" @%s wants to /chat", n.from))
			case "tempchat":
				icon = lipgloss.NewStyle().Foreground(colorPurple).Render("◆")
				label = styleNotifDim.Render(fmt.Sprintf(" @%s /tempchat", n.from))
			case "msg":
				icon = lipgloss.NewStyle().Foreground(colorAccent).Render("◆")
				label = styleNotifDim.Render(fmt.Sprintf(" msg from @%s", n.from))
			default:
				icon = styleMuted.Render("◆")
				label = styleNotifDim.Render(" @" + n.from)
			}
			sb.WriteString(icon + label + "\n")
		}
		sb.WriteString("\n")
	}

	// ── Rooms ──────────────────────────────────────────────────
	sb.WriteString(styleHeader.Render("ROOMS") + "\n")
	if len(m.rooms) == 0 {
		sb.WriteString(styleMuted.Render("  none yet") + "\n")
	} else {
		for _, r := range m.rooms {
			dot := lipgloss.NewStyle().Foreground(colorRoomDot).Render("●")
			sb.WriteString(fmt.Sprintf(" %s %s\n", dot, styleWhite.Render(r)))
		}
	}
	sb.WriteString("\n")

	// ── Search results ─────────────────────────────────────────
	if m.state == stateSearch && len(m.searchResult) > 0 {
		sb.WriteString(styleHeader.Render("SEARCH") + "\n")
		for _, r := range m.searchResult {
			sb.WriteString(fmt.Sprintf("  %s\n", styleWhite.Render(r)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(styleMuted.Render(strings.Repeat("─", w-4)) + "\n")
	sb.WriteString(styleMuted.Render("/chat") + " " + styleMuted.Render("<user>") + " w/ history\n")
	sb.WriteString(styleMuted.Render("/tempchat") + " " + styleMuted.Render("<user>") + " ephemeral\n")
	sb.WriteString(styleMuted.Render("/help") + " for all commands\n")

	return styleBorder.
		Width(w).
		Height(m.height-6).
		Padding(0, 1).
		Render(sb.String())
}

func renderMainPanel(m Model, w int) string {
	title := "Activity"
	if m.state == stateSearch {
		title = "Search Results"
	}

	m.viewport.Width = w - 4
	m.viewport.Height = m.height - 12
	m.viewport.SetContent(m.renderMessages())

	content := styleHeader.Render("▸ "+title) + "\n" + m.viewport.View()

	return styleBorder.
		Width(w).
		Height(m.height-6).
		Padding(0, 1).
		Render(content)
}

func renderInputBar(m Model, w int) string {
	banner := renderBanner(m)
	prompt := styleAccent.Render("❯ ")
	inputLine := prompt + m.msgInput.View()
	hints := styleMuted.Render("[Enter] send  [↑↓] history  [Ctrl+C] quit")
	gap := w - lipgloss.Width(hints) - 4
	if gap < 0 {
		gap = 0
	}
	bottom := strings.Repeat(" ", gap) + hints

	inner := inputLine + "\n" + bottom
	if banner != "" {
		inner = banner + "\n" + inputLine + "\n" + bottom
	}

	return styleBorderActive.
		Width(w).
		Padding(0, 1).
		Render(inner)
}

// ─── Chat & History Screen ────────────────────────────────────────────────────
func renderChatScreen(m Model, chatType string, withHistory bool) string {
	hdr := renderChatTopBar(m, chatType, withHistory)

	vpW := m.width - 4
	vpH := m.height - 8

	m.viewport.Width = vpW - 2
	m.viewport.Height = vpH - 2
	m.viewport.SetContent(m.renderMessages())

	var msgBox string
	if withHistory {
		loadingLine := ""
		if !m.chatReady {
			loadingLine = "\n" + styleOrange.Render("  ⏳ loading history...")
		}
		msgBox = styleBorderChat.
			Width(vpW).
			Height(vpH).
			Padding(0, 1).
			Render(m.viewport.View() + loadingLine)
	} else {
		msgBox = styleBorder.
			Width(vpW).
			Height(vpH).
			Padding(0, 1).
			Render(m.viewport.View())
	}

	// Input bar
	var prompt string
	if withHistory {
		prompt = styleOrange.Render("❯ ")
	} else {
		prompt = stylePurple.Render("❯ ")
	}

	inputLine := prompt + m.msgInput.View()
	banner := renderBanner(m)
	hints := styleMuted.Render("[Enter] send  [↑↓] history  /exit to leave  [Ctrl+C] quit")
	gap := vpW - lipgloss.Width(hints) - 4
	if gap < 0 {
		gap = 0
	}
	bottom := strings.Repeat(" ", gap) + hints

	inner := inputLine + "\n" + bottom
	if banner != "" {
		inner = banner + "\n" + inputLine + "\n" + bottom
	}

	var inputBox string
	if withHistory {
		inputBox = styleBorderChat.Width(vpW).Padding(0, 1).Render(inner)
	} else {
		inputBox = styleBorderActive.Width(vpW).Padding(0, 1).Render(inner)
	}

	return lipgloss.JoinVertical(lipgloss.Left, hdr, msgBox, inputBox)
}

func renderChatTopBar(m Model, chatType string, withHistory bool) string {
	var badge, status string

	if withHistory {
		badge = lipgloss.NewStyle().
			Background(colorOrange).
			Foreground(colorBg).
			Bold(true).
			Padding(0, 1).
			Render("chat")
		if m.chatReady {
			status = styleOK.Render("● saved")
		} else {
			status = styleOrange.Render("● loading")
		}
	} else {
		badge = lipgloss.NewStyle().
			Background(colorPurple).
			Foreground(colorBg).
			Bold(true).
			Padding(0, 1).
			Render("tempchat")
		status = styleOK.Render("● live")
	}

	left := styleAccent.Render("⬡ TermChat  ›  ") +
		stylePurple.Render("@"+m.chatPartner) +
		"  " + badge

	right := styleMuted.Render(time.Now().Format("15:04")) + "  " + status

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 0 {
		gap = 0
	}
	return lipgloss.NewStyle().
		Background(colorSurface).
		Foreground(colorWhite).
		Width(m.width).
		Padding(0, 1).
		Render(left + strings.Repeat(" ", gap) + right)
}

// ─── Message Formatting ───────────────────────────────────────────────────────
func formatChatMessage(msg ChatMessage, currentUser string) string {
	if msg.isSystem {
		return styleSystemMsg.Render("  · " + msg.content)
	}

	if msg.isHistory {
		var nameStyle lipgloss.Style
		if msg.isSelf || msg.sender == currentUser {
			nameStyle = styleHistSelf
		} else {
			nameStyle = styleHistOther
		}
		ts := ""
		if msg.timestamp != "" {
			ts = styleHistTs.Render(" " + shortTimestamp(msg.timestamp))
		}
		return fmt.Sprintf("%s%s  %s",
			nameStyle.Render(msg.sender),
			ts,
			styleMuted.Render(msg.content),
		)
	}

	var nameStyle lipgloss.Style
	if msg.isSelf || msg.sender == currentUser {
		nameStyle = styleSelfMsg
	} else {
		nameStyle = styleOtherMsg
	}

	ts := ""
	if msg.timestamp != "" {
		ts = styleTimestamp.Render(" " + shortTimestamp(msg.timestamp))
	}

	return fmt.Sprintf("%s%s  %s",
		nameStyle.Render(msg.sender),
		ts,
		styleWhite.Render(msg.content),
	)
}

func shortTimestamp(ts string) string {
	if len(ts) >= 16 {
		return ts[11:16]
	}
	return ts
}

// ─── Banner ───────────────────────────────────────────────────────────────────
func renderBanner(m Model) string {
	if m.banner == "" {
		return ""
	}
	if m.bannerOK {
		return styleOK.Render(m.banner)
	}
	low := strings.ToLower(m.banner)
	if strings.Contains(low, "✗") || strings.Contains(low, "error") || strings.Contains(low, "err") {
		return styleDanger.Render(m.banner)
	}
	return styleMuted.Render(m.banner)
}

// ─── Logo ─────────────────────────────────────────────────────────────────────
func logo() string {
	return `
 ████████╗███████╗██████╗ ███╗   ███╗ ██████╗██╗  ██╗ █████╗ ████████╗
    ██╔══╝██╔════╝██╔══██╗████╗ ████║██╔════╝██║  ██║██╔══██╗╚══██╔══╝
    ██║   █████╗  ██████╔╝██╔████╔██║██║     ███████║███████║   ██║   
    ██║   ██╔══╝  ██╔══██╗██║╚██╔╝██║██║     ██╔══██║██╔══██║   ██║   
    ██║   ███████╗██║  ██║██║ ╚═╝ ██║╚██████╗██║  ██║██║  ██║   ██║   
    ╚═╝   ╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝  
`
}

func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max-1]) + "…"
}

var _ = truncate
