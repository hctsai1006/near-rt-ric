package e2

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
)

type E2Interface struct {
	listener net.Listener
}

func NewE2Interface(addr string) (*E2Interface, error) {
	return &E2Interface{}, nil
}

func (e *E2Interface) Start() error {
	return nil
}

func (e *E2Interface) Stop() error {
	return nil
}

func (e *E2Interface) SendE2SetupRequest(nodeID string, req *models.E2SetupRequest) (*models.E2SetupResponse, error) {
	return &models.E2SetupResponse{},
		nil
}

func (e *E2Interface) CreateSubscription(nodeID string, req *models.RICSubscriptionRequest) (*models.RICSubscriptionResponse, error) {
	return &models.RICSubscriptionResponse{},
		nil
}