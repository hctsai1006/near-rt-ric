package o1

import (
	"context"
)

// NetconfServerInterface defines the interface for the NETCONF server.
type NetconfServerInterface interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// FCAPSManagerInterface defines the interface for the FCAPS manager.
type FCAPSManagerInterface interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
