package o1

import (
	"github.com/hctsai1006/near-rt-ric/pkg/o1/netconf"
)

// O1Interface defines the operations for the O1 interface
type O1Interface interface {
	StartNetconfServer() error
	HandleRPCRequest(*netconf.RPCRequest) (*netconf.RPCResponse, error)
	SendNotification(*Notification) error
	ManageConfiguration(*ConfigOperation) error
	CollectPerformanceData() error
}

// Notification represents an O1 notification
type Notification struct {
	EventTime time.Time
	Message   string
}

// ConfigOperation represents a configuration operation
type ConfigOperation struct {
	Type   string
	Target string
	Data   string
}
