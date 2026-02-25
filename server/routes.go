package server

import "net/http"

func (s *Server) RegisterRoutes() {
	s.router.HandleFunc("/ping", s.HandlePong()).Methods(http.MethodGet)
	s.router.HandleFunc("/ws", s.HandleWS()).Methods(http.MethodGet)
	s.router.HandleFunc("/terminal", s.HandleWebClient())

}

func (s *Server) HandlePong() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.respond(
			w,
			"pong",
			http.StatusOK,
			nil,
		)

	}
}
