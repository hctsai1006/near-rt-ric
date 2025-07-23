package a1

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
)

// EnrichmentManager manages enrichment information in the Near-RT RIC
type EnrichmentManager struct {
	config  *config.A1Config
	logger  *logrus.Logger
	metrics *monitoring.MetricsCollector
	jobs    map[string]*EIJob
	mutex   sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// EIJob represents an enrichment information job
type EIJob struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Owner       string    `json:"owner"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewEnrichmentManager creates a new enrichment manager
func NewEnrichmentManager(config *config.A1Config, logger *logrus.Logger, metrics *monitoring.MetricsCollector) *EnrichmentManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &EnrichmentManager{
		config:  config,
		logger:  logger.WithField("component", "enrichment-manager"),
		metrics: metrics,
		jobs:    make(map[string]*EIJob),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// CreateEIJob creates a new enrichment information job
func (m *EnrichmentManager) CreateEIJob(jobType, owner string) (*EIJob, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	job := &EIJob{
		ID:        uuid.New().String(),
		Type:      jobType,
		Owner:     owner,
		Status:    "RUNNING",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	m.jobs[job.ID] = job
	m.logger.WithFields(logrus.Fields{
		"job_id":   job.ID,
		"type":     jobType,
		"owner":    owner,
	}).Info("Created EI job")

	return job, nil
}

// GetEIJob retrieves a job by its ID
func (m *EnrichmentManager) GetEIJob(id string) (*EIJob, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	job, ok := m.jobs[id]
	if !ok {
		return nil, fmt.Errorf("job with ID %s not found", id)
	}
	return job, nil
}

// GetAllEIJobs returns all jobs
func (m *EnrichmentManager) GetAllEIJobs() []*EIJob {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var jobs []*EIJob
	for _, job := range m.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// DeleteEIJob deletes a job by its ID
func (m *EnrichmentManager) DeleteEIJob(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.jobs[id]; !ok {
		return fmt.Errorf("job with ID %s not found", id)
	}

	delete(m.jobs, id)
	m.logger.WithField("job_id", id).Info("Deleted EI job")
	return nil
}
