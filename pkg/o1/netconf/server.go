package netconf

import (
	"encoding/xml"
	"fmt"
	"io"
	"net"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// RPCReply represents a NETCONF RPC reply
type RPCReply struct {
	XMLName   xml.Name `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 rpc-reply"`
	MessageID string   `xml:"message-id,attr"`
	Data      string   `xml:",innerxml"`
}

// RPCHandler is an interface for handling NETCONF RPCs
type RPCHandler interface {
	HandleRPC(rpc *RPCRequest) (*RPCReply, error)
}
type Server struct {
	listener   net.Listener
	logger     *logrus.Logger
	config     *ssh.ServerConfig
	rpcHandler RPCHandler
}

// NewServer creates a new NETCONF server
func NewServer(logger *logrus.Logger, rpcHandler RPCHandler, config *ssh.ServerConfig) (*Server, error) {
	return &Server{
		logger:     logger,
		config:     config,
		rpcHandler: rpcHandler,
	}, nil
}

// Start starts the NETCONF server
func (s *Server) Start(addr string) error {
	var err error
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.logger.WithField("address", addr).Info("NETCONF server listening")

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// Check if the listener was closed
			if s.listener == nil {
				return nil
			}
			return fmt.Errorf("failed to accept connection: %w", err)
		}
		go s.handleConnection(conn)
	}
}

// Stop stops the NETCONF server
func (s *Server) Stop() {
	if s.listener != nil {
		s.listener.Close()
		s.listener = nil
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	s.logger.WithField("remote_addr", conn.RemoteAddr()).Info("New NETCONF client connection")
	defer s.logger.WithField("remote_addr", conn.RemoteAddr()).Info("NETCONF client disconnected")

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		s.logger.WithError(err).Error("Failed to establish SSH connection")
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			s.logger.WithError(err).Error("Could not accept channel")
			return
		}

		go func(in <-chan *ssh.Request) {
			for req := range in {
				if req.Type == "subsystem" && string(req.Payload[4:]) == "netconf" {
					req.Reply(true, nil)
				} else {
					req.Reply(false, nil)
				}
			}
		}(requests)

		s.handleSession(channel)
	}
}

func (s *Server) handleSession(channel ssh.Channel) {
	defer channel.Close()

	// In a real implementation, you would handle hello messages and capabilities.
	// For now, we'll just read RPC requests and pass them to the handler.
	decoder := xml.NewDecoder(channel)
	for {
		var rpc RPCRequest
		err := decoder.Decode(&rpc)
		if err != nil {
			if err != io.EOF {
				s.logger.WithError(err).Error("Failed to decode RPC request")
			}
			break
		}

		reply, err := s.rpcHandler.HandleRPC(&rpc)
		if err != nil {
			s.logger.WithError(err).Error("Failed to handle RPC request")
			// In a real implementation, you would send an RPC error reply
			continue
		}

		if err := sendRPCReply(channel, reply); err != nil {
			s.logger.WithError(err).Error("Failed to send RPC reply")
		}
	}
}

func sendRPCReply(w io.Writer, reply *RPCReply) error {
	// In a real implementation, you would use xml.Marshal
	// but for now, we'll just write the raw XML.
	replyXML := fmt.Sprintf(`<rpc-reply message-id="%s" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">%s</rpc-reply>]]>]]>`, reply.MessageID, reply.Data)
	_, err := w.Write([]byte(replyXML))
	return err
}