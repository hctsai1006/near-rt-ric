package e2

import (
	"encoding/asn1"
	"fmt"

	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
)

// NOTE: This implementation uses the standard library's ASN.1 support, which
// uses DER encoding. For full O-RAN compliance, this should be replaced with
// a library that supports PER (Packed Encoding Rules).

// Criticality values
const (
	reject asn1.Enumerated = 0
	ignore asn1.Enumerated = 1
	notify asn1.Enumerated = 2
)

// PERCodec defines the interface for Packed Encoding Rules (PER) encoding/decoding.
// This is a placeholder for a future, full O-RAN compliant PER implementation.
type PERCodec interface {
	EncodePER(interface{}) ([]byte, error)
	DecodePER([]byte, interface{}) error
}

// E2AP_PDU is the top-level structure for E2AP messages
// The Value field is now asn1.RawValue to allow for flexible decoding based on ProcedureCode.
// TODO: Replace with a proper PER-compliant structure and encoding/decoding logic.
type E2AP_PDU struct {
	ProcedureCode int64           `asn1:"value"`
	Criticality   asn1.Enumerated `asn1:"value"`
	Value         asn1.RawValue   `asn1:"value"` // Use RawValue for deferred decoding
}

func EncodeGlobalE2NodeID(msg *models.GlobalE2NodeID) ([]byte, error) {
    // This is a placeholder. Proper ASN.1 DER encoding would be implemented here.
    return asn1.Marshal(*msg)
}

func EncodeE2SetupRequest(req *models.E2SetupRequest) ([]byte, error) {
    // This is a placeholder. Proper ASN.1 DER encoding would be implemented here.
    // TODO: Replace with PER encoding.
    reqBytes, err := asn1.Marshal(*req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal E2SetupRequest: %w", err)
    }

    pdu := &E2AP_PDU{
        ProcedureCode: 1,
        Criticality:   reject,
        Value:         asn1.RawValue{Bytes: reqBytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true},
    }
    return asn1.Marshal(*pdu)
}

func EncodeSubscriptionRequest(req *models.RICSubscriptionRequest) ([]byte, error) {
    // This is a placeholder. Proper ASN.1 DER encoding would be implemented here.
    // TODO: Replace with PER encoding.
    reqBytes, err := asn1.Marshal(*req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal RICSubscriptionRequest: %w", err)
    }

    pdu := &E2AP_PDU{
        ProcedureCode: 12,
        Criticality:   reject,
        Value:         asn1.RawValue{Bytes: reqBytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true},
    }
    return asn1.Marshal(*pdu)
}

func EncodeControlRequest(req *models.RICcontrolRequest) ([]byte, error) {
	// This is a placeholder. Proper ASN.1 DER encoding would be implemented here.
    // TODO: Replace with PER encoding.
    reqBytes, err := asn1.Marshal(*req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal RICcontrolRequest: %w", err)
    }

	pdu := &E2AP_PDU{
		ProcedureCode: 13, // ProcedureCode for RIC Control
		Criticality:   reject,
		Value:         asn1.RawValue{Bytes: reqBytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true},
	}
	return asn1.Marshal(*pdu)
}

// DecodeE2AP_PDU decodes an E2AP PDU from ASN.1 DER bytes.
// TODO: Replace with PER decoding.
func DecodeE2AP_PDU(data []byte) (*E2AP_PDU, error) {
    var pdu E2AP_PDU
    rest, err := asn1.Unmarshal(data, &pdu)
    if err != nil {
        return nil, fmt.Errorf("failed to unmarshal E2AP PDU: %w", err)
    }
    if len(rest) > 0 {
        return nil, fmt.Errorf("unexpected remaining bytes after PDU decoding: %d bytes", len(rest))
    }

    // Now, decode the Value based on ProcedureCode
    switch pdu.ProcedureCode {
    case 1: // E2SetupRequest
        var req models.E2SetupRequest
        _, err := asn1.Unmarshal(pdu.Value.Bytes, &req)
        if err != nil {
            return nil, fmt.Errorf("failed to unmarshal E2SetupRequest value: %w", err)
        }
        pdu.Value = asn1.RawValue{Bytes: pdu.Value.Bytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true}
        // For now, we'll just keep the RawValue, as the higher-level codec will handle the specific message type.
        // In a full PER implementation, this would be where the specific message struct is populated.

    case 12: // RICSubscriptionRequest
        var req models.RICSubscriptionRequest
        _, err := asn1.Unmarshal(pdu.Value.Bytes, &req)
        if err != nil {
            return nil, fmt.Errorf("failed to unmarshal RICSubscriptionRequest value: %w", err)
        }
        pdu.Value = asn1.RawValue{Bytes: pdu.Value.Bytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true}

    case 13: // RICcontrolRequest
        var req models.RICcontrolRequest
        _, err := asn1.Unmarshal(pdu.Value.Bytes, &req)
        if err != nil {
            return nil, fmt.Errorf("failed to unmarshal RICcontrolRequest value: %w", err)
        }
        pdu.Value = asn1.RawValue{Bytes: pdu.Value.Bytes, Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true}

    default:
        return nil, fmt.Errorf("unsupported ProcedureCode: %d", pdu.ProcedureCode)
    }

    return &pdu, nil
}