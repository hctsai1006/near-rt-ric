package e2

import (
	"fmt"

	"github.com/onosproject/onos-lib-go/pkg/asn1/aper"
	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
)

// Codec defines the interface for ASN.1 encoding/decoding.
type Codec interface {
	Encode(interface{}) ([]byte, error)
	Decode([]byte, interface{}) error
}

// ASN1DERCodec provides a concrete implementation of Codec using the standard library's DER encoding.
type ASN1DERCodec struct{}

// NewASN1DERCodec creates a new ASN1DERCodec.
func NewASN1DERCodec() *ASN1DERCodec {
	return &ASN1DERCodec{}
}

// Encode encodes the given value using APER.
func (c *ASN1DERCodec) Encode(val interface{}) ([]byte, error) {
	return aper.Marshal(val, nil, nil)
}

// Decode decodes the given data using APER.
func (c *ASN1DERCodec) Decode(data []byte, val interface{}) error {
	return aper.Unmarshal(data, val, nil, nil)
}



// EncodeE2SetupRequest encodes an E2SetupRequest into a APER-encoded E2AP_PDU.
func EncodeE2SetupRequest(req *models.E2SetupRequest) ([]byte, error) {
	pdu := &models.E2AP_PDU{
		InitiatingMessage: &models.InitiatingMessage{
			ProcedureCode: models.ProcedureCodeE2Setup,
			Criticality:   models.CriticalityReject,
			Value:         req,
		},
		SuccessfulOutcome:   nil,
		UnsuccessfulOutcome: nil,
	}
	return aper.Marshal(pdu, nil, models.E2AP_PDU_TypeMaps)
}

func DecodeE2SetupRequest(data []byte) (*models.E2SetupRequest, error) {
	var pdu models.E2AP_PDU
	if err := aper.Unmarshal(data, &pdu, nil, models.E2AP_PDU_TypeMaps); err != nil {
		return nil, fmt.Errorf("failed to decode E2AP_PDU: %w", err)
	}
	if pdu.InitiatingMessage == nil || pdu.InitiatingMessage.Value == nil {
		return nil, fmt.Errorf("decoded PDU does not contain an InitiatingMessage with a value")
	}
	if req, ok := pdu.InitiatingMessage.Value.(*models.E2SetupRequest); ok {
		return req, nil
	}
	return nil, fmt.Errorf("decoded PDU is not an E2SetupRequest")
}

// EncodeSubscriptionRequest encodes a RICSubscriptionRequest into a APER-encoded E2AP_PDU.
func EncodeSubscriptionRequest(req *models.RICSubscriptionRequest) ([]byte, error) {
	pdu := &models.E2AP_PDU{
		InitiatingMessage: &models.InitiatingMessage{
			ProcedureCode: models.ProcedureCodeRICSubscription,
			Criticality:   models.CriticalityReject,
			Value:         req,
		},
		SuccessfulOutcome:   nil,
		UnsuccessfulOutcome: nil,
	}
	return aper.Marshal(pdu, nil, models.E2AP_PDU_TypeMaps)
}

func DecodeSubscriptionRequest(data []byte) (*models.RICSubscriptionRequest, error) {
	var pdu models.E2AP_PDU
	if err := aper.Unmarshal(data, &pdu, nil, models.E2AP_PDU_TypeMaps); err != nil {
		return nil, fmt.Errorf("failed to decode E2AP_PDU: %w", err)
	}
	if pdu.InitiatingMessage == nil || pdu.InitiatingMessage.Value == nil {
		return nil, fmt.Errorf("decoded PDU does not contain an InitiatingMessage with a value")
	}
	if req, ok := pdu.InitiatingMessage.Value.(*models.RICSubscriptionRequest); ok {
		return req, nil
	}
	return nil, fmt.Errorf("decoded PDU is not a RICSubscriptionRequest")
}

// EncodeControlRequest encodes a RICcontrolRequest into a APER-encoded E2AP_PDU.
func EncodeControlRequest(req *models.RICcontrolRequest) ([]byte, error) {
	pdu := &models.E2AP_PDU{
		InitiatingMessage: &models.InitiatingMessage{
			ProcedureCode: models.ProcedureCodeRICControl,
			Criticality:   models.CriticalityReject,
			Value:         req,
		},
		SuccessfulOutcome:   nil,
		UnsuccessfulOutcome: nil,
	}
	return aper.Marshal(pdu, nil, models.E2AP_PDU_TypeMaps)
}

