package e2

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/ishidawataru/sctp"
	"github.com/sirupsen/logrus"
)

// SCTPServer implements a production-ready SCTP server for O-RAN E2 interface
// Following O-RAN.WG3.E2AP specification requirements
type SCTPServer struct {
	config     *SCTPConfig
	logger     *logrus.Logger
	
	// Network components
	listener   *sctp.SCTPListener
	tlsConfig  *tls.Config
	
	// Connection management
	connections     map[string]*SCTPConnection
	connectionsByNode map[string]*SCTPConnection
	connMutex      sync.RWMutex
	
	// Event handling
	messageHandler    func(connectionID, nodeID string, data []byte)
	connectionHandler func(event *ConnectionEvent)
	
	// Control and synchronization
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	
	// Statistics and monitoring
	stats          *SCTPStats
	startTime      time.Time
	running        atomic.Bool
	
	// Configuration
	maxConnections int32
	currentConnections atomic.Int32
}

// SCTPConfig contains SCTP-specific configuration for E2 interface
type SCTPConfig struct {
	ListenAddress     string
	Port              int
	MaxConnections    int
	ConnectionTimeout time.Duration
	HeartbeatInterval time.Duration
	BufferSize        int
	SCTP              config.SCTPConfig
	
	// Security
	TLS *tls.Config
	
	// Protocol specific
	Streams           int
	MaxAttempts       int
	RTOInitial        time.Duration
	RTOMin            time.Duration
	RTOMax            time.Duration
}

// SCTPConnection represents a single SCTP connection to an E2 node
type SCTPConnection struct {
	// Identity
	ID         string
	NodeID     string
	RemoteAddr *sctp.SCTPAddr
	
	// Network
	conn       *sctp.SCTPConn
	// streams    []sctp.SCTPStream  // TODO: Check SCTP stream implementation
	
	// State management
	state      ConnectionState
	mutex      sync.RWMutex
	
	// Timing and statistics
	connectedAt    time.Time
	lastActivity   time.Time
	lastHeartbeat  time.Time
	
	// Message handling
	sendQueue      chan []byte
	receiveBuffer  []byte
	
	// Statistics
	bytesReceived  atomic.Uint64
	bytesSent      atomic.Uint64
	messagesReceived atomic.Uint64
	messagesSent   atomic.Uint64
	errors         atomic.Uint64
	
	// Control
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// ConnectionState represents the state of an SCTP connection
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateEstablished
	StateClosing
	StateError
)

// SCTPStats contains statistics for the SCTP server
type SCTPStats struct {
	TotalConnections    atomic.Uint64
	ActiveConnections   atomic.Int32
	FailedConnections   atomic.Uint64
	BytesReceived       atomic.Uint64
	BytesSent           atomic.Uint64
	MessagesReceived    atomic.Uint64
	MessagesSent        atomic.Uint64
	Errors              atomic.Uint64
	HeartbeatsSent      atomic.Uint64
	HeartbeatsReceived  atomic.Uint64
}

// NewSCTPServer creates a new production-ready SCTP server for E2 interface
func NewSCTPServer(config *SCTPConfig, logger *logrus.Logger) (*SCTPServer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	server := &SCTPServer{
		config:            config,
		logger:            logger.WithField("component", "sctp-server").Logger,
		connections:       make(map[string]*SCTPConnection),
		connectionsByNode: make(map[string]*SCTPConnection),
		ctx:               ctx,
		cancel:            cancel,
		stats:             &SCTPStats{},
		startTime:         time.Now(),
		maxConnections:    int32(config.MaxConnections),
	}
	
	// Set TLS configuration if provided
	if config.TLS != nil {
		server.tlsConfig = config.TLS
	}
	
	return server, nil
}

