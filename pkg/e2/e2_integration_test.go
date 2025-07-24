// +build linux

package e2

import (
	"context"
	"testing"
	"time"

	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
	"github.com/ishidawataru/sctp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestE2Integration(t *testing.T) {
	log := logrus.New()

	// Create a basic SCTPConfig for the test
	serverConfig := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		Port:              38412,
		MaxConnections:    10,
		ConnectionTimeout: 5 * time.Second,
		HeartbeatInterval: 1 * time.Second,
		BufferSize:        1500,
		Streams:           3,
	}

	server, err := NewSCTPServer(serverConfig, log)
	assert.NoError(t, err)
	defer server.Stop(context.Background())

	// Setup message handler
	server.SetMessageHandler(func(connectionID, nodeID string, data []byte) {
		// Try to decode as E2SetupRequest
		setupReq, err := DecodeE2SetupRequest(data)
		if err == nil && setupReq != nil {
			// It's a setup request, send a response
			resp := &models.E2SetupResponse{
				TransactionID: setupReq.TransactionID,
				GlobalRICID: models.GlobalRICID{
					PLMNIdentity: []byte{0x01, 0x02, 0x03},
					RICIdentity:  []byte{0x01, 0x02, 0x03, 0x04},
				},
			}
			encodedResp, err := EncodeE2SetupResponse(resp)
			if err != nil {
				t.Errorf("Failed to encode setup response: %v", err)
				return
			}
			server.SendToConnection(connectionID, encodedResp)
			return
		}

		// Try to decode as RICSubscriptionRequest
		subReq, err := DecodeSubscriptionRequest(data)
		if err == nil && subReq != nil {
			// It's a subscription request, send a response
			resp := &models.RICSubscriptionResponse{
				RICRequestID: *subReq.RICrequestID,
				RICActionAdmitted: []models.RICActionAdmitted{
					{RICActionID: 1},
				},
			}
			encodedResp, err := EncodeRICSubscriptionResponse(resp)
			if err != nil {
				t.Errorf("Failed to encode subscription response: %v", err)
				return
			}
			server.SendToConnection(connectionID, encodedResp)
			return
		}
	})

	err = server.Start(context.Background())
	assert.NoError(t, err)

	// Client
	clientAddr, err := sctp.ResolveSCTPAddr("sctp", "127.0.0.1:38412")
	assert.NoError(t, err)
	clientConn, err := sctp.DialSCTP("sctp", nil, clientAddr)
	assert.NoError(t, err)
	defer clientConn.Close()

	t.Run("TestE2Setup", func(t *testing.T) {
		// E2 Setup Procedure
		setupReq := &models.E2SetupRequest{
			TransactionID:  1,
			GlobalE2NodeID: &models.GlobalE2NodeID{GNB_ID: &models.GNB_ID{GNB_ID: []byte("test-gnb-id")}},
			RANfunctions:   []*models.RANfunction{},
		}
		encodedReq, err := EncodeE2SetupRequest(setupReq)
		assert.NoError(t, err)

		_, err = clientConn.Write(encodedReq)
		assert.NoError(t, err)

		// Read response
		buf := make([]byte, 1500)
		n, err := clientConn.Read(buf)
		assert.NoError(t, err)
		assert.True(t, n > 0)

		// Decode and validate response
		setupResp, err := DecodeE2SetupResponse(buf[:n])
		assert.NoError(t, err)
		assert.NotNil(t, setupResp)
		assert.Equal(t, int64(1), setupResp.TransactionID)
	})

	t.Run("TestSubscription", func(t *testing.T) {
		// Subscription Procedure
		subscriptionReq := &models.RICSubscriptionRequest{
			RICrequestID:         &models.RICrequestID{RICrequestorID: 1, RICinstanceID: 1},
			RANfunctionID:        1,
			RICsubscriptionDetails: &models.RICsubscriptionDetails{},
		}
		encodedReq, err := EncodeSubscriptionRequest(subscriptionReq)
		assert.NoError(t, err)

		_, err = clientConn.Write(encodedReq)
		assert.NoError(t, err)

		// Read response
		buf := make([]byte, 1500)
		n, err := clientConn.Read(buf)
		assert.NoError(t, err)
		assert.True(t, n > 0)

		// Decode and validate response
		subResp, err := DecodeRICSubscriptionResponse(buf[:n])
		assert.NoError(t, err)
		assert.NotNil(t, subResp)
		assert.Equal(t, 1, subResp.RICRequestID.RICrequestorID)
		assert.Equal(t, 1, subResp.RICRequestID.RICinstanceID)
		assert.Len(t, subResp.RICActionAdmitted, 1)
		assert.Equal(t, int64(1), subResp.RICActionAdmitted[0].RICActionID)
	})
}
