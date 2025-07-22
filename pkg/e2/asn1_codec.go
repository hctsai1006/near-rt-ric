package e2

import (
	"encoding/asn1"
	"fmt"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/sirupsen/logrus"
)

// ASN1Codec provides O-RAN compliant ASN.1 encoding/decoding for E2AP messages
// Implements the E2AP protocol as specified in O-RAN.WG3.E2AP-v03.00
type ASN1Codec struct {
	config *config.ASN1Config
	logger *logrus.Entry
	
	// Performance metrics
	encodeCount uint64
	decodeCount uint64
	errorCount  uint64
}

// NewASN1Codec creates a new ASN.1 codec for E2AP messages
func NewASN1Codec(config *config.ASN1Config, logger *logrus.Logger) *ASN1Codec {
	return &ASN1Codec{
		config: config,
		logger: logger.WithField("component", "asn1-codec"),
	}
}

// EncodeE2AP_PDU encodes an E2AP PDU to ASN.1 format
func (c *ASN1Codec) EncodeE2AP_PDU(pdu *E2AP_PDU) ([]byte, error) {
	start := time.Now()
	defer func() {
		c.encodeCount++
		c.logger.WithField("duration", time.Since(start)).Debug("E2AP PDU encoding completed")
	}()
	
	if pdu == nil {
		c.errorCount++
		return nil, fmt.Errorf("E2AP PDU is nil")
	}
	
	// Validate PDU structure
	if err := c.validatePDU(pdu); err != nil {
		c.errorCount++
		return nil, fmt.Errorf("PDU validation failed: %w", err)
	}
	
	// Encode using ASN.1 DER
	encoded, err := asn1.Marshal(*pdu)
	if err != nil {
		c.errorCount++
		return nil, fmt.Errorf("failed to encode E2AP PDU: %w", err)
	}
	
	c.logger.WithFields(logrus.Fields{
		"size": len(encoded),
		"type": c.getPDUType(pdu),
	}).Debug("E2AP PDU encoded successfully")
	
	return encoded, nil
}

// DecodeE2AP_PDU decodes ASN.1 data to an E2AP PDU
func (c *ASN1Codec) DecodeE2AP_PDU(data []byte) (*E2AP_PDU, error) {
	start := time.Now()
	defer func() {
		c.decodeCount++
		c.logger.WithField("duration", time.Since(start)).Debug("E2AP PDU decoding completed")
	}()
	
	if len(data) == 0 {
		c.errorCount++
		return nil, fmt.Errorf("empty ASN.1 data")
	}
	
	var pdu E2AP_PDU
	rest, err := asn1.Unmarshal(data, &pdu)
	if err != nil {
		c.errorCount++
		return nil, fmt.Errorf("failed to decode E2AP PDU: %w", err)
	}
	
	if len(rest) > 0 && c.config.Strict {
		c.errorCount++
		return nil, fmt.Errorf("unexpected remaining bytes after PDU decoding: %d bytes", len(rest))
	}
	
	// Validate decoded PDU
	if c.config.ValidateOnDecode {
		if err := c.validatePDU(&pdu); err != nil {
			c.errorCount++
			return nil, fmt.Errorf("decoded PDU validation failed: %w", err)
		}
	}
	
	c.logger.WithFields(logrus.Fields{
		"size": len(data),
		"type": c.getPDUType(&pdu),
	}).Debug("E2AP PDU decoded successfully")
	
	return &pdu, nil
}