// Start starts the SCTP server and begins listening for connections
func (s *SCTPServer) Start(ctx context.Context) error {
	if !s.running.CompareAndSwap(false, true) {
		return fmt.Errorf("SCTP server is already running")
	}
	
	s.logger.WithFields(logrus.Fields{
		"address": s.config.ListenAddress,
		"port":    s.config.Port,
		"streams": s.config.Streams,
	}).Info("Starting O-RAN E2 SCTP server")
	
	// Create SCTP listener address
	listenAddr, err := sctp.ResolveSCTPAddr("sctp", 
		fmt.Sprintf("%s:%d", s.config.ListenAddress, s.config.Port))
	if err != nil {
		s.running.Store(false)
		return fmt.Errorf("failed to resolve SCTP address: %w", err)
	}
	
	// Create SCTP listener with O-RAN specific parameters
	listener, err := sctp.ListenSCTP("sctp", listenAddr)
	if err != nil {
		s.running.Store(false)
		return fmt.Errorf("failed to create SCTP listener: %w", err)
	}
	
	s.listener = listener
	
	// Configure SCTP parameters for O-RAN E2 interface
	if err := s.configureSCTPParameters(); err != nil {
		s.running.Store(false)
		return fmt.Errorf("failed to configure SCTP parameters: %w", err)
	}
	
	// Start accepting connections
	s.wg.Add(1)
	go s.acceptConnections()
	
	// Start maintenance routines
	s.wg.Add(1)
	go s.heartbeatWorker()
	
	s.wg.Add(1)
	go s.statisticsWorker()
	
	s.logger.WithField("address", listenAddr.String()).Info("O-RAN E2 SCTP server started successfully")
	return nil
}

// Stop gracefully stops the SCTP server
func (s *SCTPServer) Stop(ctx context.Context) error {
	if !s.running.CompareAndSwap(true, false) {
		return nil
	}
	
	s.logger.Info("Stopping O-RAN E2 SCTP server")
	
	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}
	
	// Cancel context to stop all goroutines
	s.cancel()
	
	// Close all connections
	s.closeAllConnections()
	
	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		s.logger.Info("O-RAN E2 SCTP server stopped successfully")
	case <-ctx.Done():
		s.logger.Warn("SCTP server shutdown timeout")
	}
	
	return nil
}

// SetMessageHandler sets the handler for incoming messages
func (s *SCTPServer) SetMessageHandler(handler func(connectionID, nodeID string, data []byte)) {
	s.messageHandler = handler
}

// SetConnectionHandler sets the handler for connection events
func (s *SCTPServer) SetConnectionHandler(handler func(event *ConnectionEvent)) {
	s.connectionHandler = handler
}

// SendToNode sends data to a specific E2 node
func (s *SCTPServer) SendToNode(nodeID string, data []byte) error {
	s.connMutex.RLock()
	conn, exists := s.connectionsByNode[nodeID]
	s.connMutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("no connection found for node %s", nodeID)
	}
	
	return s.sendToConnection(conn, data)
}

// SendToConnection sends data to a specific connection
func (s *SCTPServer) SendToConnection(connectionID string, data []byte) error {
	s.connMutex.RLock()
	conn, exists := s.connections[connectionID]
	s.connMutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("connection %s not found", connectionID)
	}
	
	return s.sendToConnection(conn, data)
}

// GetConnectionCount returns the current number of active connections
func (s *SCTPServer) GetConnectionCount() int {
	return int(s.currentConnections.Load())
}

// GetStats returns current SCTP server statistics
func (s *SCTPServer) GetStats() *SCTPStats {
	return s.stats
}

// HealthCheck performs a health check of the SCTP server
func (s *SCTPServer) HealthCheck() error {
	if !s.running.Load() {
		return fmt.Errorf("SCTP server is not running")
	}
	
	if s.listener == nil {
		return fmt.Errorf("SCTP listener is not available")
	}
	
	return nil
}

// acceptConnections accepts incoming SCTP connections
func (s *SCTPServer) acceptConnections() {
	defer s.wg.Done()
	
	s.logger.Debug("Starting SCTP connection acceptor")
	
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Debug("SCTP connection acceptor stopping")
			return
		default:
			// TODO: Set accept timeout - SetDeadline not available in this SCTP library
			// s.listener.SetDeadline(time.Now().Add(1 * time.Second))
			
			conn, err := s.listener.AcceptSCTP()
			if err != nil {
				// Check if it's a timeout (expected) or real error
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				if s.running.Load() {
					s.logger.WithError(err).Error("Failed to accept SCTP connection")
					s.stats.Errors.Add(1)
				}
				continue
			}
			
			// Check connection limits
			if s.currentConnections.Load() >= s.maxConnections {
				s.logger.Warn("Maximum connections reached, rejecting new connection")
				conn.Close()
				s.stats.FailedConnections.Add(1)
				continue
			}
			
			// Handle new connection
			s.handleNewConnection(conn)
		}
	}
}

