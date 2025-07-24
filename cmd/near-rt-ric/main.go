package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/ric"
)

func main() {
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	server, err := ric.NewRICServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create RIC server: %v\n", err)
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start RIC server: %v\n", err)
			os.Exit(1)
		}
	}()

	sig := <-sigChan
	fmt.Printf("Received signal %s, shutting down\n", sig)

	if err := server.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
		os.Exit(1)
	}
}
