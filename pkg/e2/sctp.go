package e2

import (
	"fmt"
	"github.com/ishidawataru/sctp"
	"net"
)

type SCTPTransport struct {
	listener    *sctp.SCTPListener
	connections map[string]*sctp.SCTPConn
	streamID    uint16
}

func (s *SCTPTransport) Listen() error {
	addr, err := sctp.ResolveSCTPAddr("sctp", ":36421")
	if err != nil {
		return err
	}
	listener, err := sctp.ListenSCTP("sctp", addr)
	if err != nil {
		return err
	}
	s.listener = listener

	// Configure multi-streaming
	err = listener.SetInitMsg(sctp.InitMsg{
		NumOstreams:  10,
		MaxInstreams: 10,
		MaxAttempts:  4,
		MaxInitTimeout: 60,
	})
	return err
}

func (s *SCTPTransport) SendMessage(nodeID string, pdu []byte, streamID uint16) error {
	conn := s.connections[nodeID]
	if conn == nil {
		return fmt.Errorf("no connection for node %s", nodeID)
	}
	info := &sctp.SndRcvInfo{
		Stream: streamID,
		PPID:   0,
	}
	_, err := conn.SCTPWrite(pdu, info)
	return err
}
