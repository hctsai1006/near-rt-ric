package o1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// NetconfServer implements NETCONF protocol server
type NetconfServer struct {
	config  *config.O1Config
	logger  *logrus.Entry
	metrics *monitoring.MetricsCollector

	// Server state
	listener    net.Listener
	tlsListener net.Listener
	sshConfig   *ssh.ServerConfig
	
	// Session management
	sessions      map[uint32]*NetconfSession
	sessionsMutex sync.RWMutex
	nextSessionID uint32
	
	// YANG models and capabilities
	yangModels   []YANGModel
	capabilities []NetconfCapability
	
	// Message handlers
	messageHandler NetconfMessageHandler
	
	// Control
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mutex   sync.RWMutex
	
	// Statistics
	stats NetconfStats
}

// NetconfStats holds NETCONF server statistics
type NetconfStats struct {
	ActiveSessions    int32 `json:"active_sessions"`
	TotalSessions     int64 `json:"total_sessions"`
	TotalMessages     int64 `json:"total_messages"`
	SuccessfulRPCs    int64 `json:"successful_rpcs"`
	FailedRPCs        int64 `json:"failed_rpcs"`
	BytesReceived     int64 `json:"bytes_received"`
	BytesSent         int64 `json:"bytes_sent"`
	AverageRPCTime    time.Duration `json:"average_rpc_time"`
	LastActivity      time.Time `json:"last_activity"`
	UptimeSeconds     int64 `json:"uptime_seconds"`
}

// NetconfMessageHandler defines interface for handling NETCONF messages
type NetconfMessageHandler interface {
	HandleGetConfig(sessionID uint32, req *GetConfigRequest) (interface{}, error)
	HandleEditConfig(sessionID uint32, req *EditConfigRequest) error
	HandleGet(sessionID uint32, filter string) (interface{}, error)
	HandleCopyConfig(sessionID uint32, source, target DatastoreType) error
	HandleDeleteConfig(sessionID uint32, target DatastoreType) error
	HandleLock(sessionID uint32, target DatastoreType) error
	HandleUnlock(sessionID uint32, target DatastoreType) error
	HandleCloseSession(sessionID uint32) error
	HandleKillSession(sessionID uint32, targetSessionID uint32) error
}

// NetconfSessionHandler handles individual NETCONF session
type NetconfSessionHandler struct {
	server    *NetconfServer
	session   *NetconfSession
	conn      net.Conn
	logger    *logrus.Entry
	ctx       context.Context
	cancel    context.CancelFunc
	decoder   *xml.Decoder
	encoder   *xml.Encoder
	buffer    []byte
	mutex     sync.Mutex
}

// NewNetconfServer creates a new NETCONF server
func NewNetconfServer(cfg *config.O1Config, baseLogger *logrus.Logger, metrics *monitoring.MetricsCollector) (*NetconfServer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	server := &NetconfServer{
		config:        cfg,
		logger:        baseLogger.WithField("component", "netconf-server"),
		metrics:       metrics,
		sessions:      make(map[uint32]*NetconfSession),
		nextSessionID: 1,
		ctx:           ctx,
		cancel:        cancel,
	}
	
	// Initialize YANG models
	server.initializeYANGModels()
	
	// Initialize capabilities
	server.initializeCapabilities()
	
	// Setup SSH server configuration
	if err := server.setupSSHConfig(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to setup SSH config: %w", err)
	}

	// Initialize datastore and message handler
	datastore := NewNetconfDatastore()
	server.messageHandler = NewMessageHandler(datastore)
	
	server.logger.Info("NETCONF server initialized")
	return server, nil
}

