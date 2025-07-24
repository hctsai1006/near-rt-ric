package o1

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// NETCONF protocol constants
const (
	NetconfBase10          = "urn:ietf:params:netconf:base:1.0"
	NetconfBase11          = "urn:ietf:params:netconf:base:1.1"
	NetconfHelloTag        = "hello"
	NetconfRPCTag          = "rpc"
	NetconfReplyTag        = "rpc-reply"
	NetconfNotificationTag = "notification"
	NetconfEndDelimiter    = "]]>]]>"
)

// NETCONF message framing types
type FramingType int

const (
	FramingEOM     FramingType = iota // End-of-message framing (NETCONF 1.0)
	FramingChunked                    // Chunked framing (NETCONF 1.1)
)

// sendHello sends NETCONF hello message
func (nsh *NetconfSessionHandler) sendHello() error {
	hello := struct {
		XMLName      xml.Name            `xml:"hello"`
		Xmlns        string              `xml:"xmlns,attr"`
		Capabilities []NetconfCapability `xml:"capabilities>capability"`
		SessionID    uint32              `xml:"session-id"`
	}{
		Xmlns:        NetconfBase10,
		Capabilities: nsh.server.capabilities,
		SessionID:    nsh.session.SessionID,
	}

	if err := nsh.sendMessage(hello); err != nil {
		return fmt.Errorf("failed to send hello: %w", err)
	}

	nsh.logger.WithField("session_id", nsh.session.SessionID).Debug("NETCONF hello sent")
	return nil
}

// processMessages processes incoming NETCONF messages
func (nsh *NetconfSessionHandler) processMessages() {
	nsh.logger.Debug("Starting NETCONF message processing")

	// First, read and process hello message from client
	if err := nsh.receiveHello(); err != nil {
		nsh.logger.WithError(err).Error("Failed to receive client hello")
		return
	}

	// Process RPC messages
	for {
		select {
		case <-nsh.ctx.Done():
			return
		default:
			if err := nsh.processNextMessage(); err != nil {
				if err == io.EOF {
					nsh.logger.Debug("Client closed connection")
					return
				}
				nsh.logger.WithError(err).Error("Error processing message")
				return
			}
		}
	}
}

// receiveHello receives and processes client hello message
func (nsh *NetconfSessionHandler) receiveHello() error {
	var hello struct {
		XMLName      xml.Name `xml:"hello"`
		Capabilities []struct {
			URI string `xml:",chardata"`
		} `xml:"capabilities>capability"`
	}

	if err := nsh.receiveMessage(&hello); err != nil {
		return fmt.Errorf("failed to receive hello: %w", err)
	}

	// Process client capabilities
	clientCaps := make([]string, len(hello.Capabilities))
	for i, cap := range hello.Capabilities {
		clientCaps[i] = cap.URI
	}

	nsh.logger.WithFields(logrus.Fields{
		"client_capabilities": len(clientCaps),
		"session_id":          nsh.session.SessionID,
	}).Info("Client hello received")

	// Determine framing type based on capabilities
	nsh.determineFraming(clientCaps)

	return nil
}

// determineFraming determines the framing type to use
func (nsh *NetconfSessionHandler) determineFraming(clientCaps []string) {
	// Check if client supports NETCONF 1.1 (chunked framing)
	for _, cap := range clientCaps {
		if cap == NetconfBase11 {
			nsh.logger.Debug("Using chunked framing (NETCONF 1.1)")
			return // Use chunked framing
		}
	}
	nsh.logger.Debug("Using EOM framing (NETCONF 1.0)")
}

