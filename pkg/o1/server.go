package o1

import (
	"fmt"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/o1/netconf"
	"github.com/hctsai1006/near-rt-ric/pkg/o1/yang"
	"github.com/sirupsen/logrus"
)

// Server represents the O1 interface server.
type Server struct {
	config *config.O1Config
	logger *logrus.Logger
	netconfServer *netconf.Server
	yangManager   *yang.Manager
}

// NewServer creates a new O1 server.
func NewServer(config *config.O1Config, logger *logrus.Logger) (*Server, error) {
	yangManager := yang.NewManager(logger)
	if err := yangManager.LoadModels(config.YANG.ModulesPath); err != nil {
		return nil, fmt.Errorf("failed to load YANG models: %w", err)
	}

	netconfServer, err := netconf.NewServer(logger, yangManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create NETCONF server: %w", err)
	}

	return &Server{
		config: config,
		logger: logger,
		netconfServer: netconfServer,
		yangManager:   yangManager,
	}, nil
}

// Start starts the O1 server.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.NETCONF.ListenAddress, s.config.NETCONF.Port)
	s.logger.Infof("Starting O1 server on %s", addr)
	return s.netconfServer.Start(addr)
}

// Stop stops the O1 server.
func (s *Server) Stop() {
	s.logger.Info("Stopping O1 server")
	s.netconfServer.Stop()
}

// HealthCheck performs a health check of the O1 interface.
func (s *Server) HealthCheck() error {
	// In a real implementation, we would check the status of the NETCONF server
	// and other components of the O1 interface.
	return nil
}