// initializeYANGModels sets up supported YANG models
func (ns *NetconfServer) initializeYANGModels() {
	ns.yangModels = []YANGModel{
		{
			Name:      "ietf-netconf",
			Namespace: "urn:ietf:params:xml:ns:netconf:base:1.0",
			Version:   "1.0",
			Revision:  "2011-06-01",
		},
		{
			Name:      "ietf-netconf-monitoring",
			Namespace: "urn:ietf:params:xml:ns:yang:ietf-netconf-monitoring",
			Version:   "1.0",
			Revision:  "2010-10-04",
		},
		{
			Name:      "o-ran-sc-ric-gnb-status",
			Namespace: "urn:o-ran:ric:gnb-status:1.0",
			Version:   "1.0",
			Revision:  "2022-11-14",
		},
		{
			Name:      "o-ran-sc-ric-alarms",
			Namespace: "urn:o-ran:ric:alarms:1.0",
			Version:   "1.0",
			Revision:  "2022-11-14",
		},
		{
			Name:      "o-ran-sc-ric-xapp-mgr",
			Namespace: "urn:o-ran:ric:xapp-mgr:1.0",
			Version:   "1.0",
			Revision:  "2022-11-14",
		},
	}
	
	ns.logger.WithField("models", len(ns.yangModels)).Info("YANG models initialized")
}

// initializeCapabilities sets up NETCONF capabilities
func (ns *NetconfServer) initializeCapabilities() {
	ns.capabilities = []NetconfCapability{
		{URI: "urn:ietf:params:netconf:base:1.0"},
		{URI: "urn:ietf:params:netconf:base:1.1"},
		{URI: "urn:ietf:params:netconf:capability:writable-running:1.0"},
		{URI: "urn:ietf:params:netconf:capability:candidate:1.0"},
		{URI: "urn:ietf:params:netconf:capability:confirmed-commit:1.1"},
		{URI: "urn:ietf:params:netconf:capability:rollback-on-error:1.0"},
		{URI: "urn:ietf:params:netconf:capability:validate:1.1"},
		{URI: "urn:ietf:params:netconf:capability:startup:1.0"},
		{URI: "urn:ietf:params:netconf:capability:url:1.0"},
		{URI: "urn:ietf:params:netconf:capability:xpath:1.0"},
		{URI: "urn:ietf:params:netconf:capability:notification:1.0"},
		{URI: "urn:ietf:params:netconf:capability:interleave:1.0"},
	}
	
	// Add YANG model capabilities
	for _, model := range ns.yangModels {
		cap := NetconfCapability{
			URI:      fmt.Sprintf("%s?module=%s&revision=%s", model.Namespace, model.Name, model.Revision),
			Module:   model.Name,
			Revision: model.Revision,
		}
		ns.capabilities = append(ns.capabilities, cap)
	}
	
	ns.logger.WithField("capabilities", len(ns.capabilities)).Info("NETCONF capabilities initialized")
}

// setupSSHConfig configures SSH server for NETCONF over SSH
func (ns *NetconfServer) setupSSHConfig() error {
	ns.sshConfig = &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// In production, implement proper authentication
			if c.User() == "netconf" && string(pass) == "netconf" {
				return nil, nil
			}
			return nil, fmt.Errorf("authentication failed")
		},
	}
	
	// Generate or load host key (in production, use proper key management)
	hostKey, err := generateHostKey()
	if err != nil {
		return fmt.Errorf("failed to generate host key: %w", err)
	}
	
	ns.sshConfig.AddHostKey(hostKey)
	return nil
}

