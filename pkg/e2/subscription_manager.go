package e2

import (
	"time"

	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
)

// RICSubscription represents a single RIC subscription.
type RICSubscription struct {
	ID             string
	NodeID         string
	Status         SubscriptionStatus
	Request        *models.RICSubscriptionRequest
	Response       *models.RICSubscriptionResponse
	Failure        *models.RICSubscriptionFailure
	CreatedAt      time.Time
	LastIndication time.Time
}

// SubscriptionStatus represents the status of a RIC subscription.
type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "Active"
	SubscriptionStatusInactive SubscriptionStatus = "Inactive"
	SubscriptionStatusFailed   SubscriptionStatus = "Failed"
)