// processNextMessage processes the next incoming message
func (nsh *NetconfSessionHandler) processNextMessage() error {
	var rpc NetconfRPC
	if err := nsh.receiveMessage(&rpc); err != nil {
		return err
	}

	nsh.server.stats.TotalMessages++
	nsh.session.InRPCs++

	start := time.Now()
	reply, err := nsh.processRPC(&rpc)
	duration := time.Since(start)

	// Update statistics
	if err != nil {
		nsh.server.stats.FailedRPCs++
		nsh.session.InBadRPCs++
	} else {
		nsh.server.stats.SuccessfulRPCs++
	}

	// Update average RPC time
	if nsh.server.stats.SuccessfulRPCs > 0 {
		nsh.server.stats.AverageRPCTime = time.Duration(
			(int64(nsh.server.stats.AverageRPCTime)*nsh.server.stats.SuccessfulRPCs + int64(duration)) /
				(nsh.server.stats.SuccessfulRPCs + 1),
		)
	}

	// Send reply
	if err := nsh.sendMessage(reply); err != nil {
		return fmt.Errorf("failed to send reply: %w", err)
	}

	nsh.logger.WithFields(logrus.Fields{
		"message_id": rpc.MessageID,
		"duration":   duration,
		"success":    err == nil,
	}).Debug("RPC processed")

	return nil
}

// processRPC processes an RPC message and returns a reply
func (nsh *NetconfSessionHandler) processRPC(rpc *NetconfRPC) (*NetconfRPCReply, error) {
	reply := &NetconfRPCReply{
		MessageID: rpc.MessageID,
	}

	// Parse the operation from the inner XML
	operationXML := string(rpc.Operation.([]byte))
	operation, err := nsh.parseOperation(operationXML)
	if err != nil {
		reply.Error = &NetconfRPCError{
			Type:     "application",
			Tag:      "operation-not-supported",
			Severity: "error",
			Message:  fmt.Sprintf("Failed to parse operation: %v", err),
		}
		return reply, nil
	}

	// Dispatch to appropriate handler
	if nsh.server.messageHandler == nil {
		reply.Error = &NetconfRPCError{
			Type:     "application",
			Tag:      "operation-not-supported",
			Severity: "error",
			Message:  "No message handler configured",
		}
		return reply, nil
	}

	var data interface{}
	var handleErr error

	switch operation {
	case NetconfGet:
		filter := nsh.extractFilter(operationXML)
		data, handleErr = nsh.server.messageHandler.HandleGet(nsh.session.SessionID, filter)

	case NetconfGetConfig:
		req := nsh.parseGetConfigRequest(operationXML)
		data, handleErr = nsh.server.messageHandler.HandleGetConfig(nsh.session.SessionID, req)

	case NetconfEditConfig:
		req := nsh.parseEditConfigRequest(operationXML)
		handleErr = nsh.server.messageHandler.HandleEditConfig(nsh.session.SessionID, req)
		if handleErr == nil {
			reply.OK = &struct{}{}
		}

	case NetconfCopyConfig:
		source, target := nsh.parseCopyConfigRequest(operationXML)
		handleErr = nsh.server.messageHandler.HandleCopyConfig(nsh.session.SessionID, source, target)
		if handleErr == nil {
			reply.OK = &struct{}{}
		}

	case NetconfDeleteConfig:
		target := nsh.parseDeleteConfigRequest(operationXML)
		handleErr = nsh.server.messageHandler.HandleDeleteConfig(nsh.session.SessionID, target)
		if handleErr == nil {
			reply.OK = &struct{}{}
		}

	case NetconfLock:
		target := nsh.parseLockRequest(operationXML)
		handleErr = nsh.server.messageHandler.HandleLock(nsh.session.SessionID, target)
		if handleErr == nil {
			reply.OK = &struct{}{}
		}

	case NetconfUnlock:
		target := nsh.parseUnlockRequest(operationXML)
		handleErr = nsh.server.messageHandler.HandleUnlock(nsh.session.SessionID, target)
		if handleErr == nil {
			reply.OK = &struct{}{}
		}

	case NetconfCloseSession:
		handleErr = nsh.server.messageHandler.HandleCloseSession(nsh.session.SessionID)
		if handleErr == nil {
			reply.OK = &struct{}{}
		}

	case NetconfKillSession:
		targetSessionID := nsh.parseKillSessionRequest(operationXML)
		handleErr = nsh.server.messageHandler.HandleKillSession(nsh.session.SessionID, targetSessionID)
		if handleErr == nil {
			reply.OK = &struct{}{}
		}

	default:
		reply.Error = &NetconfRPCError{
			Type:     "application",
			Tag:      "operation-not-supported",
			Severity: "error",
			Message:  fmt.Sprintf("Unsupported operation: %s", operation),
		}
		return reply, nil
	}

	// Handle operation errors
	if handleErr != nil {
		reply.Error = &NetconfRPCError{
			Type:     "application",
			Tag:      "operation-failed",
			Severity: "error",
			Message:  handleErr.Error(),
		}
		return reply, nil
	}

	// Set data if available
	if data != nil {
		reply.Data = data
	}

	return reply, nil
}

