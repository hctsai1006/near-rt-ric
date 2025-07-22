package e2

import (
	"encoding/asn1"
	"fmt"
	"time"

	"github.com/free5gc/aper"
	"github.com/sirupsen/logrus"
)

// ASN1Encoder handles ASN.1 encoding/decoding for E2AP messages
type ASN1Encoder struct {
	logger *logrus.Logger
}

// NewASN1Encoder creates a new ASN.1 encoder
func NewASN1Encoder() *ASN1Encoder {
	return &ASN1Encoder{
		logger: logrus.New(),
	}
}

// E2AP_PDU represents the main E2AP Protocol Data Unit structure
type E2AP_PDU struct {
	InitiatingMessage    *InitiatingMessage    `asn1:"tag:0,optional"`
	SuccessfulOutcome    *SuccessfulOutcome    `asn1:"tag:1,optional"`
	UnsuccessfulOutcome  *UnsuccessfulOutcome  `asn1:"tag:2,optional"`
}

// InitiatingMessage represents an initiating message
type InitiatingMessage struct {
	ProcedureCode int64             `asn1:"tag:0"`
	Criticality   asn1.Enumerated   `asn1:"tag:1"`
	Value         E2APElementaryProcedure `asn1:"tag:2"`
}

// SuccessfulOutcome represents a successful outcome message
type SuccessfulOutcome struct {
	ProcedureCode int64             `asn1:"tag:0"`
	Criticality   asn1.Enumerated   `asn1:"tag:1"`
	Value         E2APElementaryProcedure `asn1:"tag:2"`
}

// UnsuccessfulOutcome represents an unsuccessful outcome message
type UnsuccessfulOutcome struct {
	ProcedureCode int64             `asn1:"tag:0"`
	Criticality   asn1.Enumerated   `asn1:"tag:1"`
	Value         E2APElementaryProcedure `asn1:"tag:2"`
}

// E2APElementaryProcedure is a placeholder for specific procedure values
type E2APElementaryProcedure interface{}

// E2SetupRequestIEs represents E2 Setup Request Information Elements
type E2SetupRequestIEs struct {
	GlobalE2NodeID asn1.RawValue `asn1:"tag:3"`
	RANFunctions   asn1.RawValue `asn1:"tag:10,optional"`
	E2NodeConfigUpdate asn1.RawValue `asn1:"tag:50,optional"`
}

// E2SetupResponseIEs represents E2 Setup Response Information Elements
type E2SetupResponseIEs struct {
	GlobalRICID          asn1.RawValue `asn1:"tag:4"`
	RANFunctionsAccepted asn1.RawValue `asn1:"tag:9,optional"`
	RANFunctionsRejected asn1.RawValue `asn1:"tag:13,optional"`
	E2NodeConfigUpdateAck asn1.RawValue `asn1:"tag:52,optional"`
}

// E2SetupFailureIEs represents E2 Setup Failure Information Elements  
type E2SetupFailureIEs struct {
	Cause                  asn1.RawValue `asn1:"tag:1"`
	TimeToWait             asn1.RawValue `asn1:"tag:31,optional"`
	CriticalityDiagnostics asn1.RawValue `asn1:"tag:17,optional"`
}

// RICSubscriptionRequestIEs represents RIC Subscription Request Information Elements
type RICSubscriptionRequestIEs struct {
	RICRequestID           asn1.RawValue `asn1:"tag:29"`
	RANFunctionID          asn1.RawValue `asn1:"tag:5"`
	RICSubscriptionDetails asn1.RawValue `asn1:"tag:30"`
}

// RICSubscriptionResponseIEs represents RIC Subscription Response Information Elements
type RICSubscriptionResponseIEs struct {
	RICRequestID          asn1.RawValue `asn1:"tag:29"`
	RANFunctionID         asn1.RawValue `asn1:"tag:5"`
	RICActionAdmitted     asn1.RawValue `asn1:"tag:17"`
	RICActionNotAdmitted  asn1.RawValue `asn1:"tag:18,optional"`
}

// RICIndicationIEs represents RIC Indication Information Elements
type RICIndicationIEs struct {
	RICRequestID       asn1.RawValue `asn1:"tag:29"`
	RANFunctionID      asn1.RawValue `asn1:"tag:5"`
	RICActionID        asn1.RawValue `asn1:"tag:15"`
	RICIndicationSN    asn1.RawValue `asn1:"tag:27,optional"`
	RICIndicationType  asn1.RawValue `asn1:"tag:28"`
	RICIndicationHeader asn1.RawValue `asn1:"tag:25"`
	RICIndicationMessage asn1.RawValue `asn1:"tag:26"`
	RICCallProcessID   asn1.RawValue `asn1:"tag:20,optional"`
}

