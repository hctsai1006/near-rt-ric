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

// E2AP_PDU_Value is a CHOICE of the different E2AP messages
type E2AP_PDU_Value struct {
	E2SetupRequest         *models.E2SetupRequest         `asn1:"choice:e2SetupRequest"`
	RICsubscriptionRequest *models.RICSubscriptionRequest `asn1:"choice:ricSubscriptionRequest"`
	RICcontrolRequest      *models.RICcontrolRequest      `asn1:"choice:ricControlRequest"`
}

// E2AP_PDU is the top-level structure for E2AP messages
type E2AP_PDU struct {
	ProcedureCode int64           `asn1:"value"`
	Criticality   asn1.Enumerated `asn1:"value"`
	Value         E2AP_PDU_Value  `asn1:"value"`
}

func EncodeGlobalE2NodeID(msg *models.GlobalE2NodeID) ([]byte, error) {
    // This is a placeholder. Proper ASN.1 DER encoding would be implemented here.
    return asn1.Marshal(*msg)
}

func EncodeE2SetupRequest(req *models.E2SetupRequest) ([]byte, error) {
    // This is a placeholder. Proper ASN.1 DER encoding would be implemented here.
    pdu := &E2AP_PDU{
        ProcedureCode: 1,
        Criticality:   reject,
        Value: E2AP_PDU_Value{
            E2SetupRequest: req,
        },
    }
    return asn1.Marshal(*pdu)
}

func EncodeSubscriptionRequest(req *models.RICSubscriptionRequest) ([]byte, error) {
    // This is a placeholder. Proper ASN.1 DER encoding would be implemented here.
    pdu := &E2AP_PDU{
        ProcedureCode: 12,
        Criticality:   reject,
        Value: E2AP_PDU_Value{
            RICsubscriptionRequest: req,
        },
    }
    return asn1.Marshal(*pdu)
}

func EncodeControlRequest(req *models.RICcontrolRequest) ([]byte, error) {
	// This is a placeholder. Proper ASN.1 DER encoding would be implemented here.
	pdu := &E2AP_PDU{
		ProcedureCode: 13, // ProcedureCode for RIC Control
		Criticality:   reject,
		Value: E2AP_PDU_Value{
			RICcontrolRequest: req,
		},
	}
	return asn1.Marshal(*pdu)
}

// DecodeE2AP_PDU decodes an E2AP PDU from ASN.1 DER bytes.
func DecodeE2AP_PDU(data []byte) (*E2AP_PDU, error) {
    var pdu E2AP_PDU
    _, err := asn1.Unmarshal(data, &pdu)
    if err != nil {
        return nil, err
    }
    return &pdu, nil
}