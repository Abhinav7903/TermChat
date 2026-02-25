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
	// Mode: "server" (default) or "client" (TUI)
	mode := flag.String("mode", "server", "run mode: server | client")
	host := flag.String("host", "localhost", "server host (client mode only)")
	port := flag.String("port", "9000", "TCP port (client mode only)")

	// Server flags (existing)
	envType := flag.String("env", "dev", "set the env type to dev or prod or staging")
	flag.Parse()

	switch *mode {
	case "client":
		if err := client.RunClient(*host, *port); err != nil {
			log.Fatalf("client error: %v", err)
		}
	default:
		slog.Info("Running in", "env", *envType)
		server.Run(envType)
	}
}
