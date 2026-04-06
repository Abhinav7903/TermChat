package client

import (
	"encoding/json"
	"os"

	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Name    string `json:"name"`
	Bg      string `json:"bg"`
	Surface string `json:"surface"`
	Border  string `json:"border"`
	Accent  string `json:"accent"`
	Accent2 string `json:"accent2"`
	Muted   string `json:"muted"`
	Danger  string `json:"danger"`
	Self    string `json:"self"`
	Other   string `json:"other"`
	System  string `json:"system"`
	White   string `json:"white"`
	Purple  string `json:"purple"`
	History string `json:"history"`
	Orange  string `json:"orange"`
	Notif   string `json:"notif"`
}

var CurrentTheme Theme

func LoadTheme(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &CurrentTheme)
	if err != nil {
		return err
	}
	ApplyTheme()
	return nil
}

func ApplyTheme() {
	colorBg = lipgloss.Color(CurrentTheme.Bg)
	colorSurface = lipgloss.Color(CurrentTheme.Surface)
	colorBorder = lipgloss.Color(CurrentTheme.Border)
	colorAccent = lipgloss.Color(CurrentTheme.Accent)
	colorAccent2 = lipgloss.Color(CurrentTheme.Accent2)
	colorMuted = lipgloss.Color(CurrentTheme.Muted)
	colorDanger = lipgloss.Color(CurrentTheme.Danger)
	colorSelf = lipgloss.Color(CurrentTheme.Self)
	colorOther = lipgloss.Color(CurrentTheme.Other)
	colorSystem = lipgloss.Color(CurrentTheme.System)
	colorWhite = lipgloss.Color(CurrentTheme.White)
	colorPurple = lipgloss.Color(CurrentTheme.Purple)
	colorRoomDot = lipgloss.Color(CurrentTheme.Accent2)
	colorHistory = lipgloss.Color(CurrentTheme.History)
	colorOrange = lipgloss.Color(CurrentTheme.Orange)
	colorNotif = lipgloss.Color(CurrentTheme.Notif)

	// Update styles
	styleHeader = styleHeader.Foreground(colorAccent)
	styleBorder = styleBorder.BorderForeground(colorBorder)
	styleBorderActive = styleBorderActive.BorderForeground(colorAccent)
	styleBorderChat = styleBorderChat.BorderForeground(colorOrange)
	styleMuted = styleMuted.Foreground(colorMuted)
	styleAccent = styleAccent.Foreground(colorAccent)
	styleDanger = styleDanger.Foreground(colorDanger)
	styleOK = styleOK.Foreground(colorAccent2)
	styleWhite = styleWhite.Foreground(colorWhite)
	styleOrange = styleOrange.Foreground(colorOrange)
	styleNotif = styleNotif.Foreground(colorNotif)
	styleNotifDim = styleNotifDim.Foreground(colorNotif)
	styleSelfMsg = styleSelfMsg.Foreground(colorSelf)
	styleOtherMsg = styleOtherMsg.Foreground(colorOther)
	styleHistSelf = styleHistSelf.Foreground(colorHistory)
	styleHistOther = styleHistOther.Foreground(colorHistory)
	styleSystemMsg = styleSystemMsg.Foreground(colorSystem)
	styleTimestamp = styleTimestamp.Foreground(colorMuted)
	styleHistTs = styleHistTs.Foreground(colorHistory)
	stylePurple = stylePurple.Foreground(colorPurple)
	styleLogo = styleLogo.Foreground(colorAccent)
}

func init() {
	// Default GitHub Dark theme
	CurrentTheme = Theme{
		Name:    "GitHub Dark",
		Bg:      "#0d1117",
		Surface: "#161b22",
		Border:  "#30363d",
		Accent:  "#58a6ff",
		Accent2: "#3fb950",
		Muted:   "#8b949e",
		Danger:  "#f85149",
		Self:    "#79c0ff",
		Other:   "#56d364",
		System:  "#6e7681",
		White:   "#e6edf3",
		Purple:  "#bc8cff",
		History: "#484f58",
		Orange:  "#f0883e",
		Notif:   "#e3b341",
	}
}
