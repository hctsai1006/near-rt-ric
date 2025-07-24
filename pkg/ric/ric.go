package ric

import (
	"context"
	"fmt"
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

// RICServer represents the main Near-RT RIC server
type RICServer struct {
	config      *config.Config
	logger      *logrus.Logger
	e2Interface *e2.E2Interface
		a1Interface *a1.A1Interface
	o1Handler   *o1.O1Handler
	xappManager xapp.XAppManager
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewRICServer creates a new RIC server instance
func NewRICServer(cfg *config.Config) (*RICServer, error) {
	ctx, cancel := context.WithCancel(context.Background())

	logger := logging.NewLogger(cfg.Logging)
	log := logger.WithFields(logging.Fields{
		"app":     "O-RAN Near-RT RIC",
		"version": "1.0.0",
	})
	log.Info("Initializing O-RAN Near-RT RIC server")

	server := &RICServer{
		config: cfg,
		logger: log.Logger,
		ctx:    ctx,
		cancel: cancel,
	}

	if err := server.initializeInterfaces(log); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize O-RAN interfaces: %w", err)
	}

	// This is a placeholder for a real implementation of the repository and orchestrator
	repo := xapp.NewMemoryRepository()
	orch := xapp.NewDummyOrchestrator(log)
	reg := xapp.NewDummyRegistry(log)
	xappConfig := &xapp.XAppFrameworkConfig{
		ConflictDetection:   true,
		HealthCheckInterval: 30 * time.Second,
		MetricsInterval:     60 * time.Second,
		Namespace:           "default",
	}

	server.xappManager = xapp.NewXAppManager(repo, orch, reg, xappConfig, log.Logger)

	return server, nil
}

// initializeInterfaces initializes all O-RAN interfaces (E2, A1, O1)
func (s *RICServer) initializeInterfaces(logger *logrus.Entry) error {
	var err error

	s.e2Interface = e2.NewE2Interface(fmt.Sprintf("%s:%d", s.config.E2.ListenAddress, s.config.E2.ListenPort))

	repo := a1.NewMemoryRepository()
	validator := a1.NewA1PolicyValidator()
	s.a1Interface = a1.NewA1Interface(s.config.A1, s.logger, repo, validator)

	s.o1Handler, err = o1.NewO1Handler(s.config.O1, logger.Logger)
	if err != nil {
		return fmt.Errorf("failed to create O1 interface: %w", err)
	}

	return nil
}

// Start starts all RIC components
func (s *RICServer) Start() error {
	s.logger.Info("Starting O-RAN Near-RT RIC server")

	g, ctx := errgroup.WithContext(s.ctx)

	g.Go(func() error {
		s.logger.Info("Starting E2 interface")
		return s.e2Interface.Start(ctx)
	})

	g.Go(func() error {
		s.logger.Info("Starting A1 interface")
		// The A1 interface Start method needs to be implemented
		return nil
	})

	g.Go(func() error {
		s.logger.Info("Starting O1 interface")
		return s.o1Handler.Start()
	})

	g.Go(func() error {
		s.logger.Info("Starting xApp manager")
		// The xApp manager Start method needs to be implemented
		return nil
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to start RIC components: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"e2_port": s.config.E2.Port,
		"a1_port": s.config.A1.ListenPort,
		"o1_port": s.config.O1.Port,
	}).Info("O-RAN Near-RT RIC server started successfully")

	return nil
}

// Stop gracefully stops all RIC components
func (s *RICServer) Stop() error {
	s.logger.Info("Stopping O-RAN Near-RT RIC server")

	s.cancel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	g, _ := errgroup.WithContext(ctx)

	g.Go(func() error {
		s.xappManager.Cleanup()
		return nil
	})

	g.Go(func() error {
		// The A1 interface Stop method needs to be implemented
		return nil
	})

	g.Go(func() error {
		s.e2Interface.Stop()
		return nil
	})

	_ = g.Wait()

	s.logger.Info("O-RAN Near-RT RIC server stopped")
	return nil
}