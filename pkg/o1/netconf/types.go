package netconf

import "encoding/xml"

// RPCRequest represents a NETCONF RPC request
type RPCRequest struct {
	XMLName   xml.Name `xml:"rpc"`
	MessageID string   `xml:"message-id,attr"`
	Payload   []byte   `xml:",innerxml"`
}

// RPCResponse represents a NETCONF RPC response
type RPCResponse struct {
	XMLName   xml.Name `xml:"rpc-reply"`
	MessageID string   `xml:"message-id,attr"`
	Errors    []RPCError
	Data      string `xml:",innerxml"`
}

// RPCError represents a NETCONF RPC error
type RPCError struct {
	XMLName       xml.Name `xml:"rpc-error"`
	ErrorType     string   `xml:"error-type"`
	ErrorTag      string   `xml:"error-tag"`
	ErrorSeverity string   `xml:"error-severity"`
	ErrorMessage  string   `xml:"error-message"`
}

// HelloMessage represents a NETCONF hello message
type HelloMessage struct {
	XMLName      xml.Name `xml:"hello"`
	Capabilities []string `xml:"capabilities>capability"`
	SessionID    int      `xml:"session-id,omitempty"`
}
