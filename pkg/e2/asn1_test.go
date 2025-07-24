package e2

import (
	"testing"

	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2SetupRequestEncoding tests the encoding of an E2SetupRequest.
func TestE2SetupRequestEncoding(t *testing.T) {
	t.Skip("Skipping test due to issue with aper library handling of CHOICE types")
	req := &models.E2SetupRequest{
		GlobalE2NodeID: &models.GlobalE2NodeID{
			GNB_ID: &models.GNB_ID{
				GNB_ID: []byte{0xDE, 0xAD, 0xBE, 0xEF},
			},
		},
		RANfunctions: []*models.RANfunction{
			{
				RANfunctionID:         2,
				RANfunctionDefinition: []byte("Test_Function"),
				RANfunctionRevision:   1,
			},
		},
	}

	encoded, err := EncodeE2SetupRequest(req)
	require.NoError(t, err, "Should encode E2SetupRequest without errors")
	assert.NotEmpty(t, encoded, "Encoded data should not be empty")
}

// TestE2SetupRequestDecoding tests the decoding of an E2SetupRequest.
func TestE2SetupRequestDecoding(t *testing.T) {
	t.Skip("Skipping test due to issue with aper library handling of CHOICE types")
	req := &models.E2SetupRequest{
		GlobalE2NodeID: &models.GlobalE2NodeID{
			GNB_ID: &models.GNB_ID{
				GNB_ID: []byte{0xDE, 0xAD, 0xBE, 0xEF},
			},
		},
		RANfunctions: []*models.RANfunction{
			{
				RANfunctionID:         2,
				RANfunctionDefinition: []byte("Test_Function"),
				RANfunctionRevision:   1,
			},
		},
	}

	encoded, err := EncodeE2SetupRequest(req)
	require.NoError(t, err, "Should encode E2SetupRequest without errors")

	decoded, err := DecodeE2SetupRequest(encoded)
	require.NoError(t, err, "Should decode E2SetupRequest without errors")
	assert.Equal(t, req.GlobalE2NodeID.GNB_ID.GNB_ID, decoded.GlobalE2NodeID.GNB_ID.GNB_ID, "Decoded GlobalE2NodeID should match original")
	assert.Equal(t, len(req.RANfunctions), len(decoded.RANfunctions), "Number of RAN functions should match")
	assert.Equal(t, req.RANfunctions[0].RANfunctionID, decoded.RANfunctions[0].RANfunctionID, "RAN function ID should match")
}
