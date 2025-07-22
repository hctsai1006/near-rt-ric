package e2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data structures for E2AP testing
var (
	testGlobalE2NodeID = GlobalE2NodeID{
		GNBId: &GlobalGNBId{
			PLMNId: []byte{0x12, 0x34, 0x56},
			GNBId:  []byte{0x78, 0x9A, 0xBC, 0xDE},
		},
	}

	testGlobalRICID = GlobalRICID{
		PLMNId:      []byte{0x12, 0x34, 0x56},
		RICId:       []byte{0x78, 0x9A},
		RICInstance: 1,
	}

	testRANFunctions = []RANFunction{
		{
			RanFunctionId:         1,
			RanFunctionDefinition: []byte("test-ran-function-definition"),
			RanFunctionRevision:   1,
		},
	}
)

func TestNewASN1Encoder(t *testing.T) {
	encoder := NewASN1Encoder()
	
	assert.NotNil(t, encoder)
	assert.NotNil(t, encoder.logger)
}

func TestE2SetupRequestEncoding(t *testing.T) {
	encoder := NewASN1Encoder()
	
	// Create test E2 Setup Request
	setupRequest := &E2SetupRequest{
		GlobalE2NodeID: testGlobalE2NodeID,
		RANFunctions:   testRANFunctions,
	}
	
	// Test encoding
	start := time.Now()
	encoded, err := encoder.EncodeE2SetupRequest(setupRequest)
	duration := time.Since(start)
	
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)
	assert.Less(t, duration, 10*time.Millisecond, "Encoding should be fast")
	
	t.Logf("Encoded E2 Setup Request: %d bytes in %v", len(encoded), duration)
}

func TestE2SetupResponseEncoding(t *testing.T) {
	encoder := NewASN1Encoder()
	
	// Create test E2 Setup Response
	setupResponse := &E2SetupResponse{
		GlobalRICID:          testGlobalRICID,
		RANFunctionsAccepted: testRANFunctions,
	}
	
	// Test encoding
	start := time.Now()
	encoded, err := encoder.EncodeE2SetupResponse(setupResponse)
	duration := time.Since(start)
	
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)
	assert.Less(t, duration, 10*time.Millisecond, "Encoding should be fast")
	
	t.Logf("Encoded E2 Setup Response: %d bytes in %v", len(encoded), duration)
}

func TestPDUEncodeDecodeRoundtrip(t *testing.T) {
	encoder := NewASN1Encoder()
	
	// Create test PDU with initiating message
	originalPDU := &E2AP_PDU{
		InitiatingMessage: &InitiatingMessage{
			ProcedureCode: E2SetupRequestID,
			Criticality:   CriticalityReject,
			Value: &E2SetupRequestIEs{
				GlobalE2NodeID: asn1RawValueFromBytes(t, []byte("test-node-id")),
				RANFunctions:   asn1RawValueFromBytes(t, []byte("test-ran-functions")),
			},
		},
	}
	
	// Encode PDU
	encoded, err := encoder.encodePDU(originalPDU)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)
	
	// Decode PDU
	decodedPDU, err := encoder.DecodeE2AP_PDU(encoded)
	require.NoError(t, err)
	assert.NotNil(t, decodedPDU)
	
	// Verify structure
	assert.NotNil(t, decodedPDU.InitiatingMessage)
	assert.Equal(t, E2SetupRequestID, decodedPDU.InitiatingMessage.ProcedureCode)
	assert.Equal(t, CriticalityReject, decodedPDU.InitiatingMessage.Criticality)
	
	t.Logf("Round-trip encoding/decoding successful for %d bytes", len(encoded))
}

func TestGlobalE2NodeIDEncoding(t *testing.T) {
	encoder := NewASN1Encoder()
	
	// Test encoding Global E2 Node ID
	encoded, err := encoder.encodeGlobalE2NodeID(testGlobalE2NodeID)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)
	
	t.Logf("Encoded Global E2 Node ID: %d bytes", len(encoded))
}

func TestGlobalRICIDEncoding(t *testing.T) {
	encoder := NewASN1Encoder()
	
	// Test encoding Global RIC ID
	encoded, err := encoder.encodeGlobalRICID(testGlobalRICID)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)
	
	t.Logf("Encoded Global RIC ID: %d bytes", len(encoded))
}

