package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hctsai1006/near-rt-ric/pkg/o1"
	"github.com/hctsai1006/near-rt-ric/pkg/o1/config"
	"github.com/sirupsen/logrus"
)

const (
	version = "1.0.0"
	appName = "O-RAN O1 Interface"
)

var (
	listenAddr = flag.String("listen-addr", "0.0.0.0", "Listen address for O1 interface")
	listenPort = flag.Int("listen-port", 830, "Listen port for O1 interface")
	logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	showVersion = flag.Bool("version", false, "Show version information")
	yangDir    = flag.String("yang-dir", "yang", "Directory containing YANG models")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("%s version %s\n", appName, version)
		os.Exit(0)
	}

	logger := logrus.New()
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logger.WithError(err).Fatal("Invalid log level")
	}
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.JSONFormatter{})

	logger.WithFields(logrus.Fields{
		"version":     version,
		"listen_addr": *listenAddr,
		"listen_port": *listenPort,
	}).Info("Starting O-RAN O1 Interface")

	config := &o1.Config{
		Netconf: &netconf.Config{
			Host: *listenAddr,
			Port: *listenPort,
		},
		YangDir: *yangDir,
	}

	server, err := o1.NewServer(config, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create O1 server")
	}

	go func() {
		if err := server.Start(); err != nil {
			logger.WithError(err).Fatal("Failed to start O1 server")
		}
	}()

	logger.Info("O-RAN O1 Interface started successfully")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	logger.WithField("signal", sig.String()).Info("Received shutdown signal")

	server.Stop()
	logger.Info("O-RAN O1 Interface shutdown completed successfully")
}
