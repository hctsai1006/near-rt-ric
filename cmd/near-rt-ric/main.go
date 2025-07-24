package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/a1"
	"github.com/hctsai1006/near-rt-ric/pkg/common/logging"
	"github.com/hctsai1006/near-rt-ric/pkg/e2"
	"github.com/hctsai1006/near-rt-ric/pkg/o1"
	"github.com/hctsai1006/near-rt-ric/pkg/xapp"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	// Application metadata
	AppName    = "O-RAN Near-RT RIC"
	AppVersion = "1.0.0"
	AppDescription = "Production-grade O-RAN Near Real-Time RAN Intelligent Controller"
)

'''// RICServer represents the main Near-RT RIC server
type RICServer struct {
	config      *config.Config
	logger      *logrus.Logger

	// O-RAN interfaces
	e2Interface *e2.E2Interface
	a1Interface *a1.A1Interface
	rpcHandler *o1.RPCHandler

	// xApp framework
	xappManager *xapp.Manager

	ctx         context.Context
	cancel      context.CancelFunc
}

// NewRICServer creates a new RIC server instance
func NewRICServer(cfg *config.Config) (*RICServer, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize structured logging
	logger := logging.NewLogger(cfg.Logging)
	logger.WithFields(logrus.Fields{
		"app":     AppName,
		"version": AppVersion,
	}).Info("Initializing O-RAN Near-RT RIC server")

	server := &RICServer{
		config: cfg,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize O-RAN interfaces
	if err := server.initializeInterfaces(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize O-RAN interfaces: %w", err)
	}

	// Initialize xApp manager
	var err error
	server.xappManager, err = xapp.NewManager(cfg.XApp, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize xApp manager: %w", err)
	}

	return server, nil
}

// initializeInterfaces initializes all O-RAN interfaces (E2, A1, O1)
func (s *RICServer) initializeInterfaces() error {
	var err error

	// Initialize E2 interface
	s.e2Interface, err = e2.NewE2Interface(s.config.E2, s.logger)
	if err != nil {
		return fmt.Errorf("failed to create E2 interface: %w", err)
	}

	// Initialize A1 interface
	repo := a1.NewMemoryRepository()
	validator := a1.NewA1PolicyValidator()
	s.a1Interface = a1.NewA1Interface(s.logger, repo, validator)

	// Initialize O1 interface
	s.rpcHandler, err = o1.NewRPCHandler(s.config.O1, s.logger)
	if err != nil {
		return fmt.Errorf("failed to create O1 interface: %w", err)
	}

	return nil
}'''

// Start starts all RIC components
func (s *RICServer) Start() error {
	s.logger.Info("Starting O-RAN Near-RT RIC server")

	// Use errgroup for concurrent startup with proper error handling
	g, ctx := errgroup.WithContext(s.ctx)

	// Start E2 interface
	g.Go(func() error {
		s.logger.Info("Starting E2 interface")
		if err := s.e2Interface.Start(ctx); err != nil {
			return fmt.Errorf("E2 interface failed: %w", err)
		}
		return nil
	})

	// Start A1 interface
	g.Go(func() error {
		s.logger.Info("Starting A1 interface")
		if err := s.a1Interface.Start(ctx); err != nil {
			return fmt.Errorf("A1 interface failed: %w", err)
		}
		return nil
	})

	// Start O1 interface
	g.Go(func() error {
		s.logger.Info("Starting O1 interface")
		if err := s.rpcHandler.Start(); err != nil {
			return fmt.Errorf("O1 interface failed: %w", err)
		}
		return nil
	})

	// Start xApp manager
	g.Go(func() error {
		s.logger.Info("Starting xApp manager")
		if err := s.xappManager.Start(ctx); err != nil {
			return fmt.Errorf("xApp manager failed: %w", err)
		}
		return nil
	})

	// Wait for startup completion or error
	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to start RIC components: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"e2_port": s.config.E2.Port,
		"a1_port": s.config.A1.Port,
		"o1_port": s.config.O1.Port,
	}).Info("O-RAN Near-RT RIC server started successfully")

	return nil
}

// Stop gracefully stops all RIC components
func (s *RICServer) Stop() error {
	s.logger.Info("Stopping O-RAN Near-RT RIC server")

	// Cancel context to signal shutdown
	s.cancel()

	// Shutdown timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use errgroup for concurrent shutdown
	g, _ := errgroup.WithContext(ctx)

	// Stop components in reverse order
	g.Go(func() error {
		if err := s.xappManager.Stop(ctx); err != nil {
			s.logger.WithError(err).Error("Error stopping xApp manager")
		}
		return nil
	})

	'''	g.Go(func() error {
		s.o1Interface.Stop()
		return nil
	})'''

	g.Go(func() error {
		if err := s.a1Interface.Stop(ctx); err != nil {
			s.logger.WithError(err).Error("Error stopping A1 interface")
		}
		return nil
	})

	g.Go(func() error {
		if err := s.e2Interface.Stop(ctx); err != nil {
			s.logger.WithError(err).Error("Error stopping E2 interface")
		}
		return nil
	})

	// Wait for shutdown completion
	_ = g.Wait()

	s.logger.Info("O-RAN Near-RT RIC server stopped")
	return nil
}

// HealthCheck performs a health check of all components
func (s *RICServer) HealthCheck() error {
	checks := map[string]func() error{
		"e2": s.e2Interface.HealthCheck,
		"a1": s.a1Interface.HealthCheck,
		"o1": s.o1Interface.HealthCheck,
		"xapp": s.xappManager.HealthCheck,
	}

	for name, check := range checks {
		if err := check(); err != nil {
			return fmt.Errorf("health check failed for %s: %w", name, err)
		}
	}

	return nil
}

func main() {
	// Command line flags
	var (
		configFile   = flag.String("config", "/etc/near-rt-ric/config.yaml", "Path to configuration file")
		healthCheck  = flag.Bool("health-check", false, "Perform health check and exit")
		version      = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	// Handle version flag
	if *version {
		fmt.Printf("%s version %s\n", AppName, AppVersion)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Create RIC server
	server, err := NewRICServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create RIC server: %v\n", err)
		os.Exit(1)
	}

	// Handle health check flag
	if *healthCheck {
		if err := server.HealthCheck(); err != nil {
			fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Health check passed")
		os.Exit(0)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server
	if err := server.Start(); err != nil {
		server.logger.WithError(err).Fatal("Failed to start RIC server")
	}

	// Wait for shutdown signal
	sig := <-sigChan
	server.logger.WithField("signal", sig.String()).Info("Received shutdown signal")

	// Graceful shutdown
	if err := server.Stop(); err != nil {
		server.logger.WithError(err).Error("Error during shutdown")
		os.Exit(1)
	}

	server.logger.Info("Shutdown completed successfully")
}