func TestRANFunctionsEncoding(t *testing.T) {
	encoder := NewASN1Encoder()
	
	// Test encoding RAN Functions
	encoded, err := encoder.encodeRANFunctions(testRANFunctions)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)
	
	t.Logf("Encoded RAN Functions: %d bytes", len(encoded))
}

func TestInvalidPDUDecoding(t *testing.T) {
	encoder := NewASN1Encoder()
	
	// Test with invalid data
	invalidData := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	pdu, err := encoder.DecodeE2AP_PDU(invalidData)
	
	assert.Error(t, err)
	assert.Nil(t, pdu)
	assert.Contains(t, err.Error(), "failed to decode E2AP-PDU with PER decoding")
}

func TestEmptyDataDecoding(t *testing.T) {
	encoder := NewASN1Encoder()
	
	// Test with empty data
	emptyData := []byte{}
	pdu, err := encoder.DecodeE2AP_PDU(emptyData)
	
	assert.Error(t, err)
	assert.Nil(t, pdu)
}

// Benchmark tests for performance validation
func BenchmarkE2SetupRequestEncoding(b *testing.B) {
	encoder := NewASN1Encoder()
	setupRequest := &E2SetupRequest{
		GlobalE2NodeID: testGlobalE2NodeID,
		RANFunctions:   testRANFunctions,
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := encoder.EncodeE2SetupRequest(setupRequest)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPDUDecoding(b *testing.B) {
	encoder := NewASN1Encoder()
	
	// Create test data
	pdu := &E2AP_PDU{
		InitiatingMessage: &InitiatingMessage{
			ProcedureCode: E2SetupRequestID,
			Criticality:   CriticalityReject,
			Value: &E2SetupRequestIEs{
				GlobalE2NodeID: asn1RawValueFromBytes(b, []byte("test-node-id")),
			},
		},
	}
	
	encoded, err := encoder.encodePDU(pdu)
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := encoder.DecodeE2AP_PDU(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper functions for testing
func asn1RawValueFromBytes(t testing.TB, data []byte) asn1.RawValue {
	return asn1.RawValue{Bytes: data}
}

// Test table-driven approach for multiple scenarios
func TestE2APEncoding_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		procedureID int64
		criticality asn1.Enumerated
		expectError bool
	}{
		{
			name:        "E2 Setup Request",
			procedureID: E2SetupRequestID,
			criticality: CriticalityReject,
			expectError: false,
		},
		{
			name:        "RIC Subscription Request", 
			procedureID: RICSubscriptionRequestID,
			criticality: CriticalityReject,
			expectError: false,
		},
		{
			name:        "RIC Indication",
			procedureID: RICIndicationID,
			criticality: CriticalityIgnore,
			expectError: false,
		},
	}
	
	encoder := NewASN1Encoder()
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := &E2AP_PDU{
				InitiatingMessage: &InitiatingMessage{
					ProcedureCode: tt.procedureID,
					Criticality:   tt.criticality,
					Value:         &E2SetupRequestIEs{},
				},
			}
			
			encoded, err := encoder.encodePDU(pdu)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, encoded)
				
				// Verify round-trip
				decoded, err := encoder.DecodeE2AP_PDU(encoded)
				assert.NoError(t, err)
				assert.NotNil(t, decoded)
				assert.Equal(t, tt.procedureID, decoded.InitiatingMessage.ProcedureCode)
			}
		})
	}
}

// Performance test to ensure E2 latency requirements (< 10ms)
func TestE2LatencyRequirement(t *testing.T) {
	encoder := NewASN1Encoder()
	setupRequest := &E2SetupRequest{
		GlobalE2NodeID: testGlobalE2NodeID,
		RANFunctions:   testRANFunctions,
	}
	
	// Test multiple iterations to ensure consistent performance
	const iterations = 100
	var totalDuration time.Duration
	
	for i := 0; i < iterations; i++ {
		start := time.Now()
		encoded, err := encoder.EncodeE2SetupRequest(setupRequest)
		require.NoError(t, err)
		
		_, err = encoder.DecodeE2AP_PDU(encoded)
		require.NoError(t, err)
		
		duration := time.Since(start)
		totalDuration += duration
	}
	
	avgDuration := totalDuration / iterations
	
	// O-RAN E2 interface requires latency < 10ms for most operations
	assert.Less(t, avgDuration, 5*time.Millisecond, 
		"Average E2AP encode/decode cycle should be < 5ms for O-RAN compliance, got %v", avgDuration)
	
	t.Logf("Average E2AP encode/decode latency: %v (requirement: < 10ms)", avgDuration)
}