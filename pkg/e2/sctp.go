package e2

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ishidawataru/sctp"
	"github.com/sirupsen/logrus"
)

// SCTPConnection represents a single SCTP connection to an E2 node
type SCTPConnection struct {
	NodeID    string
	Conn      net.Conn // Using net.Conn interface for compatibility
	Address   string
	Port      int
	Connected bool
	CreatedAt time.Time
	LastUsed  time.Time
	mutex     sync.RWMutex
}

// SCTPManager handles SCTP connections for E2 interface according to O-RAN specifications
type SCTPManager struct {
	connections     map[string]*SCTPConnection
	mutex           sync.RWMutex
	logger          *logrus.Logger
	listener        net.Listener
	listenAddress   string
	listenPort      int
	ctx             context.Context
	cancel          context.CancelFunc
	messageHandler  func(nodeID string, data []byte) error
	connectTimeout  time.Duration
	readTimeout     time.Duration
	writeTimeout    time.Duration
	keepaliveInterval time.Duration
}

// SCTPConfig contains configuration for SCTP manager
type SCTPConfig struct {
	ListenAddress     string
	ListenPort        int
	ConnectTimeout    time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	KeepaliveInterval time.Duration
}

// NewSCTPManager creates a new O-RAN compliant SCTPManager
func NewSCTPManager(config *SCTPConfig) *SCTPManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &SCTPManager{
		connections:       make(map[string]*SCTPConnection),
		logger:            logrus.WithField("component", "sctp-manager").Logger,
		listenAddress:     config.ListenAddress,
		listenPort:        config.ListenPort,
		ctx:               ctx,
		cancel:            cancel,
		connectTimeout:    config.ConnectTimeout,
		readTimeout:       config.ReadTimeout,
		writeTimeout:      config.WriteTimeout,
		keepaliveInterval: config.KeepaliveInterval,
	}
}

// SetMessageHandler sets the handler function for incoming messages
func (m *SCTPManager) SetMessageHandler(handler func(nodeID string, data []byte) error) {
	m.messageHandler = handler
}

// Listen starts listening for incoming SCTP connections
func (m *SCTPManager) Listen() error {
	listenAddr := &sctp.SCTPAddr{
		IPAddrs: []net.IPAddr{{IP: net.ParseIP(m.listenAddress)}},
		Port:    m.listenPort,
	}
	
	m.logger.WithFields(logrus.Fields{
		"address": m.listenAddress,
		"port":    m.listenPort,
	}).Info("Starting SCTP listener for E2 interface")

	// Use proper SCTP listener for O-RAN E2 interface
	listener, err := sctp.ListenSCTP("sctp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to start SCTP listener: %w", err)
	}
	m.listener = listener

	// Start accepting connections
	go m.acceptConnections()

	m.logger.Info("SCTP listener started successfully")
	return nil
}