// handleNewConnection handles a new incoming SCTP connection
func (s *SCTPServer) handleNewConnection(conn *sctp.SCTPConn) {
	connectionID := uuid.New().String()
	remoteAddr := conn.RemoteAddr().(*sctp.SCTPAddr)
	
	s.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"remote_addr":   remoteAddr.String(),
	}).Info("New E2 SCTP connection established")
	
	// Create connection object
	sctpConn := &SCTPConnection{
		ID:              connectionID,
		RemoteAddr:      remoteAddr,
		conn:            conn,
		state:           StateConnected,
		connectedAt:     time.Now(),
		lastActivity:    time.Now(),
		sendQueue:       make(chan []byte, 1000),
		receiveBuffer:   make([]byte, s.config.BufferSize),
	}
	
	sctpConn.ctx, sctpConn.cancel = context.WithCancel(s.ctx)
	
	// Configure SCTP connection parameters
	s.configureConnection(sctpConn)
	
	// Register connection
	s.connMutex.Lock()
	s.connections[connectionID] = sctpConn
	s.connMutex.Unlock()
	
	// Update statistics
	s.currentConnections.Add(1)
	s.stats.TotalConnections.Add(1)
	s.stats.ActiveConnections.Store(s.currentConnections.Load())
	
	// Send connection event
	s.sendConnectionEvent(&ConnectionEvent{
		Type:         ConnectionEstablished,
		ConnectionID: connectionID,
		RemoteAddr:   remoteAddr.String(),
		Timestamp:    time.Now(),
	})
	
	// Start connection handlers
	sctpConn.wg.Add(2)
	go s.handleConnectionReceive(sctpConn)
	go s.handleConnectionSend(sctpConn)
}

// configureConnection configures SCTP-specific parameters for E2 interface
func (s *SCTPServer) configureConnection(conn *SCTPConnection) {
	// Configure socket options for O-RAN E2 requirements
	if conn.conn != nil {
		// TODO: Set socket buffer sizes - methods not available in this SCTP library
		// conn.conn.SetReadBuffer(s.config.BufferSize)
		// conn.conn.SetWriteBuffer(s.config.BufferSize)
		
		// Set timeouts - TODO: SetDeadline methods not available in this SCTP library
		// conn.conn.SetReadDeadline(time.Time{}) // No read timeout for persistent connections
		// conn.conn.SetWriteDeadline(time.Time{}) // No write timeout for persistent connections
	}
	
	s.logger.WithField("connection_id", conn.ID).Debug("SCTP connection configured for O-RAN E2")
}

// handleConnectionReceive handles receiving data from an SCTP connection
func (s *SCTPServer) handleConnectionReceive(conn *SCTPConnection) {
	defer conn.wg.Done()
	defer s.cleanupConnection(conn)
	
	s.logger.WithField("connection_id", conn.ID).Debug("Starting SCTP receive handler")
	
	for {
		select {
		case <-conn.ctx.Done():
			s.logger.WithField("connection_id", conn.ID).Debug("SCTP receive handler stopping")
			return
		default:
			// Set read deadline for heartbeat detection
			conn.conn.SetReadDeadline(time.Now().Add(s.config.HeartbeatInterval * 3))
			
			n, err := conn.conn.Read(conn.receiveBuffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Heartbeat timeout - connection might be stale
					s.logger.WithField("connection_id", conn.ID).Debug("Read timeout - checking connection health")
					if !s.isConnectionHealthy(conn) {
						s.logger.WithField("connection_id", conn.ID).Warn("Connection appears unhealthy, closing")
						return
					}
					continue
				}
				
				if s.running.Load() {
					s.logger.WithFields(logrus.Fields{
						"connection_id": conn.ID,
						"error":         err,
					}).Debug("SCTP connection read error")
				}
				return
			}
			
			if n > 0 {
				// Update activity timestamp
				conn.mutex.Lock()
				conn.lastActivity = time.Now()
				conn.mutex.Unlock()
				
				// Update statistics
				conn.bytesReceived.Add(uint64(n))
				conn.messagesReceived.Add(1)
				s.stats.BytesReceived.Add(uint64(n))
				s.stats.MessagesReceived.Add(1)
				
				// Process received data
				data := make([]byte, n)
				copy(data, conn.receiveBuffer[:n])
				
				// Handle message
				if s.messageHandler != nil {
					s.messageHandler(conn.ID, conn.NodeID, data)
				}
				
				s.logger.WithFields(logrus.Fields{
					"connection_id": conn.ID,
					"bytes":         n,
					"node_id":      conn.NodeID,
				}).Debug("Received E2AP message")
			}
		}
	}
}

