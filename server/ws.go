package server

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins (later restrict)
	},
}

func (s *Server) HandleWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.logger.Error("WS upgrade failed", "error", err)
			return
		}

		s.clients[conn] = true
		s.logger.Info("New WS client connected")

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				delete(s.clients, conn)
				conn.Close()
				break
			}

			s.broadcast(msg)
		}
	}
}

func (s *Server) broadcast(message []byte) {
	for client := range s.clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			client.Close()
			delete(s.clients, client)
		}
	}
}
