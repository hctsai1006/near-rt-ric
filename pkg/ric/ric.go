package ric

import (
	"context"
	"fmt"

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
	a1Interface a1.O1Interface
	rpcHandler  *o1.RPCHandler
	xappManager *xapp.Manager
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewRICServer creates a new RIC server instance
func NewRICServer(cfg *config.Config) (*RICServer, error) {
	ctx, cancel := context.WithCancel(context.Background())

	logger := logging.NewLogger(cfg.Logging)
	logger.WithFields(logrus.Fields{
		"app":     "O-RAN Near-RT RIC",
		"version": "1.0.0",
	}).Info("Initializing O-RAN Near-RT RIC server")

	server := &RICServer{
		config: cfg,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	if err := server.initializeInterfaces(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize O-RAN interfaces: %w", err)
	}

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

	s.e2Interface, err = e2.NewE2Interface(s.config.E2, s.logger)
	if err != nil {
		return fmt.Errorf("failed to create E2 interface: %w", err)
	}

	repo := a1.NewMemoryRepository()
	validator := a1.NewA1PolicyValidator()
	s.a1Interface = a1.NewA1Interface(s.logger, repo, validator)

	s.rpcHandler, err = o1.NewRPCHandler(s.config.O1, s.logger)
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
		if err := s.e2Interface.Start(ctx); err != nil {
			return fmt.Errorf("E2 interface failed: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		s.logger.Info("Starting A1 interface")
		if err := s.a1Interface.Start(ctx); err != nil {
			return fmt.Errorf("A1 interface failed: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		s.logger.Info("Starting O1 interface")
		if err := s.rpcHandler.Start(); err != nil {
			return fmt.Errorf("O1 interface failed: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		s.logger.Info("Starting xApp manager")
		if err := s.xappManager.Start(ctx); err != nil {
			return fmt.Errorf("xApp manager failed: %w", err)
		}
		return nil
	})

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

	s.cancel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	g, _ := errgroup.WithContext(ctx)

	g.Go(func() error {
		if err := s.xappManager.Stop(ctx); err != nil {
			s.logger.WithError(err).Error("Error stopping xApp manager")
		}
		return nil
	})

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

	_ = g.Wait()

	s.logger.Info("O-RAN Near-RT RIC server stopped")
	return nil
}