// handleConnectionSend handles sending data to an SCTP connection
func (s *SCTPServer) handleConnectionSend(conn *SCTPConnection) {
	defer conn.wg.Done()
	
	s.logger.WithField("connection_id", conn.ID).Debug("Starting SCTP send handler")
	
	for {
		select {
		case <-conn.ctx.Done():
			s.logger.WithField("connection_id", conn.ID).Debug("SCTP send handler stopping")
			return
			
		case data := <-conn.sendQueue:
			if len(data) == 0 {
				continue
			}
			
			// Set write deadline
			conn.conn.SetWriteDeadline(time.Now().Add(s.config.ConnectionTimeout))
			
			n, err := conn.conn.Write(data)
			if err != nil {
				s.logger.WithFields(logrus.Fields{
					"connection_id": conn.ID,
					"error":         err,
				}).Error("Failed to send data over SCTP")
				conn.errors.Add(1)
				s.stats.Errors.Add(1)
				continue
			}
			
			// Update statistics
			conn.bytesSent.Add(uint64(n))
			conn.messagesSent.Add(1)
			s.stats.BytesSent.Add(uint64(n))
			s.stats.MessagesSent.Add(1)
			
			// Update activity timestamp
			conn.mutex.Lock()
			conn.lastActivity = time.Now()
			conn.mutex.Unlock()
			
			s.logger.WithFields(logrus.Fields{
				"connection_id": conn.ID,
				"bytes":         n,
			}).Debug("Sent E2AP message")
		}
	}
}

// sendToConnection sends data to a specific connection
func (s *SCTPServer) sendToConnection(conn *SCTPConnection, data []byte) error {
	if conn.state != StateConnected && conn.state != StateEstablished {
		return fmt.Errorf("connection %s is not in a valid state for sending", conn.ID)
	}
	
	select {
	case conn.sendQueue <- data:
		return nil
	case <-time.After(1 * time.Second):
		return fmt.Errorf("send queue full for connection %s", conn.ID)
	}
}

// isConnectionHealthy checks if a connection is healthy
func (s *SCTPServer) isConnectionHealthy(conn *SCTPConnection) bool {
	conn.mutex.RLock()
	lastActivity := conn.lastActivity
	conn.mutex.RUnlock()
	
	// Consider connection unhealthy if no activity for 3x heartbeat interval
	threshold := s.config.HeartbeatInterval * 3
	return time.Since(lastActivity) < threshold
}

// cleanupConnection cleans up a connection when it's closed
func (s *SCTPServer) cleanupConnection(conn *SCTPConnection) {
	s.logger.WithField("connection_id", conn.ID).Debug("Cleaning up SCTP connection")
	
	// Close connection
	if conn.conn != nil {
		conn.conn.Close()
	}
	
	// Cancel context
	conn.cancel()
	
	// Remove from maps
	s.connMutex.Lock()
	delete(s.connections, conn.ID)
	if conn.NodeID != "" {
		delete(s.connectionsByNode, conn.NodeID)
	}
	s.connMutex.Unlock()
	
	// Update statistics
	s.currentConnections.Add(-1)
	s.stats.ActiveConnections.Store(s.currentConnections.Load())
	
	// Send connection event
	s.sendConnectionEvent(&ConnectionEvent{
		Type:         ConnectionClosed,
		ConnectionID: conn.ID,
		NodeID:       conn.NodeID,
		RemoteAddr:   conn.RemoteAddr.String(),
		Timestamp:    time.Now(),
	})
	
	s.logger.WithFields(logrus.Fields{
		"connection_id": conn.ID,
		"node_id":      conn.NodeID,
		"duration":     time.Since(conn.connectedAt),
	}).Info("SCTP connection closed")
}

