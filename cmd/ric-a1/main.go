package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/a1"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
)

const (
	version = "1.0.0"
	appName = "O-RAN A1 Interface"
)

var (
	listenAddr = flag.String("listen-addr", "0.0.0.0", "Listen address for A1 interface")
	listenPort = flag.Int("listen-port", 10020, "Listen port for A1 interface")
	tlsEnabled = flag.Bool("tls-enabled", true, "Enable TLS")
	tlsCert    = flag.String("tls-cert", "/certs/tls.crt", "TLS certificate file")
	tlsKey     = flag.String("tls-key", "/certs/tls.key", "TLS private key file")
	logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	showVersion = flag.Bool("version", false, "Show version information")
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
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	logger.WithFields(logrus.Fields{
		"version":     version,
		"listen_addr": *listenAddr,
		"listen_port": *listenPort,
		"tls_enabled": *tlsEnabled,
	}).Info("Starting O-RAN A1 Interface")

	// Load configuration
	cfg, err := config.LoadA1Config()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Create metrics collector
	metrics := monitoring.NewMetricsCollector("near-rt-ric", "a1-interface")
	metrics.RegisterA1Metrics()

	// Create database connection
	db, err := db.NewPostgresDB(&cfg.Database, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Create authentication service
	authService, err := a1.NewAuthService(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create authentication service")
	}

	// Create policy manager
	policyManager := a1.NewPolicyManager(cfg, logger, metrics, db.Pool)
	if err := policyManager.Start(context.Background()); err != nil {
		logger.WithError(err).Fatal("Failed to start policy manager")
	}

	// Create ML model manager
	modelManager := a1.NewMLModelManager(cfg, logger, metrics, db.Pool)

	// Create enrichment manager
	enrichmentManager := a1.NewEnrichmentManager(cfg, logger, metrics, db.Pool)

	// Create API handlers
	apiHandlers := a1.NewAPIHandlers(policyManager, modelManager, enrichmentManager, authService, logger, metrics)

	// Create router and setup routes
	router := mux.NewRouter()
	apiHandlers.SetupRoutes(router)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", *listenAddr, *listenPort)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("address", addr).Info("A1 interface listening")
		var err error
		if *tlsEnabled {
			err = server.ListenAndServeTLS(*tlsCert, *tlsKey)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("O-RAN A1 Interface started successfully")

	// Wait for shutdown signal
	sig := <-sigChan
	logger.WithField("signal", sig.String()).Info("Received shutdown signal")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Error during server shutdown")
	}

	if err := policyManager.Stop(shutdownCtx); err != nil {
		logger.WithError(err).Error("Error during policy manager shutdown")
	}

	logger.Info("O-RAN A1 Interface shutdown completed successfully")
}