// Start starts the NETCONF server
func (ns *NetconfServer) Start(ctx context.Context) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()
	
	if ns.running {
		return fmt.Errorf("NETCONF server is already running")
	}
	
	ns.logger.WithFields(logrus.Fields{
		"port":        ns.config.NETCONF.Port,
		"tls_port":    ns.config.NETCONF.TLSPort,
		"tls_enabled": ns.config.NETCONF.TLS.Enabled,
	}).Info("Starting NETCONF server")
	
	// Start SSH/NETCONF listener
	addr := fmt.Sprintf("%s:%d", ns.config.NETCONF.ListenAddress, ns.config.NETCONF.Port)
	var err error
	ns.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	
	// Start TLS listener if enabled
	if ns.config.NETCONF.TLS.Enabled {
		tlsAddr := fmt.Sprintf("%s:%d", ns.config.NETCONF.ListenAddress, ns.config.NETCONF.TLSPort)
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		
		// Load certificates (in production, use proper certificate management)
		if ns.config.NETCONF.TLS.CertFile != "" && ns.config.NETCONF.TLS.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(ns.config.NETCONF.TLS.CertFile, ns.config.NETCONF.TLS.KeyFile)
			if err != nil {
				ns.listener.Close()
				return fmt.Errorf("failed to load TLS certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
		
		ns.tlsListener, err = tls.Listen("tcp", tlsAddr, tlsConfig)
		if err != nil {
			ns.listener.Close()
			return fmt.Errorf("failed to listen on TLS %s: %w", tlsAddr, err)
		}
		
		// Accept TLS connections
		ns.wg.Add(1)
		go ns.acceptTLSConnections()
	}
	
	// Accept SSH connections
	ns.wg.Add(1)
	go ns.acceptSSHConnections()
	
	// Start cleanup routine
	ns.wg.Add(1)
	go ns.cleanupRoutine()
	
	ns.running = true
	ns.stats.LastActivity = time.Now()
	
	ns.logger.Info("NETCONF server started successfully")
	return nil
}

// Stop stops the NETCONF server gracefully
func (ns *NetconfServer) Stop(ctx context.Context) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()
	
	if !ns.running {
		return nil
	}
	
	ns.logger.Info("Stopping NETCONF server")
	
	// Close listeners
	if ns.listener != nil {
		ns.listener.Close()
	}
	if ns.tlsListener != nil {
		ns.tlsListener.Close()
	}
	
	// Close all active sessions
	ns.sessionsMutex.Lock()
	for _, session := range ns.sessions {
		ns.logger.WithField("session_id", session.SessionID).Debug("Closing NETCONF session")
	}
	ns.sessions = make(map[uint32]*NetconfSession)
	ns.sessionsMutex.Unlock()
	
	// Cancel context and wait for goroutines
	ns.cancel()
	
	done := make(chan struct{})
	go func() {
		ns.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		ns.logger.Info("NETCONF server stopped successfully")
	case <-ctx.Done():
		ns.logger.Warn("NETCONF server shutdown timeout")
	}
	
	ns.running = false
	return nil
}

// acceptSSHConnections accepts incoming SSH connections
func (ns *NetconfServer) acceptSSHConnections() {
	defer ns.wg.Done()
	
	for {
		conn, err := ns.listener.Accept()
		if err != nil {
			select {
			case <-ns.ctx.Done():
				return
			default:
				ns.logger.WithError(err).Error("Failed to accept connection")
				continue
			}
		}
		
		go ns.handleSSHConnection(conn)
	}
}

// acceptTLSConnections accepts incoming TLS connections
func (ns *NetconfServer) acceptTLSConnections() {
	defer ns.wg.Done()
	
	for {
		conn, err := ns.tlsListener.Accept()
		if err != nil {
			select {
			case <-ns.ctx.Done():
				return
			default:
				ns.logger.WithError(err).Error("Failed to accept TLS connection")
				continue
			}
		}
		
		go ns.handleTLSConnection(conn)
	}
}

// handleSSHConnection handles an individual SSH connection
func (ns *NetconfServer) handleSSHConnection(conn net.Conn) {
	defer conn.Close()
	
	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, ns.sshConfig)
	if err != nil {
		ns.logger.WithError(err).Error("SSH handshake failed")
		return
	}
	defer sshConn.Close()
	
	ns.logger.WithFields(logrus.Fields{
		"user":        sshConn.User(),
		"remote_addr": sshConn.RemoteAddr(),
	}).Info("SSH connection established")
	
	// Handle incoming channel requests
	go ssh.DiscardRequests(reqs)
	
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		
		channel, requests, err := newChannel.Accept()
		if err != nil {
			ns.logger.WithError(err).Error("Failed to accept channel")
			continue
		}
		
		go ns.handleNetconfSession(channel, requests, sshConn.User(), conn.RemoteAddr().String())
	}
}

// handleTLSConnection handles an individual TLS connection
func (ns *NetconfServer) handleTLSConnection(conn net.Conn) {
	defer conn.Close()
	
	ns.logger.WithField("remote_addr", conn.RemoteAddr()).Info("TLS connection established")
	
	// For TLS connections, NETCONF is directly over TLS
	go ns.handleNetconfSession(conn, nil, "tls-user", conn.RemoteAddr().String())
}