func DecodeControlRequest(data []byte) (*models.RICcontrolRequest, error) {
	var pdu models.E2AP_PDU
	if err := aper.Unmarshal(data, &pdu, nil, models.E2AP_PDU_TypeMaps); err != nil {
		return nil, fmt.Errorf("failed to decode E2AP_PDU: %w", err)
	}
	if pdu.InitiatingMessage == nil || pdu.InitiatingMessage.Value == nil {
		return nil, fmt.Errorf("decoded PDU does not contain an InitiatingMessage with a value")
	}
	if req, ok := pdu.InitiatingMessage.Value.(*models.RICcontrolRequest); ok {
		return req, nil
	}
	return nil, fmt.Errorf("decoded PDU is not a RICcontrolRequest")
}

// EncodeE2SetupResponse encodes an E2SetupResponse into a APER-encoded E2AP_PDU.
func EncodeE2SetupResponse(resp *models.E2SetupResponse) ([]byte, error) {
	pdu := &models.E2AP_PDU{
		SuccessfulOutcome: &models.SuccessfulOutcome{
			ProcedureCode: models.ProcedureCodeE2Setup,
			Criticality:   models.CriticalityReject,
			Value:         resp,
		},
		InitiatingMessage:   nil,
		UnsuccessfulOutcome: nil,
	}
	return aper.Marshal(pdu, nil, models.E2AP_PDU_TypeMaps)
}

// DecodeE2SetupResponse decodes an E2SetupResponse from a APER-encoded E2AP_PDU.
func DecodeE2SetupResponse(data []byte) (*models.E2SetupResponse, error) {
	var pdu models.E2AP_PDU
	if err := aper.Unmarshal(data, &pdu, nil, models.E2AP_PDU_TypeMaps); err != nil {
		return nil, fmt.Errorf("failed to decode E2AP_PDU: %w", err)
	}
	if pdu.SuccessfulOutcome == nil || pdu.SuccessfulOutcome.Value == nil {
		return nil, fmt.Errorf("decoded PDU does not contain a SuccessfulOutcome with a value")
	}
	if resp, ok := pdu.SuccessfulOutcome.Value.(*models.E2SetupResponse); ok {
		return resp, nil
	}
	return nil, fmt.Errorf("decoded PDU is not an E2SetupResponse")
}

// EncodeRICSubscriptionResponse encodes a RICSubscriptionResponse into a APER-encoded E2AP_PDU.
func EncodeRICSubscriptionResponse(resp *models.RICSubscriptionResponse) ([]byte, error) {
	pdu := &models.E2AP_PDU{
		SuccessfulOutcome: &models.SuccessfulOutcome{
			ProcedureCode: models.ProcedureCodeRICSubscription,
			Criticality:   models.CriticalityReject,
			Value:         resp,
		},
		InitiatingMessage:   nil,
		UnsuccessfulOutcome: nil,
	}
	return aper.Marshal(pdu, nil, models.E2AP_PDU_TypeMaps)
}

// DecodeRICSubscriptionResponse decodes a RICSubscriptionResponse from a APER-encoded E2AP_PDU.
func DecodeRICSubscriptionResponse(data []byte) (*models.RICSubscriptionResponse, error) {
	var pdu models.E2AP_PDU
	if err := aper.Unmarshal(data, &pdu, nil, models.E2AP_PDU_TypeMaps); err != nil {
		return nil, fmt.Errorf("failed to decode E2AP_PDU: %w", err)
	}
	if pdu.SuccessfulOutcome == nil || pdu.SuccessfulOutcome.Value == nil {
		return nil, fmt.Errorf("decoded PDU does not contain a SuccessfulOutcome with a value")
	}
	if resp, ok := pdu.SuccessfulOutcome.Value.(*models.RICSubscriptionResponse); ok {
		return resp, nil
	}
	return nil, fmt.Errorf("decoded PDU is not a RICSubscriptionResponse")
}