// EncodeE2SetupRequest encodes an E2 Setup Request message
func (enc *ASN1Encoder) EncodeE2SetupRequest(req *E2SetupRequest) ([]byte, error) {
	start := time.Now()
	defer func() {
		enc.logger.WithField("duration", time.Since(start)).Debug("E2 Setup Request encoding completed")
	}()

	// Create procedure-specific IEs
	ies := &E2SetupRequestIEs{}

	// Encode Global E2 Node ID
	globalE2NodeIDBytes, err := enc.encodeGlobalE2NodeID(req.GlobalE2NodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to encode Global E2 Node ID: %w", err)
	}
	ies.GlobalE2NodeID = asn1.RawValue{Bytes: globalE2NodeIDBytes}

	// Encode RAN Functions if present
	if len(req.RANFunctions) > 0 {
		ranFunctionsBytes, err := enc.encodeRANFunctions(req.RANFunctions)
		if err != nil {
			return nil, fmt.Errorf("failed to encode RAN Functions: %w", err)
		}
		ies.RANFunctions = asn1.RawValue{Bytes: ranFunctionsBytes}
	}

	// Create initiating message
	initMsg := &InitiatingMessage{
		ProcedureCode: E2SetupRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         ies,
	}

	// Create PDU
	pdu := &E2AP_PDU{
		InitiatingMessage: initMsg,
	}

	return enc.encodePDU(pdu)
}

// EncodeE2SetupResponse encodes an E2 Setup Response message
func (enc *ASN1Encoder) EncodeE2SetupResponse(resp *E2SetupResponse) ([]byte, error) {
	start := time.Now()
	defer func() {
		enc.logger.WithField("duration", time.Since(start)).Debug("E2 Setup Response encoding completed")
	}()

	// Create procedure-specific IEs
	ies := &E2SetupResponseIEs{}

	// Encode Global RIC ID
	globalRICIDBytes, err := enc.encodeGlobalRICID(resp.GlobalRICID)
	if err != nil {
		return nil, fmt.Errorf("failed to encode Global RIC ID: %w", err)
	}
	ies.GlobalRICID = asn1.RawValue{Bytes: globalRICIDBytes}

	// Create successful outcome
	successMsg := &SuccessfulOutcome{
		ProcedureCode: E2SetupRequestID,
		Criticality:   asn1.Enumerated(CriticalityReject),
		Value:         ies,
	}

	// Create PDU
	pdu := &E2AP_PDU{
		SuccessfulOutcome: successMsg,
	}

	return enc.encodePDU(pdu)
}

// DecodeE2AP_PDU decodes an ASN.1 PER encoded E2AP PDU
func (enc *ASN1Encoder) DecodeE2AP_PDU(data []byte) (*E2AP_PDU, error) {
	start := time.Now()
	defer func() {
		enc.logger.WithField("duration", time.Since(start)).Debug("E2AP PDU decoding completed")
	}()

	var pdu E2AP_PDU
	err := aper.Unmarshal(data, &pdu)
	if err != nil {
		return nil, fmt.Errorf("failed to decode E2AP-PDU with PER decoding: %w", err)
	}

	enc.logger.WithField("data_length", len(data)).Debug("E2AP PDU decoded successfully")
	return &pdu, nil
}

// Helper methods for encoding specific structures

func (enc *ASN1Encoder) encodePDU(pdu *E2AP_PDU) ([]byte, error) {
	// Use APER (Aligned PER) encoding as required by O-RAN E2AP specification
	encoded, err := aper.Marshal(*pdu)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal E2AP-PDU with PER encoding: %w", err)
	}
	return encoded, nil
}

func (enc *ASN1Encoder) encodeGlobalE2NodeID(nodeID GlobalE2NodeID) ([]byte, error) {
	// Use APER (Aligned PER) encoding for Global E2 Node ID according to E2AP specification
	encoded, err := aper.Marshal(nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to encode Global E2 Node ID with PER: %w", err)
	}
	return encoded, nil
}

func (enc *ASN1Encoder) encodeGlobalRICID(ricID GlobalRICID) ([]byte, error) {
	// Use APER (Aligned PER) encoding for Global RIC ID according to E2AP specification
	encoded, err := aper.Marshal(ricID)
	if err != nil {
		return nil, fmt.Errorf("failed to encode Global RIC ID with PER: %w", err)
	}
	return encoded, nil
}

func (enc *ASN1Encoder) encodeRANFunctions(functions []RANFunction) ([]byte, error) {
	// Use APER (Aligned PER) encoding for RAN Functions according to E2AP specification
	encoded, err := aper.Marshal(functions)
	if err != nil {
		return nil, fmt.Errorf("failed to encode RAN Functions with PER: %w", err)
	}
	return encoded, nil
}
