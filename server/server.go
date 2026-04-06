package server

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"
	"termchat/db/postgres"
	"termchat/db/redis"
	"termchat/pkg/message"
	"termchat/pkg/users"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

type Server struct {
	router  *mux.Router
	redis   *redis.Redis
	logger  *slog.Logger
	user    users.Repository
	message message.Repository
	clients map[*websocket.Conn]bool
}

type ResponseMsg struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func Run(env *string) {
	viper.SetConfigFile("json")

	var level slog.Level
	switch *env {
	case "dev":
		viper.SetConfigName("term_chat_dev")
		level = slog.LevelDebug
	case "prod":
		viper.SetConfigName("term_chat_prod")
		level = slog.LevelInfo
	default:
		viper.SetConfigName("term_chat_staging")
		level = slog.LevelDebug
	}
	viper.AutomaticEnv()
	viper.SetEnvPrefix("TERMCHAT") // Optional: allow TERMCHAT_POSTGRES_URL
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		slog.Warn("No config file found, relying on environment variables", "error", err)

	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	postgres, err := postgres.NewPostgres()
	if err != nil {
		logger.Error("Error initializing Postgres", "error", err)
		return
	}
	redis := redis.NewRedis(env)

	server := &Server{
		redis:   redis,
		router:  mux.NewRouter(),
		logger:  logger,
		user:    postgres,
		message: postgres,
		clients: make(map[*websocket.Conn]bool),
	}

	server.RegisterRoutes()

	// Start HTTP server
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}

		logger.Info("Starting HTTP server", "port", port)

		handler := enableCORS(server)

		if err := http.ListenAndServe(":"+port, handler); err != nil {
			logger.Error("HTTP server failed", "error", err)
		}
	}()

	// Start TCP server
	StartTCPServer("9000", server)
}

func (s *Server) respond(w http.ResponseWriter, data interface{}, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	var resp *ResponseMsg
	if err == nil {
		resp = &ResponseMsg{
			Message: "success",
			Data:    data,
		}
	} else {
		resp = &ResponseMsg{
			Message: err.Error(),
			Data:    nil,
		}
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		s.logger.Error("Error in encoding the response", "error", err)
	}
}

func StartTCPServer(port string, srv *Server) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		srv.logger.Error("TCP listener failed", "error", err)
		return
	}
	srv.logger.Info("TCP server listening", "port", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			srv.logger.Error("Failed to accept TCP connection", "error", err)
			continue
		}
		go handleTelnetClient(conn, srv)
	}
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*") // dev only
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
