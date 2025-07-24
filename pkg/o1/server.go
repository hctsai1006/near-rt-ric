package o1

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/o1/netconf"
	"github.com/hctsai1006/near-rt-ric/pkg/o1/yang"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// Server represents the O1 interface server.
type Server struct {
	config        *config.O1Config
	logger        *logrus.Logger
	netconfServer *netconf.Server
	yangManager   *yang.Manager
}

// NewServer creates a new O1 server.
func NewServer(config *config.O1Config, logger *logrus.Logger) (*Server, error) {
	yangManager := yang.NewManager(logger)
	if err := yangManager.LoadModels(config.YANG.ModulesPath); err != nil {
		return nil, fmt.Errorf("failed to load YANG models: %w", err)
	}

	sshConfig, err := createSSHConfig(config.SSH)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH config: %w", err)
	}

	netconfServer, err := netconf.NewServer(logger, yangManager, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create NETCONF server: %w", err)
	}

	return &Server{
		config:        config,
		logger:        logger,
		netconfServer: netconfServer,
		yangManager:   yangManager,
	}, nil
}

func createSSHConfig(cfg config.SSHConfig) (*ssh.ServerConfig, error) {
	privateBytes, err := ioutil.ReadFile(cfg.HostKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load host key: %w", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse host key: %w", err)
	}

	sshConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			authorizedKeysBytes, err := ioutil.ReadFile(cfg.AuthorizedKeysPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read authorized keys: %w", err)
			}

			for len(authorizedKeysBytes) > 0 {
				pub, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
				if err != nil {
					return nil, err
				}

				if bytes.Equal(pub.Marshal(), pubKey.Marshal()) {
					return &ssh.Permissions{
						Extensions: map[string]string{
							"user": c.User(),
						},
					}, nil
				}

				authorizedKeysBytes = rest
			}

			return nil, fmt.Errorf("public key not authorized")
		},
	}
	sshConfig.AddHostKey(private)

	return sshConfig, nil
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