// sendMessage sends a message to the client
func (nsh *NetconfSessionHandler) sendMessage(msg interface{}) error {
	nsh.mutex.Lock()
	defer nsh.mutex.Unlock()

	// Encode message to XML
	if err := nsh.encoder.Encode(msg); err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Send end-of-message delimiter
	if _, err := nsh.conn.Write([]byte(NetconfEndDelimiter)); err != nil {
		return fmt.Errorf("failed to write delimiter: %w", err)
	}

	// Update statistics
	nsh.server.stats.BytesSent += int64(len(NetconfEndDelimiter))

	return nil
}

// receiveMessage receives a message from the client
func (nsh *NetconfSessionHandler) receiveMessage(msg interface{}) error {
	// Read until we find the end-of-message delimiter
	var messageData []byte
	buffer := make([]byte, 1024)

	for {
		n, err := nsh.conn.Read(buffer)
		if err != nil {
			return err
		}

		messageData = append(messageData, buffer[:n]...)
		nsh.server.stats.BytesReceived += int64(n)

		// Check for end-of-message delimiter
		if strings.Contains(string(messageData), NetconfEndDelimiter) {
			// Remove the delimiter
			delimiterPos := strings.Index(string(messageData), NetconfEndDelimiter)
			messageData = messageData[:delimiterPos]
			break
		}
	}

	// Decode XML message
	if err := xml.Unmarshal(messageData, msg); err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	return nil
}

// parseOperation extracts the operation type from RPC XML
func (nsh *NetconfSessionHandler) parseOperation(operationXML string) (NetconfOperation, error) {
	operationXML = strings.TrimSpace(operationXML)

	if strings.Contains(operationXML, "<get>") {
		return NetconfGet, nil
	} else if strings.Contains(operationXML, "<get-config>") {
		return NetconfGetConfig, nil
	} else if strings.Contains(operationXML, "<edit-config>") {
		return NetconfEditConfig, nil
	} else if strings.Contains(operationXML, "<copy-config>") {
		return NetconfCopyConfig, nil
	} else if strings.Contains(operationXML, "<delete-config>") {
		return NetconfDeleteConfig, nil
	} else if strings.Contains(operationXML, "<lock>") {
		return NetconfLock, nil
	} else if strings.Contains(operationXML, "<unlock>") {
		return NetconfUnlock, nil
	} else if strings.Contains(operationXML, "<close-session>") {
		return NetconfCloseSession, nil
	} else if strings.Contains(operationXML, "<kill-session>") {
		return NetconfKillSession, nil
	}

	return "", fmt.Errorf("unknown operation in XML: %s", operationXML)
}

// extractFilter extracts filter from get operation
func (nsh *NetconfSessionHandler) extractFilter(operationXML string) string {
	// Simple filter extraction - in production, use proper XML parsing
	start := strings.Index(operationXML, "<filter")
	if start == -1 {
		return ""
	}
	end := strings.Index(operationXML[start:], "</filter>")
	if end == -1 {
		return ""
	}
	return operationXML[start : start+end+9]
}

