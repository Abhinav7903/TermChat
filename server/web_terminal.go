package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

type resizeMsg struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func (s *Server) HandleWebClient() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.logger.Error("WS upgrade failed", "error", err)
			return
		}
		defer conn.Close()

		// Spawn client mode
		cmd := exec.Command("./termchat", "-mode=client")
		ptmx, err := pty.Start(cmd)
		if err != nil {
			s.logger.Error("PTY start failed", "error", err)
			return
		}
		defer func() {
			ptmx.Close()
			cmd.Process.Kill()
		}()

		// 🔥 IMPORTANT: set initial terminal size
		pty.Setsize(ptmx, &pty.Winsize{
			Rows: 40,
			Cols: 120,
		})

		// Terminal → Browser
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := ptmx.Read(buf)
				if err != nil {
					if err != io.EOF {
						s.logger.Error("PTY read error", "error", err)
					}
					return
				}
				conn.WriteMessage(websocket.BinaryMessage, buf[:n])
			}
		}()

		// Browser → Terminal
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}

			// Handle resize messages
			if msgType == websocket.TextMessage {
				var rmsg resizeMsg
				if err := json.Unmarshal(msg, &rmsg); err == nil && rmsg.Type == "resize" {
					pty.Setsize(ptmx, &pty.Winsize{
						Rows: rmsg.Rows,
						Cols: rmsg.Cols,
					})
					continue
				}
			}

			// Normal keystrokes
			_, err = ptmx.Write(msg)
			if err != nil {
				break
			}
		}
	}
}
