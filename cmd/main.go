package main

import (
	"flag"
	"log/slog"
	_ "net/http/pprof"
	"termchat/server"
)

func main() {
	// Setup the configuration management
	envType := flag.String("env", "dev", "set the env type to dev or prod or staging")
	flag.Parse()

	slog.Info("Running in", "env", *envType)

	server.Run(envType)
}