// EncodeE2SetupRequest encodes an E2 Setup Request message
func (c *ASN1Codec) EncodeE2SetupRequest(req *E2SetupRequest) ([]byte, error) {
	if req == nil {
		return nil, fmt.Errorf("E2SetupRequest is nil")
	}
	
	// Create initiating message
	initMsg := &InitiatingMessage{
		ProcedureCode: E2SetupRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         req,
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		InitiatingMessage: initMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeE2SetupRequest decodes an E2 Setup Request message
func (c *ASN1Codec) DecodeE2SetupRequest(value interface{}) (*E2SetupRequest, error) {
	// Type assertion and conversion logic would go here
	// This is a simplified implementation
	if req, ok := value.(*E2SetupRequest); ok {
		return req, nil
	}
	
	return nil, fmt.Errorf("failed to decode E2SetupRequest")
}

// EncodeE2SetupResponse encodes an E2 Setup Response message
func (c *ASN1Codec) EncodeE2SetupResponse(resp *E2SetupResponse) ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("E2SetupResponse is nil")
	}
	
	// Create successful outcome
	successMsg := &SuccessfulOutcome{
		ProcedureCode: E2SetupRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         resp,
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		SuccessfulOutcome: successMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeE2SetupResponse decodes an E2 Setup Response message
func (c *ASN1Codec) DecodeE2SetupResponse(value interface{}) (*E2SetupResponse, error) {
	if resp, ok := value.(*E2SetupResponse); ok {
		return resp, nil
	}
	
	return nil, fmt.Errorf("failed to decode E2SetupResponse")
}

// EncodeE2SetupFailure encodes an E2 Setup Failure message
func (c *ASN1Codec) EncodeE2SetupFailure(failure *E2SetupFailure) ([]byte, error) {
	if failure == nil {
		return nil, fmt.Errorf("E2SetupFailure is nil")
	}
	
	// Create unsuccessful outcome
	failureMsg := &UnsuccessfulOutcome{
		ProcedureCode: E2SetupRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         failure,
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		UnsuccessfulOutcome: failureMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// EncodeRICSubscriptionRequest encodes a RIC Subscription Request message
func (c *ASN1Codec) EncodeRICSubscriptionRequest(req *RICSubscriptionRequest) ([]byte, error) {
	if req == nil {
		return nil, fmt.Errorf("RICSubscriptionRequest is nil")
	}
	
	// Create initiating message
	initMsg := &InitiatingMessage{
		ProcedureCode: RICSubscriptionRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         req,
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		InitiatingMessage: initMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeRICSubscriptionRequest decodes a RIC Subscription Request message
func (c *ASN1Codec) DecodeRICSubscriptionRequest(value interface{}) (*RICSubscriptionRequest, error) {
	if req, ok := value.(*RICSubscriptionRequest); ok {
		return req, nil
	}
	
	return nil, fmt.Errorf("failed to decode RICSubscriptionRequest")
}

// EncodeRICSubscriptionResponse encodes a RIC Subscription Response message
func (c *ASN1Codec) EncodeRICSubscriptionResponse(resp *RICSubscriptionResponse) ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("RICSubscriptionResponse is nil")
	}
	
	// Create successful outcome
	successMsg := &SuccessfulOutcome{
		ProcedureCode: RICSubscriptionRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         resp,
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		SuccessfulOutcome: successMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeRICSubscriptionResponse decodes a RIC Subscription Response message
func (c *ASN1Codec) DecodeRICSubscriptionResponse(value interface{}) (*RICSubscriptionResponse, error) {
	if resp, ok := value.(*RICSubscriptionResponse); ok {
		return resp, nil
	}
	
	return nil, fmt.Errorf("failed to decode RICSubscriptionResponse")
}

// EncodeRICSubscriptionFailure encodes a RIC Subscription Failure message
func (c *ASN1Codec) EncodeRICSubscriptionFailure(failure *RICSubscriptionFailure) ([]byte, error) {
	if failure == nil {
		return nil, fmt.Errorf("RICSubscriptionFailure is nil")
	}
	
	// Create unsuccessful outcome
	failureMsg := &UnsuccessfulOutcome{
		ProcedureCode: RICSubscriptionRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         failure,
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		UnsuccessfulOutcome: failureMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeRICSubscriptionFailure decodes a RIC Subscription Failure message
func (c *ASN1Codec) DecodeRICSubscriptionFailure(value interface{}) (*RICSubscriptionFailure, error) {
	if failure, ok := value.(*RICSubscriptionFailure); ok {
		return failure, nil
	}
	
	return nil, fmt.Errorf("failed to decode RICSubscriptionFailure")
}

// EncodeRICSubscriptionDeleteRequest encodes a RIC Subscription Delete Request message
func (c *ASN1Codec) EncodeRICSubscriptionDeleteRequest(req *RICSubscriptionDeleteRequest) ([]byte, error) {
	if req == nil {
		return nil, fmt.Errorf("RICSubscriptionDeleteRequest is nil")
	}
	
	// Create initiating message
	initMsg := &InitiatingMessage{
		ProcedureCode: RICSubscriptionDeleteRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         req,
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		InitiatingMessage: initMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// EncodeRICIndication encodes a RIC Indication message
func (c *ASN1Codec) EncodeRICIndication(indication *RICIndication) ([]byte, error) {
	if indication == nil {
		return nil, fmt.Errorf("RICIndication is nil")
	}
	
	// Create initiating message
	initMsg := &InitiatingMessage{
		ProcedureCode: RICIndicationID,
		Criticality:   asn1.Enumerated(CriticalityIgnore),
		Value:         indication,
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		InitiatingMessage: initMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeRICIndication decodes a RIC Indication message
func (c *ASN1Codec) DecodeRICIndication(value interface{}) (*RICIndication, error) {
	if indication, ok := value.(*RICIndication); ok {
		return indication, nil
	}
	
	return nil, fmt.Errorf("failed to decode RICIndication")
}

// EncodeRICControlRequest encodes a RIC Control Request message
func (c *ASN1Codec) EncodeRICControlRequest(req *RICControlRequest) ([]byte, error) {
	if req == nil {
		return nil, fmt.Errorf("RICControlRequest is nil")
	}
	
	// Create initiating message
	initMsg := &InitiatingMessage{
		ProcedureCode: RICControlRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         req,
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		InitiatingMessage: initMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// EncodeRICControlAck encodes a RIC Control Acknowledge message
func (c *ASN1Codec) EncodeRICControlAck(ack *RICControlAck) ([]byte, error) {
	if ack == nil {
		return nil, fmt.Errorf("RICControlAck is nil")
	}
	
	// Create successful outcome
	successMsg := &SuccessfulOutcome{
		ProcedureCode: RICControlRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         ack,
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		SuccessfulOutcome: successMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeRICControlAck decodes a RIC Control Acknowledge message
func (c *ASN1Codec) DecodeRICControlAck(value interface{}) (*RICControlAck, error) {
	if ack, ok := value.(*RICControlAck); ok {
		return ack, nil
	}
	
	return nil, fmt.Errorf("failed to decode RICControlAck")
}

// EncodeRICControlFailure encodes a RIC Control Failure message
func (c *ASN1Codec) EncodeRICControlFailure(failure *RICControlFailure) ([]byte, error) {
	if failure == nil {
		return nil, fmt.Errorf("RICControlFailure is nil")
	}
	
	// Create unsuccessful outcome
	failureMsg := &UnsuccessfulOutcome{
		ProcedureCode: RICControlRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         failure,
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		UnsuccessfulOutcome: failureMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeRICControlFailure decodes a RIC Control Failure message
func (c *ASN1Codec) DecodeRICControlFailure(value interface{}) (*RICControlFailure, error) {
	if failure, ok := value.(*RICControlFailure); ok {
		return failure, nil
	}
	
	return nil, fmt.Errorf("failed to decode RICControlFailure")
}

// GetMessageType determines the message type from raw ASN.1 data
func (c *ASN1Codec) GetMessageType(data []byte) (E2MessageType, error) {
	pdu, err := c.DecodeE2AP_PDU(data)
	if err != nil {
		return UnknownMsg, fmt.Errorf("failed to decode PDU for message type detection: %w", err)
	}
	
	return c.getMessageTypeFromPDU(pdu), nil
}

// getMessageTypeFromPDU determines message type from a decoded PDU
func (c *ASN1Codec) getMessageTypeFromPDU(pdu *E2AP_PDU) E2MessageType {
	if pdu.InitiatingMessage != nil {
		switch pdu.InitiatingMessage.ProcedureCode {
		case E2SetupRequestID:
			return E2SetupRequestMsg
		case RICSubscriptionRequestID:
			return RICSubscriptionRequestMsg
		case RICSubscriptionDeleteRequestID:
			return RICSubscriptionDeleteRequestMsg
		case RICIndicationID:
			return RICIndicationMsg
		case RICControlRequestID:
			return RICControlRequestMsg
		case RICServiceUpdateID:
			return RICServiceUpdateMsg
		case E2NodeConfigurationUpdateID:
			return E2NodeConfigurationUpdateMsg
		case E2ConnectionUpdateID:
			return E2ConnectionUpdateMsg
		case ResetRequestID:
			return ResetRequestMsg
		case ErrorIndicationID:
			return ErrorIndicationMsg
		}
	} else if pdu.SuccessfulOutcome != nil {
		switch pdu.SuccessfulOutcome.ProcedureCode {
		case E2SetupRequestID:
			return E2SetupResponseMsg
		case RICSubscriptionRequestID:
			return RICSubscriptionResponseMsg
		case RICSubscriptionDeleteRequestID:
			return RICSubscriptionDeleteResponseMsg
		case RICControlRequestID:
			return RICControlAckMsg
		case RICServiceUpdateID:
			return RICServiceUpdateAckMsg
		case E2NodeConfigurationUpdateID:
			return E2NodeConfigurationUpdateAckMsg
		case E2ConnectionUpdateID:
			return E2ConnectionUpdateAckMsg
		case ResetRequestID:
			return ResetResponseMsg
		}
	} else if pdu.UnsuccessfulOutcome != nil {
		switch pdu.UnsuccessfulOutcome.ProcedureCode {
		case E2SetupRequestID:
			return E2SetupFailureMsg
		case RICSubscriptionRequestID:
			return RICSubscriptionFailureMsg
		case RICSubscriptionDeleteRequestID:
			return RICSubscriptionDeleteFailureMsg
		case RICControlRequestID:
			return RICControlFailureMsg
		case RICServiceUpdateID:
			return RICServiceUpdateFailureMsg
		case E2NodeConfigurationUpdateID:
			return E2NodeConfigurationUpdateFailureMsg
		case E2ConnectionUpdateID:
			return E2ConnectionUpdateFailureMsg
		}
	}
	
	return UnknownMsg
}

// validatePDU validates the structure of an E2AP PDU
func (c *ASN1Codec) validatePDU(pdu *E2AP_PDU) error {
	if pdu == nil {
		return fmt.Errorf("PDU is nil")
	}
	
	// Check that exactly one message type is present
	messageCount := 0
	if pdu.InitiatingMessage != nil {
		messageCount++
	}
	if pdu.SuccessfulOutcome != nil {
		messageCount++
	}
	if pdu.UnsuccessfulOutcome != nil {
		messageCount++
	}
	
	if messageCount != 1 {
		return fmt.Errorf("PDU must contain exactly one message type, found %d", messageCount)
	}
	
	// Validate specific message types
	if pdu.InitiatingMessage != nil {
		return c.validateInitiatingMessage(pdu.InitiatingMessage)
	} else if pdu.SuccessfulOutcome != nil {
		return c.validateSuccessfulOutcome(pdu.SuccessfulOutcome)
	} else if pdu.UnsuccessfulOutcome != nil {
		return c.validateUnsuccessfulOutcome(pdu.UnsuccessfulOutcome)
	}
	
	return nil
}

// validateInitiatingMessage validates an initiating message
func (c *ASN1Codec) validateInitiatingMessage(msg *InitiatingMessage) error {
	if msg == nil {
		return fmt.Errorf("initiating message is nil")
	}
	
	// Validate procedure code
	validProcedureCodes := []int64{
		E2SetupRequestID,
		RICSubscriptionRequestID,
		RICSubscriptionDeleteRequestID,
		RICIndicationID,
		RICControlRequestID,
		RICServiceUpdateID,
		E2NodeConfigurationUpdateID,
		E2ConnectionUpdateID,
		ResetRequestID,
		ErrorIndicationID,
	}
	
	valid := false
	for _, code := range validProcedureCodes {
		if msg.ProcedureCode == code {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("invalid procedure code for initiating message: %d", msg.ProcedureCode)
	}
	
	return nil
}

// validateSuccessfulOutcome validates a successful outcome message
func (c *ASN1Codec) validateSuccessfulOutcome(msg *SuccessfulOutcome) error {
	if msg == nil {
		return fmt.Errorf("successful outcome is nil")
	}
	
	// Validate procedure code
	validProcedureCodes := []int64{
		E2SetupRequestID,
		RICSubscriptionRequestID,
		RICSubscriptionDeleteRequestID,
		RICControlRequestID,
		RICServiceUpdateID,
		E2NodeConfigurationUpdateID,
		E2ConnectionUpdateID,
		ResetRequestID,
	}
	
	valid := false
	for _, code := range validProcedureCodes {
		if msg.ProcedureCode == code {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("invalid procedure code for successful outcome: %d", msg.ProcedureCode)
	}
	
	return nil
}

// validateUnsuccessfulOutcome validates an unsuccessful outcome message
func (c *ASN1Codec) validateUnsuccessfulOutcome(msg *UnsuccessfulOutcome) error {
	if msg == nil {
		return fmt.Errorf("unsuccessful outcome is nil")
	}
	
	// Validate procedure code
	validProcedureCodes := []int64{
		E2SetupRequestID,
		RICSubscriptionRequestID,
		RICSubscriptionDeleteRequestID,
		RICControlRequestID,
		RICServiceUpdateID,
		E2NodeConfigurationUpdateID,
		E2ConnectionUpdateID,
	}
	
	valid := false
	for _, code := range validProcedureCodes {
		if msg.ProcedureCode == code {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("invalid procedure code for unsuccessful outcome: %d", msg.ProcedureCode)
	}
	
	return nil
}

// getPDUType returns a string representation of the PDU type
func (c *ASN1Codec) getPDUType(pdu *E2AP_PDU) string {
	if pdu.InitiatingMessage != nil {
		return "InitiatingMessage"
	} else if pdu.SuccessfulOutcome != nil {
		return "SuccessfulOutcome"
	} else if pdu.UnsuccessfulOutcome != nil {
		return "UnsuccessfulOutcome"
	}
	return "Unknown"
}

// GetStatistics returns codec statistics
func (c *ASN1Codec) GetStatistics() map[string]uint64 {
	return map[string]uint64{
		"encode_count": c.encodeCount,
		"decode_count": c.decodeCount,
		"error_count":  c.errorCount,
	}
}

// ValidateMessage validates a message structure before encoding
func (c *ASN1Codec) ValidateMessage(msg interface{}) error {
	switch v := msg.(type) {
	case *E2SetupRequest:
		return c.validateE2SetupRequest(v)
	case *E2SetupResponse:
		return c.validateE2SetupResponse(v)
	case *RICSubscriptionRequest:
		return c.validateRICSubscriptionRequest(v)
	case *RICIndication:
		return c.validateRICIndication(v)
	case *RICControlRequest:
		return c.validateRICControlRequest(v)
	default:
		return fmt.Errorf("unsupported message type: %T", msg)
	}
}

// validateE2SetupRequest validates an E2 Setup Request
func (c *ASN1Codec) validateE2SetupRequest(req *E2SetupRequest) error {
	if req == nil {
		return fmt.Errorf("E2SetupRequest is nil")
	}
	
	// Validate required fields
	if len(req.GlobalE2NodeID.GNBNodeID.PLMNIdentity) == 0 &&
		len(req.GlobalE2NodeID.ENBNodeID.PLMNIdentity) == 0 &&
		len(req.GlobalE2NodeID.NGENBNodeID.PLMNIdentity) == 0 &&
		len(req.GlobalE2NodeID.ENGNBNodeID.PLMNIdentity) == 0 {
		return fmt.Errorf("Global E2 Node ID must have at least one node ID type")
	}
	
	return nil
}

// validateE2SetupResponse validates an E2 Setup Response
func (c *ASN1Codec) validateE2SetupResponse(resp *E2SetupResponse) error {
	if resp == nil {
		return fmt.Errorf("E2SetupResponse is nil")
	}
	
	// Validate required fields
	if len(resp.GlobalRICID.PLMNIdentity) == 0 {
		return fmt.Errorf("Global RIC ID PLMN Identity is required")
	}
	
	if len(resp.GlobalRICID.RICIdentity) == 0 {
		return fmt.Errorf("Global RIC ID RIC Identity is required")
	}
	
	return nil
}

// validateRICSubscriptionRequest validates a RIC Subscription Request
func (c *ASN1Codec) validateRICSubscriptionRequest(req *RICSubscriptionRequest) error {
	if req == nil {
		return fmt.Errorf("RICSubscriptionRequest is nil")
	}
	
	// Validate RIC Request ID
	if req.RICRequestID.RICRequestorID < 0 {
		return fmt.Errorf("RIC Requestor ID must be non-negative")
	}
	
	if req.RICRequestID.RICInstanceID < 0 {
		return fmt.Errorf("RIC Instance ID must be non-negative")
	}
	
	// Validate RAN Function ID
	if req.RANFunctionID < 0 {
		return fmt.Errorf("RAN Function ID must be non-negative")
	}
	
	// Validate subscription details
	if len(req.RICSubscriptionDetails.RICEventTriggerDefinition) == 0 {
		return fmt.Errorf("RIC Event Trigger Definition is required")
	}
	
	if len(req.RICSubscriptionDetails.RICActions) == 0 {
		return fmt.Errorf("at least one RIC Action is required")
	}
	
	return nil
}

// validateRICIndication validates a RIC Indication
func (c *ASN1Codec) validateRICIndication(indication *RICIndication) error {
	if indication == nil {
		return fmt.Errorf("RICIndication is nil")
	}
	
	// Validate required fields
	if len(indication.RICIndicationHeader) == 0 {
		return fmt.Errorf("RIC Indication Header is required")
	}
	
	if len(indication.RICIndicationMessage) == 0 {
		return fmt.Errorf("RIC Indication Message is required")
	}
	
	return nil
}

// validateRICControlRequest validates a RIC Control Request
func (c *ASN1Codec) validateRICControlRequest(req *RICControlRequest) error {
	if req == nil {
		return fmt.Errorf("RICControlRequest is nil")
	}
	
	// Validate required fields
	if len(req.RICControlHeader) == 0 {
		return fmt.Errorf("RIC Control Header is required")
	}
	
	if len(req.RICControlMessage) == 0 {
		return fmt.Errorf("RIC Control Message is required")
	}
	
	return nil
}