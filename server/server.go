package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"termchat/db/postgres"
	"termchat/db/redis"
	"termchat/pkg/message"
	"termchat/pkg/users"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

type Server struct {
	router  *mux.Router
	redis   *redis.Redis
	logger  *slog.Logger
	user    users.Repository
	message message.Repository
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
		viper.SetConfigName("dev")
		level = slog.LevelDebug
	case "prod":
		viper.SetConfigName("prod")
		level = slog.LevelInfo
	default:
		viper.SetConfigName("staging")
		level = slog.LevelDebug
	}

	viper.AddConfigPath("$HOME/.conf")

	err := viper.ReadInConfig()
	if err != nil {
		slog.Error("Error reading config file", "error", err)
		return
	}

	// Initialize logger with correct log level
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger) // Set the logger as the default

	// Initialize dependencies
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
	}

	// Register routes (Ensure this function exists)
	server.RegisterRoutes()

	port := ":8080"
	if *env != "dev" {
		port = ":8194"
	}

	server.logger.Info("Starting server", "mode", *env, "port", port)

	// Start HTTP server
	if err := http.ListenAndServe(port, server); err != nil {
		server.logger.Error("Server failed to start", "error", err)
	}
}

func (s *Server) respond(
	w http.ResponseWriter,
	data interface{},
	status int,
	err error,
) {
	// Set content type header
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
			Data:    nil, // Ensure no conflicting message structure
		}
	}

	// Encode the response
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		s.logger.Error("Error in encoding the response", "error", err)
	}
}
