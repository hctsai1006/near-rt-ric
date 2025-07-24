package e2

import (
	"testing"
	"time"

	"github.com/hctsai1006/near-rt-ric/pkg/e2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2FullFlow(t *testing.T) {
	// Server setup
	serverTransport := &e2.SCTPTransport{}
	err := serverTransport.Listen()
	require.NoError(t, err, "server should listen without error")
	defer serverTransport.listener.Close()

	// Client setup
	clientTransport := &e2.SCTPTransport{}

	// Simulate a client connecting to the server
	go func() {
		conn, err := serverTransport.listener.Accept()
		if err != nil {
			return // Test will fail on timeout or other error
		}
		serverTransport.connections["test-node"] = conn.(*sctp.SCTPConn)
	}()

	// Client connects
	addr, err := sctp.ResolveSCTPAddr("sctp", "127.0.0.1:36421")
	require.NoError(t, err)
	conn, err := sctp.DialSCTP("sctp", nil, addr)
	require.NoError(t, err)
	clientTransport.connections["server"] = conn

	// 1. E2 Setup Request
	setupReq := &e2.E2SetupRequest{TransactionID: 1}
	encodedSetupReq, err := e2.EncodeE2SetupRequest(setupReq)
	require.NoError(t, err)

	err = clientTransport.SendMessage("server", encodedSetupReq, 0)
	require.NoError(t, err)

	// Server receives and decodes
	buffer := make([]byte, 1024)
	n, _, err := serverTransport.connections["test-node"].SCTPRead(buffer, nil)
	require.NoError(t, err)
	decodedSetupReq, err := e2.DecodeE2AP_PDU(buffer[:n])
	require.NoError(t, err)
	assert.NotNil(t, decodedSetupReq.Value.E2SetupRequest)
	assert.Equal(t, int64(1), decodedSetupReq.Value.E2SetupRequest.TransactionID)

	// 2. RIC Subscription Request
	subReq := &e2.RICsubscriptionRequest{TransactionID: 2}
	encodedSubReq, err := e2.EncodeSubscriptionRequest(subReq)
	require.NoError(t, err)

	err = clientTransport.SendMessage("server", encodedSubReq, 1)
	require.NoError(t, err)

	// Server receives and decodes
	n, _, err = serverTransport.connections["test-node"].SCTPRead(buffer, nil)
	require.NoError(t, err)
	decodedSubReq, err := e2.DecodeE2AP_PDU(buffer[:n])
	require.NoError(t, err)
	assert.NotNil(t, decodedSubReq.Value.RICsubscriptionRequest)
	assert.Equal(t, int64(2), decodedSubReq.Value.RICsubscriptionRequest.TransactionID)
}
