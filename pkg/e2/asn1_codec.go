package e2

import (
	"encoding/asn1"
	"fmt"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
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
	// TODO: Re-enable and update validatePDU to handle the new E2AP_PDU structure with asn1.RawValue
	// if err := c.validatePDU(pdu); err != nil {
	// 	c.errorCount++
	// 	return nil, fmt.Errorf("PDU validation failed: %w", err)
	// }
	
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
	
	// Delegate to the DecodeE2AP_PDU in asn1.go which handles RawValue decoding
	pdu, err := DecodeE2AP_PDU(data)
	if err != nil {
		c.errorCount++
		return nil, fmt.Errorf("failed to decode E2AP PDU: %w", err)
	}
	
	// TODO: Re-enable and update validatePDU to handle the new E2AP_PDU structure with asn1.RawValue
	// if c.config.ValidateOnDecode {
	// 	if err := c.validatePDU(pdu); err != nil {
	// 		c.errorCount++
	// 		return nil, fmt.Errorf("decoded PDU validation failed: %w", err)
	// 	}
	// }
	
	c.logger.WithFields(logrus.Fields{
		"size": len(data),
		"type": c.getPDUType(pdu),
	}).Debug("E2AP PDU decoded successfully")
	
	return pdu, nil
}

// EncodeE2SetupRequest encodes an E2 Setup Request message
func (c *ASN1Codec) EncodeE2SetupRequest(req *models.E2SetupRequest) ([]byte, error) {
	if req == nil {
		return nil, fmt.Errorf("E2SetupRequest is nil")
	}
	
	reqBytes, err := asn1.Marshal(*req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal E2SetupRequest: %w", err)
	}

	// Create initiating message with RawValue
	initMsg := &models.InitiatingMessage{
		ProcedureCode: E2SetupRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         asn1.RawValue{Bytes: reqBytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true},
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		InitiatingMessage: initMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeE2SetupRequest decodes an E2 Setup Request message
func (c *ASN1Codec) DecodeE2SetupRequest(pdu *E2AP_PDU) (*models.E2SetupRequest, error) {
	if pdu == nil || pdu.InitiatingMessage == nil || pdu.InitiatingMessage.Value.Bytes == nil {
		return nil, fmt.Errorf("invalid E2AP PDU for E2SetupRequest decoding")
	}

	var req models.E2SetupRequest
	_, err := asn1.Unmarshal(pdu.InitiatingMessage.Value.Bytes, &req)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal E2SetupRequest from RawValue: %w", err)
	}
	
	return &req, nil
}

// EncodeE2SetupResponse encodes an E2 Setup Response message
func (c *ASN1Codec) EncodeE2SetupResponse(resp *models.E2SetupResponse) ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("E2SetupResponse is nil")
	}
	
	respBytes, err := asn1.Marshal(*resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal E2SetupResponse: %w", err)
	}

	// Create successful outcome with RawValue
	successMsg := &SuccessfulOutcome{
		ProcedureCode: E2SetupRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         asn1.RawValue{Bytes: respBytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true},
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		SuccessfulOutcome: successMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeE2SetupResponse decodes an E2 Setup Response message
func (c *ASN1Codec) DecodeE2SetupResponse(pdu *E2AP_PDU) (*models.E2SetupResponse, error) {
	if pdu == nil || pdu.SuccessfulOutcome == nil || pdu.SuccessfulOutcome.Value.Bytes == nil {
		return nil, fmt.Errorf("invalid E2AP PDU for E2SetupResponse decoding")
	}

	var resp models.E2SetupResponse
	_, err := asn1.Unmarshal(pdu.SuccessfulOutcome.Value.Bytes, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal E2SetupResponse from RawValue: %w", err)
	}
	
	return &resp, nil
}

// EncodeE2SetupFailure encodes an E2 Setup Failure message
func (c *ASN1Codec) EncodeE2SetupFailure(failure *models.E2SetupFailure) ([]byte, error) {
	if failure == nil {
		return nil, fmt.Errorf("E2SetupFailure is nil")
	}
	
	failureBytes, err := asn1.Marshal(*failure)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal E2SetupFailure: %w", err)
	}

	// Create unsuccessful outcome with RawValue
	failureMsg := &UnsuccessfulOutcome{
		ProcedureCode: E2SetupRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         asn1.RawValue{Bytes: failureBytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true},
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		UnsuccessfulOutcome: failureMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// EncodeRICSubscriptionRequest encodes a RIC Subscription Request message
func (c *ASN1Codec) EncodeRICSubscriptionRequest(req *models.RICSubscriptionRequest) ([]byte, error) {
	if req == nil {
		return nil, fmt.Errorf("RICSubscriptionRequest is nil")
	}
	
	reqBytes, err := asn1.Marshal(*req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RICSubscriptionRequest: %w", err)
	}

	// Create initiating message with RawValue
	initMsg := &InitiatingMessage{
		ProcedureCode: RICSubscriptionRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         asn1.RawValue{Bytes: reqBytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true},
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		InitiatingMessage: initMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeRICSubscriptionRequest decodes a RIC Subscription Request message
func (c *ASN1Codec) DecodeRICSubscriptionRequest(pdu *E2AP_PDU) (*models.RICSubscriptionRequest, error) {
	if pdu == nil || pdu.InitiatingMessage == nil || pdu.InitiatingMessage.Value.Bytes == nil {
		return nil, fmt.Errorf("invalid E2AP PDU for RICSubscriptionRequest decoding")
	}

	var req models.RICSubscriptionRequest
	_, err := asn1.Unmarshal(pdu.InitiatingMessage.Value.Bytes, &req)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal RICSubscriptionRequest from RawValue: %w", err)
	}
	
	return &req, nil
}

// EncodeRICSubscriptionResponse encodes a RIC Subscription Response message
func (c *ASN1Codec) EncodeRICSubscriptionResponse(resp *models.RICSubscriptionResponse) ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("RICSubscriptionResponse is nil")
	}
	
	respBytes, err := asn1.Marshal(*resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RICSubscriptionResponse: %w", err)
	}

	// Create successful outcome with RawValue
	successMsg := &SuccessfulOutcome{
		ProcedureCode: RICSubscriptionRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         asn1.RawValue{Bytes: respBytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true},
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		SuccessfulOutcome: successMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeRICSubscriptionResponse decodes a RIC Subscription Response message
func (c *ASN1Codec) DecodeRICSubscriptionResponse(pdu *E2AP_PDU) (*models.RICSubscriptionResponse, error) {
	if pdu == nil || pdu.SuccessfulOutcome == nil || pdu.SuccessfulOutcome.Value.Bytes == nil {
		return nil, fmt.Errorf("invalid E2AP PDU for RICSubscriptionResponse decoding")
	}

	var resp models.RICSubscriptionResponse
	_, err := asn1.Unmarshal(pdu.SuccessfulOutcome.Value.Bytes, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal RICSubscriptionResponse from RawValue: %w", err)
	}
	
	return &resp, nil
}

// EncodeRICSubscriptionFailure encodes a RIC Subscription Failure message
func (c *ASN1Codec) EncodeRICSubscriptionFailure(failure *models.RICSubscriptionFailure) ([]byte, error) {
	if failure == nil {
		return nil, fmt.Errorf("RICSubscriptionFailure is nil")
	}
	
	failureBytes, err := asn1.Marshal(*failure)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RICSubscriptionFailure: %w", err)
	}

	// Create unsuccessful outcome with RawValue
	failureMsg := &UnsuccessfulOutcome{
		ProcedureCode: RICSubscriptionRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         asn1.RawValue{Bytes: failureBytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true},
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		UnsuccessfulOutcome: failureMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeRICSubscriptionFailure decodes a RIC Subscription Failure message
func (c *ASN1Codec) DecodeRICSubscriptionFailure(pdu *E2AP_PDU) (*models.RICSubscriptionFailure, error) {
	if pdu == nil || pdu.UnsuccessfulOutcome == nil || pdu.UnsuccessfulOutcome.Value.Bytes == nil {
		return nil, fmt.Errorf("invalid E2AP PDU for RICSubscriptionFailure decoding")
	}

	var failure models.RICSubscriptionFailure
	_, err := asn1.Unmarshal(pdu.UnsuccessfulOutcome.Value.Bytes, &failure)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal RICSubscriptionFailure from RawValue: %w", err)
	}
	
	return &failure, nil
}

// EncodeRICSubscriptionDeleteRequest encodes a RIC Subscription Delete Request message
func (c *ASN1Codec) EncodeRICSubscriptionDeleteRequest(req *models.RICSubscriptionDeleteRequest) ([]byte, error) {
	if req == nil {
		return nil, fmt.Errorf("RICSubscriptionDeleteRequest is nil")
	}
	
	reqBytes, err := asn1.Marshal(*req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RICSubscriptionDeleteRequest: %w", err)
	}

	// Create initiating message with RawValue
	initMsg := &InitiatingMessage{
		ProcedureCode: RICSubscriptionDeleteRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         asn1.RawValue{Bytes: reqBytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true},
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		InitiatingMessage: initMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// EncodeRICIndication encodes a RIC Indication message
func (c *ASN1Codec) EncodeRICIndication(indication *models.RICIndication) ([]byte, error) {
	if indication == nil {
		return nil, fmt.Errorf("RICIndication is nil")
	}
	
	indicationBytes, err := asn1.Marshal(*indication)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RICIndication: %w", err)
	}

	// Create initiating message with RawValue
	initMsg := &InitiatingMessage{
		ProcedureCode: RICIndicationID,
		Criticality:   asn1.Enumerated(CriticalityIgnore),
		Value:         asn1.RawValue{Bytes: indicationBytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true},
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		InitiatingMessage: initMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeRICIndication decodes a RIC Indication message
func (c *ASN1Codec) DecodeRICIndication(pdu *E2AP_PDU) (*models.RICIndication, error) {
	if pdu == nil || pdu.InitiatingMessage == nil || pdu.InitiatingMessage.Value.Bytes == nil {
		return nil, fmt.Errorf("invalid E2AP PDU for RICIndication decoding")
	}

	var indication models.RICIndication
	_, err := asn1.Unmarshal(pdu.InitiatingMessage.Value.Bytes, &indication)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal RICIndication from RawValue: %w", err)
	}
	
	return &indication, nil
}

// EncodeRICControlRequest encodes a RIC Control Request message
func (c *ASN1Codec) EncodeRICControlRequest(req *models.RICControlRequest) ([]byte, error) {
	if req == nil {
		return nil, fmt.Errorf("RICControlRequest is nil")
	}
	
	reqBytes, err := asn1.Marshal(*req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal models.RICControlRequest: %w", err)
	}

	// Create initiating message with RawValue
	initMsg := &InitiatingMessage{
		ProcedureCode: RICControlRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         asn1.RawValue{Bytes: reqBytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true},
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		InitiatingMessage: initMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// EncodeRICControlAck encodes a RIC Control Acknowledge message
func (c *ASN1Codec) EncodeRICControlAck(ack *models.RICControlAck) ([]byte, error) {
	if ack == nil {
		return nil, fmt.Errorf("RICControlAck is nil")
	}
	
	ackBytes, err := asn1.Marshal(*ack)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RICControlAck: %w", err)
	}

	// Create successful outcome with RawValue
	successMsg := &SuccessfulOutcome{
		ProcedureCode: RICControlRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         asn1.RawValue{Bytes: ackBytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true},
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		SuccessfulOutcome: successMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeRICControlAck decodes a RIC Control Acknowledge message
func (c *ASN1Codec) DecodeRICControlAck(pdu *E2AP_PDU) (*models.RICControlAck, error) {
	if pdu == nil || pdu.SuccessfulOutcome == nil || pdu.SuccessfulOutcome.Value.Bytes == nil {
		return nil, fmt.Errorf("invalid E2AP PDU for RICControlAck decoding")
	}

	var ack models.RICControlAck
	_, err := asn1.Unmarshal(pdu.SuccessfulOutcome.Value.Bytes, &ack)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal RICControlAck from RawValue: %w", err)
	}
	
	return &ack, nil
}

// EncodeRICControlFailure encodes a RIC Control Failure message
func (c *ASN1Codec) EncodeRICControlFailure(failure *models.RICControlFailure) ([]byte, error) {
	if failure == nil {
		return nil, fmt.Errorf("RICControlFailure is nil")
	}
	
	failureBytes, err := asn1.Marshal(*failure)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RICControlFailure: %w", err)
	}

	// Create unsuccessful outcome with RawValue
	failureMsg := &UnsuccessfulOutcome{
		ProcedureCode: RICControlRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         asn1.RawValue{Bytes: failureBytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true},
	}
	
	// Create PDU
	pdu := &E2AP_PDU{
		UnsuccessfulOutcome: failureMsg,
	}
	
	return c.EncodeE2AP_PDU(pdu)
}

// DecodeRICControlFailure decodes a RIC Control Failure message
func (c *ASN1Codec) DecodeRICControlFailure(pdu *E2AP_PDU) (*models.RICControlFailure, error) {
	if pdu == nil || pdu.UnsuccessfulOutcome == nil || pdu.UnsuccessfulOutcome.Value.Bytes == nil {
		return nil, fmt.Errorf("invalid E2AP PDU for RICControlFailure decoding")
	}

	var failure models.RICControlFailure
	_, err := asn1.Unmarshal(pdu.UnsuccessfulOutcome.Value.Bytes, &failure)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal models.RICControlFailure from RawValue: %w", err)
	}
	
	return &failure, nil
}

// GetMessageType determines the message type from raw ASN.1 data
func (c *ASN1Codec) GetMessageType(data []byte) (models.E2MessageType, error) {
	pdu, err := c.DecodeE2AP_PDU(data)
	if err != nil {
		return models.UnknownMsg, fmt.Errorf("failed to decode PDU for message type detection: %w", err)
	}
	
	return c.getMessageTypeFromPDU(pdu), nil
}

// getMessageTypeFromPDU determines message type from a decoded PDU
func (c *ASN1Codec) getMessageTypeFromPDU(pdu *E2AP_PDU) models.E2MessageType {
	switch pdu.ProcedureCode {
	case E2SetupRequestID:
		return models.E2SetupRequestMsg
	case RICSubscriptionRequestID:
		return models.RICSubscriptionRequestMsg
	case RICSubscriptionDeleteRequestID:
		return models.RICSubscriptionDeleteRequestMsg
	case RICIndicationID:
		return models.RICIndicationMsg
	case RICControlRequestID:
		return models.RICControlRequestMsg
	case RICServiceUpdateID:
		return models.RICServiceUpdateMsg
	case E2NodeConfigurationUpdateID:
		return models.E2NodeConfigurationUpdateMsg
	case E2ConnectionUpdateID:
		return models.E2ConnectionUpdateMsg
	case ResetRequestID:
		return models.ResetRequestMsg
	case ErrorIndicationID:
		return models.ErrorIndicationMsg
	default:
		return models.UnknownMsg
	}
}

// validatePDU validates the structure of an E2AP PDU
// TODO: Update for full PER validation of RawValue content.
func (c *ASN1Codec) validatePDU(pdu *E2AP_PDU) error {
	if pdu == nil {
		return fmt.Errorf("PDU is nil")
	}
	
	// For now, we'll just check if RawValue.Bytes is present for simplicity.
	// A full PER validation would involve unmarshaling and validating the content.
	if len(pdu.Value.Bytes) == 0 {
		return fmt.Errorf("PDU Value bytes are empty")
	}
	
	// Validate procedure code based on the message type (InitiatingMessage, SuccessfulOutcome, UnsuccessfulOutcome)
	// This logic needs to be re-evaluated with the new E2AP_PDU structure.
	// For now, we'll rely on the ProcedureCode being valid.
	return nil
}

// validateInitiatingMessage validates an initiating message
// TODO: Update for full PER validation of RawValue content.
func (c *ASN1Codec) validateInitiatingMessage(pdu *E2AP_PDU) error {
	if pdu == nil || len(pdu.Value.Bytes) == 0 {
		return fmt.Errorf("initiating message PDU or Value bytes are nil/empty")
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
		if pdu.ProcedureCode == code {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("invalid procedure code for initiating message: %d", pdu.ProcedureCode)
	}
	
	return nil
}

// validateSuccessfulOutcome validates a successful outcome message
// TODO: Update for full PER validation of RawValue content.
func (c *ASN1Codec) validateSuccessfulOutcome(pdu *E2AP_PDU) error {
	if pdu == nil || len(pdu.Value.Bytes) == 0 {
		return fmt.Errorf("successful outcome PDU or Value bytes are nil/empty")
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
		if pdu.ProcedureCode == code {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("invalid procedure code for successful outcome: %d", pdu.ProcedureCode)
	}
	
	return nil
}

// validateUnsuccessfulOutcome validates an unsuccessful outcome message
// TODO: Update for full PER validation of RawValue content.
func (c *ASN1Codec) validateUnsuccessfulOutcome(pdu *E2AP_PDU) error {
	if pdu == nil || len(pdu.Value.Bytes) == 0 {
		return fmt.Errorf("unsuccessful outcome PDU or Value bytes are nil/empty")
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
		if pdu.ProcedureCode == code {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("invalid procedure code for unsuccessful outcome: %d", pdu.ProcedureCode)
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
	case *models.E2SetupRequest:
		return c.validateE2SetupRequest(v)
	case *models.E2SetupResponse:
		return c.validateE2SetupResponse(v)
	case *models.RICSubscriptionRequest:
		return c.validateRICSubscriptionRequest(v)
	case *models.RICIndication:
		return c.validateRICIndication(v)
	case *models.RICControlRequest:
		return c.validateRICControlRequest(v)
	default:
		return fmt.Errorf("unsupported message type: %T", msg)
	}
}

// validateE2SetupRequest validates an E2 Setup Request
func (c *ASN1Codec) validateE2SetupRequest(req *models.E2SetupRequest) error {
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
func (c *ASN1Codec) validateE2SetupResponse(resp *models.E2SetupResponse) error {
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
func (c *ASN1Codec) validateRICSubscriptionRequest(req *models.RICSubscriptionRequest) error {
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
func (c *ASN1Codec) validateRICIndication(indication *models.RICIndication) error {
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
func (c *ASN1Codec) validateRICControlRequest(req *models.RICControlRequest) error {
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