// handleNetconfSession handles a NETCONF session
func (ns *NetconfServer) handleNetconfSession(conn io.ReadWriteCloser, requests <-chan *ssh.Request, username, remoteAddr string) {
	defer func() {
		if closer, ok := conn.(io.Closer); ok {
			closer.Close()
		}
	}()
	
	// Create session
	sessionID := ns.createSession(username, remoteAddr)
	session := ns.getSession(sessionID)
	if session == nil {
		ns.logger.Error("Failed to create NETCONF session")
		return
	}
	
	defer ns.removeSession(sessionID)
	
	ns.logger.WithFields(logrus.Fields{
		"session_id":  sessionID,
		"username":    username,
		"remote_addr": remoteAddr,
	}).Info("NETCONF session started")
	
	// Create session handler
	ctx, cancel := context.WithCancel(ns.ctx)
	defer cancel()
	
	handler := &NetconfSessionHandler{
		server:  ns,
		session: session,
		conn:    conn.(net.Conn),
		logger:  ns.logger.WithField("session_id", sessionID),
		ctx:     ctx,
		cancel:  cancel,
		decoder: xml.NewDecoder(conn),
		encoder: xml.NewEncoder(conn),
		buffer:  make([]byte, 4096),
	}
	
	// Handle SSH requests if available
	if requests != nil {
		go func() {
			for req := range requests {
				switch req.Type {
				case "subsystem":
					if string(req.Payload[4:]) == "netconf" {
						req.Reply(true, nil)
					} else {
						req.Reply(false, nil)
					}
				default:
					req.Reply(false, nil)
				}
			}
		}()
	}
	
	// Send NETCONF hello
	if err := handler.sendHello(); err != nil {
		ns.logger.WithError(err).Error("Failed to send NETCONF hello")
		return
	}
	
	// Process NETCONF messages
	handler.processMessages()
	
	ns.logger.WithField("session_id", sessionID).Info("NETCONF session ended")
}

// SetMessageHandler sets the message handler
func (ns *NetconfServer) SetMessageHandler(handler NetconfMessageHandler) {
	ns.messageHandler = handler
}

// GetStats returns server statistics
func (ns *NetconfServer) GetStats() NetconfStats {
	ns.sessionsMutex.RLock()
	defer ns.sessionsMutex.RUnlock()
	
	stats := ns.stats
	stats.ActiveSessions = int32(len(ns.sessions))
	return stats
}

// IsRunning returns whether the server is running
func (ns *NetconfServer) IsRunning() bool {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()
	return ns.running
}

// Helper methods

// createSession creates a new NETCONF session
func (ns *NetconfServer) createSession(username, remoteAddr string) uint32 {
	ns.sessionsMutex.Lock()
	defer ns.sessionsMutex.Unlock()
	
	sessionID := ns.nextSessionID
	ns.nextSessionID++
	
	session := &NetconfSession{
		SessionID:  sessionID,
		Username:   username,
		SourceHost: remoteAddr,
		LoginTime:  time.Now(),
		Capabilities: ns.capabilities,
	}
	
	ns.sessions[sessionID] = session
	ns.stats.TotalSessions++
	
	return sessionID
}

// getSession retrieves a session by ID
func (ns *NetconfServer) getSession(sessionID uint32) *NetconfSession {
	ns.sessionsMutex.RLock()
	defer ns.sessionsMutex.RUnlock()
	
	return ns.sessions[sessionID]
}

// removeSession removes a session
func (ns *NetconfServer) removeSession(sessionID uint32) {
	ns.sessionsMutex.Lock()
	defer ns.sessionsMutex.Unlock()
	
	delete(ns.sessions, sessionID)
}

// cleanupRoutine performs periodic cleanup
func (ns *NetconfServer) cleanupRoutine() {
	defer ns.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ns.ctx.Done():
			return
		case <-ticker.C:
			ns.performCleanup()
		}
	}
}

// performCleanup performs cleanup tasks
func (ns *NetconfServer) performCleanup() {
	// Update statistics
	ns.stats.LastActivity = time.Now()
	
	// Log session statistics
	ns.sessionsMutex.RLock()
	activeCount := len(ns.sessions)
	ns.sessionsMutex.RUnlock()
	
	ns.logger.WithFields(logrus.Fields{
		"active_sessions": activeCount,
		"total_sessions":  ns.stats.TotalSessions,
		"total_messages":  ns.stats.TotalMessages,
	}).Debug("NETCONF server statistics")
}

func generateHostKey() (ssh.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(key)
}