// parseGetConfigRequest parses get-config request
func (nsh *NetconfSessionHandler) parseGetConfigRequest(operationXML string) *GetConfigRequest {
	req := &GetConfigRequest{
		Datastore: DatastoreRunning, // Default
	}

	// Extract datastore
	if strings.Contains(operationXML, "<candidate/>") {
		req.Datastore = DatastoreCandidate
	} else if strings.Contains(operationXML, "<startup/>") {
		req.Datastore = DatastoreStartup
	}

	// Extract filter
	req.Filter = nsh.extractFilter(operationXML)

	return req
}

// parseEditConfigRequest parses edit-config request
func (nsh *NetconfSessionHandler) parseEditConfigRequest(operationXML string) *EditConfigRequest {
	req := &EditConfigRequest{
		Datastore: DatastoreRunning, // Default
	}

	// Extract target datastore
	if strings.Contains(operationXML, "<candidate/>") {
		req.Datastore = DatastoreCandidate
	}

	// Extract default operation
	if strings.Contains(operationXML, "<default-operation>merge</default-operation>") {
		req.DefaultOperation = "merge"
	} else if strings.Contains(operationXML, "<default-operation>replace</default-operation>") {
		req.DefaultOperation = "replace"
	} else if strings.Contains(operationXML, "<default-operation>none</default-operation>") {
		req.DefaultOperation = "none"
	}

	// Extract config (simplified)
	start := strings.Index(operationXML, "<config>")
	if start != -1 {
		end := strings.Index(operationXML[start:], "</config>")
		if end != -1 {
			req.Config = operationXML[start+8 : start+end]
		}
	}

	return req
}

// parseCopyConfigRequest parses copy-config request
func (nsh *NetconfSessionHandler) parseCopyConfigRequest(operationXML string) (DatastoreType, DatastoreType) {
	source := DatastoreRunning
	target := DatastoreRunning

	// Simple parsing - in production, use proper XML parsing
	if strings.Contains(operationXML, "<source><candidate/>") {
		source = DatastoreCandidate
	} else if strings.Contains(operationXML, "<source><startup/>") {
		source = DatastoreStartup
	}

	if strings.Contains(operationXML, "<target><candidate/>") {
		target = DatastoreCandidate
	} else if strings.Contains(operationXML, "<target><startup/>") {
		target = DatastoreStartup
	}

	return source, target
}

// parseDeleteConfigRequest parses delete-config request
func (nsh *NetconfSessionHandler) parseDeleteConfigRequest(operationXML string) DatastoreType {
	if strings.Contains(operationXML, "<candidate/>") {
		return DatastoreCandidate
	} else if strings.Contains(operationXML, "<startup/>") {
		return DatastoreStartup
	}
	return DatastoreRunning
}

// parseLockRequest parses lock request
func (nsh *NetconfSessionHandler) parseLockRequest(operationXML string) DatastoreType {
	if strings.Contains(operationXML, "<candidate/>") {
		return DatastoreCandidate
	} else if strings.Contains(operationXML, "<startup/>") {
		return DatastoreStartup
	}
	return DatastoreRunning
}

// parseUnlockRequest parses unlock request
func (nsh *NetconfSessionHandler) parseUnlockRequest(operationXML string) DatastoreType {
	if strings.Contains(operationXML, "<candidate/>") {
		return DatastoreCandidate
	} else if strings.Contains(operationXML, "<startup/>") {
		return DatastoreStartup
	}
	return DatastoreRunning
}

// parseKillSessionRequest parses kill-session request
func (nsh *NetconfSessionHandler) parseKillSessionRequest(operationXML string) uint32 {
	// Simple parsing - extract session ID
	start := strings.Index(operationXML, "<session-id>")
	if start == -1 {
		return 0
	}
	end := strings.Index(operationXML[start:], "</session-id>")
	if end == -1 {
		return 0
	}

	sessionIDStr := operationXML[start+12 : start+end]
	var sessionID uint32
	fmt.Sscanf(sessionIDStr, "%d", &sessionID)
	return sessionID
}