// Connect establishes an SCTP connection to the given E2 node
func (m *SCTPManager) Connect(nodeID, address string, port int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if connection already exists
	if conn, exists := m.connections[nodeID]; exists && conn.Connected {
		m.logger.WithField("node_id", nodeID).Warn("Connection already exists")
		return nil
	}

	m.logger.WithFields(logrus.Fields{
		"node_id": nodeID,
		"address": address,
		"port":    port,
	}).Info("Establishing SCTP connection to E2 node")

	// Create SCTP connection with timeout handling
	remoteAddr := &sctp.SCTPAddr{
		IPAddrs: []net.IPAddr{{IP: net.ParseIP(address)}},
		Port:    port,
	}

	// Create channel for connection result
	connChan := make(chan net.Conn, 1)
	errChan := make(chan error, 1)
	
	// Attempt SCTP connection in goroutine
	go func() {
		conn, err := sctp.DialSCTP("sctp", nil, remoteAddr)
		if err != nil {
			errChan <- err
			return
		}
		connChan <- conn
	}()

	// Wait for connection or timeout
	var conn net.Conn
	select {
	case conn = <-connChan:
		// Connection successful
	case err := <-errChan:
		return fmt.Errorf("failed to establish SCTP connection to %s: %w", nodeID, err)
	case <-time.After(m.connectTimeout):
		return fmt.Errorf("SCTP connection to %s timed out after %v", nodeID, m.connectTimeout)
	case <-m.ctx.Done():
		return fmt.Errorf("SCTP connection to %s cancelled", nodeID)
	}

	// Create connection object
	sctpConn := &SCTPConnection{
		NodeID:    nodeID,
		Conn:      conn,
		Address:   address,
		Port:      port,
		Connected: true,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	m.connections[nodeID] = sctpConn

	// Start connection handler
	go m.handleConnection(sctpConn)

	m.logger.WithField("node_id", nodeID).Info("SCTP connection established successfully")
	return nil
}

// SendToNode sends data to a specific E2 node
func (m *SCTPManager) SendToNode(nodeID string, data []byte) error {
	m.mutex.RLock()
	conn, exists := m.connections[nodeID]
	m.mutex.RUnlock()

	if !exists || !conn.Connected {
		return fmt.Errorf("no active connection to node %s", nodeID)
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	start := time.Now()
	
	// Set write timeout
	if m.writeTimeout > 0 {
		conn.Conn.SetWriteDeadline(time.Now().Add(m.writeTimeout))
	}

	// Send data
	bytesSent, err := conn.Conn.Write(data)
	if err != nil {
		m.logger.WithError(err).WithField("node_id", nodeID).Error("Failed to send data to E2 node")
		return fmt.Errorf("failed to send data to node %s: %w", nodeID, err)
	}

	conn.LastUsed = time.Now()

	m.logger.WithFields(logrus.Fields{
		"node_id": nodeID,
		"bytes_sent": bytesSent,
		"duration": time.Since(start),
	}).Debug("Data sent to E2 node successfully")

	return nil
}

// BroadcastToAllNodes sends data to all connected E2 nodes
func (m *SCTPManager) BroadcastToAllNodes(data []byte) []error {
	m.mutex.RLock()
	connections := make([]*SCTPConnection, 0, len(m.connections))
	for _, conn := range m.connections {
		if conn.Connected {
			connections = append(connections, conn)
		}
	}
	m.mutex.RUnlock()

	var errors []error
	for _, conn := range connections {
		if err := m.SendToNode(conn.NodeID, data); err != nil {
			errors = append(errors, err)
		}
	}

	m.logger.WithFields(logrus.Fields{
		"nodes_targeted": len(connections),
		"errors": len(errors),
		"data_size": len(data),
	}).Info("Broadcast to E2 nodes completed")

	return errors
}

// DisconnectNode closes the connection to a specific E2 node
func (m *SCTPManager) DisconnectNode(nodeID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	conn, exists := m.connections[nodeID]
	if !exists {
		return fmt.Errorf("no connection to node %s", nodeID)
	}

	// Close the connection
	conn.mutex.Lock()
	if conn.Conn != nil {
		conn.Conn.Close()
	}
	conn.Connected = false
	conn.mutex.Unlock()

	// Remove from connections map
	delete(m.connections, nodeID)

	m.logger.WithField("node_id", nodeID).Info("E2 node disconnected")
	return nil
}

// GetConnectionStatus returns the status of all connections
func (m *SCTPManager) GetConnectionStatus() map[string]bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	status := make(map[string]bool)
	for nodeID, conn := range m.connections {
		status[nodeID] = conn.Connected
	}

	return status
}

// GetConnectionStatistics returns detailed statistics about connections
func (m *SCTPManager) GetConnectionStatistics() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_connections": len(m.connections),
		"active_connections": 0,
	}

	for _, conn := range m.connections {
		if conn.Connected {
			stats["active_connections"] = stats["active_connections"].(int) + 1
		}
	}

	return stats
}

// Helper methods

func (m *SCTPManager) acceptConnections() {
	for {
		select {
		case <-m.ctx.Done():
			return
		default:
			conn, err := m.listener.Accept()
			if err != nil {
				m.logger.WithError(err).Error("Failed to accept SCTP connection")
				continue
			}

			// Handle incoming connection
			go m.handleIncomingConnection(conn)
		}
	}
}

func (m *SCTPManager) handleIncomingConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	m.logger.WithField("remote_addr", remoteAddr).Info("New incoming E2 connection")

	// For now, we'll generate a temporary node ID based on the address
	// In a real implementation, this would be determined from the E2 Setup Request
	nodeID := fmt.Sprintf("node_%s", remoteAddr)

	sctpConn := &SCTPConnection{
		NodeID:    nodeID,
		Conn:      conn,
		Address:   remoteAddr,
		Connected: true,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	m.mutex.Lock()
	m.connections[nodeID] = sctpConn
	m.mutex.Unlock()

	// Start connection handler
	m.handleConnection(sctpConn)
}

func (m *SCTPManager) handleConnection(conn *SCTPConnection) {
	m.logger.WithField("node_id", conn.NodeID).Info("Starting connection handler")

	buffer := make([]byte, 65536) // 64KB buffer for E2AP messages

	for {
		select {
		case <-m.ctx.Done():
			return
		default:
			// Set read timeout
			if m.readTimeout > 0 {
				conn.Conn.SetReadDeadline(time.Now().Add(m.readTimeout))
			}

			n, err := conn.Conn.Read(buffer)
			if err != nil {
				m.logger.WithError(err).WithField("node_id", conn.NodeID).Warn("Connection read error")
				m.DisconnectNode(conn.NodeID)
				return
			}

			if n > 0 {
				conn.mutex.Lock()
				conn.LastUsed = time.Now()
				conn.mutex.Unlock()

				// Process received message
				if m.messageHandler != nil {
					if err := m.messageHandler(conn.NodeID, buffer[:n]); err != nil {
						m.logger.WithError(err).WithField("node_id", conn.NodeID).Error("Message handler error")
					}
				}
			}
		}
	}
}

// Close closes all connections and stops the SCTP manager
func (m *SCTPManager) Close() error {
	m.logger.Info("Closing SCTP manager")

	// Cancel context to stop all goroutines
	m.cancel()

	// Close listener
	if m.listener != nil {
		m.listener.Close()
	}

	// Close all connections
	m.mutex.Lock()
	for nodeID := range m.connections {
		m.DisconnectNode(nodeID)
	}
	m.mutex.Unlock()

	m.logger.Info("SCTP manager closed successfully")
	return nil
}
