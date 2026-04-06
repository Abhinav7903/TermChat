package main

import (
	"flag"
	"log"
	"log/slog"
	_ "net/http/pprof"
	"termchat/client"
	"termchat/server"
)

func main() {
	// Mode: "server" (default) or "client" (TUI) or "send" (CLI)
	mode := flag.String("mode", "server", "run mode: server | client | send")
	host := flag.String("host", "localhost", "server host (client/send mode only)")
	port := flag.String("port", "9000", "TCP port (client/send mode only)")

	// Send mode flags
	email := flag.String("email", "", "user email (send mode only)")
	pass := flag.String("pass", "", "user password (send mode only)")
	to := flag.String("to", "", "recipient (@user or room name) (send mode only)")
	msg := flag.String("msg", "", "message content (send mode only, or pipe to stdin)")

	// Server flags (existing)
	envType := flag.String("env", "dev", "set the env type to dev or prod or staging")
	flag.Parse()

	switch *mode {
	case "client":
		if err := client.RunClient(*host, *port); err != nil {
			log.Fatalf("client error: %v", err)
		}
	case "send":
		if *email == "" || *pass == "" || *to == "" {
			log.Fatalf("Usage: termchat --mode send --email <email> --pass <pass> --to <recipient> [--msg <message>]")
		}
		if err := client.SendCLI(*host, *port, *email, *pass, *to, *msg); err != nil {
			log.Fatalf("send error: %v", err)
		}
	default:
		slog.Info("Running in", "env", *envType)
		server.Run(envType)
	}
}
