package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hctsai1006/near-rt-ric/pkg/e2"
	"github.com/sirupsen/logrus"
)

const (
	version = "1.0.0"
	appName = "O-RAN Near-RT RIC"
)

var (
	listenAddr  = flag.String("listen-addr", "0.0.0.0", "Listen address for E2 interface")
	listenPort  = flag.Int("listen-port", 36421, "Listen port for E2 interface")
	maxNodes    = flag.Int("max-nodes", 100, "Maximum number of E2 nodes")
	logLevel    = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	configFile  = flag.String("config", "", "Configuration file path")
	showVersion = flag.Bool("version", false, "Show version information")
)

func main() {
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("%s version %s\n", appName, version)
		os.Exit(0)
	}

	// Configure logging
	logger := logrus.New()
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logger.WithError(err).Fatal("Invalid log level")
	}
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	logger.WithFields(logrus.Fields{
		"version":     version,
		"listen_addr": *listenAddr,
		"listen_port": *listenPort,
		"max_nodes":   *maxNodes,
	}).Info("Starting O-RAN Near-RT RIC")

	// Create and start E2 interface
	addr := fmt.Sprintf("%s:%d", *listenAddr, *listenPort)
	e2Interface := e2.NewE2Interface(addr)

	if err := e2Interface.Start(); err != nil {
		logger.WithError(err).Fatal("Failed to start E2 interface")
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("O-RAN Near-RT RIC started successfully")

	// Wait for shutdown signal
	sig := <-sigChan
	logger.WithField("signal", sig.String()).Info("Received shutdown signal")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	shutdownChan := make(chan struct{})
	go func() {
		e2Interface.Stop()
		close(shutdownChan)
	}()

	select {
	case <-shutdownChan:
		logger.Info("O-RAN Near-RT RIC shutdown completed successfully")
	case <-shutdownCtx.Done():
		logger.Error("Shutdown timeout exceeded")
		os.Exit(1)
	}
}