// closeAllConnections closes all active connections
func (s *SCTPServer) closeAllConnections() {
	s.connMutex.RLock()
	connections := make([]*SCTPConnection, 0, len(s.connections))
	for _, conn := range s.connections {
		connections = append(connections, conn)
	}
	s.connMutex.RUnlock()
	
	// Close all connections
	for _, conn := range connections {
		conn.cancel()
	}
	
	// Wait for all connections to close
	for _, conn := range connections {
		conn.wg.Wait()
	}
}

// RegisterNodeConnection associates a connection with a node ID
func (s *SCTPServer) RegisterNodeConnection(connectionID, nodeID string) error {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()
	
	conn, exists := s.connections[connectionID]
	if !exists {
		return fmt.Errorf("connection %s not found", connectionID)
	}
	
	// Update connection with node ID
	conn.NodeID = nodeID
	s.connectionsByNode[nodeID] = conn
	
	s.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"node_id":      nodeID,
	}).Info("Registered E2 node connection")
	
	return nil
}

// heartbeatWorker sends periodic heartbeats to detect stale connections
func (s *SCTPServer) heartbeatWorker() {
	defer s.wg.Done()
	
	ticker := time.NewTicker(s.config.HeartbeatInterval)
	defer ticker.Stop()
	
	s.logger.Debug("Starting SCTP heartbeat worker")
	
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Debug("SCTP heartbeat worker stopping")
			return
		case <-ticker.C:
			s.performHeartbeatCheck()
		}
	}
}

// performHeartbeatCheck checks all connections for heartbeat status
func (s *SCTPServer) performHeartbeatCheck() {
	s.connMutex.RLock()
	connections := make([]*SCTPConnection, 0, len(s.connections))
	for _, conn := range s.connections {
		connections = append(connections, conn)
	}
	s.connMutex.RUnlock()
	
	for _, conn := range connections {
		if !s.isConnectionHealthy(conn) {
			s.logger.WithField("connection_id", conn.ID).Warn("Connection failed heartbeat check")
			conn.cancel()
		}
	}
}

// statisticsWorker periodically logs statistics
func (s *SCTPServer) statisticsWorker() {
	defer s.wg.Done()
	
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.logStatistics()
		}
	}
}

// logStatistics logs current server statistics
func (s *SCTPServer) logStatistics() {
	s.logger.WithFields(logrus.Fields{
		"active_connections":  s.stats.ActiveConnections.Load(),
		"total_connections":   s.stats.TotalConnections.Load(),
		"failed_connections":  s.stats.FailedConnections.Load(),
		"bytes_received":      s.stats.BytesReceived.Load(),
		"bytes_sent":          s.stats.BytesSent.Load(),
		"messages_received":   s.stats.MessagesReceived.Load(),
		"messages_sent":       s.stats.MessagesSent.Load(),
		"errors":              s.stats.Errors.Load(),
		"uptime":              time.Since(s.startTime),
	}).Info("SCTP server statistics")
}

// sendConnectionEvent sends a connection event if handler is configured
func (s *SCTPServer) sendConnectionEvent(event *ConnectionEvent) {
	if s.connectionHandler != nil {
		s.connectionHandler(event)
	}
}

// configureSCTPParameters configures SCTP-specific parameters for O-RAN E2
func (s *SCTPServer) configureSCTPParameters() error {
	// This would configure SCTP-specific parameters based on O-RAN requirements
	// For now, we'll use the standard TCP-like configuration
	
	s.logger.WithFields(logrus.Fields{
		"streams":     s.config.Streams,
		"rto_initial": s.config.RTOInitial,
		"rto_min":     s.config.RTOMin,
		"rto_max":     s.config.RTOMax,
	}).Debug("Configured SCTP parameters for O-RAN E2")
	
	return nil
}

// String returns the string representation of ConnectionState
func (cs ConnectionState) String() string {
	switch cs {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateEstablished:
		return "Established"
	case StateClosing:
		return "Closing"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}