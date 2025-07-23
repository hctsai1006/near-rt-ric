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
	db      *pgxpool.Pool
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
func NewEnrichmentManager(config *config.A1Config, logger *logrus.Logger, metrics *monitoring.MetricsCollector, db *pgxpool.Pool) *EnrichmentManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &EnrichmentManager{
		config:  config,
		logger:  logger.WithField("component", "enrichment-manager"),
		metrics: metrics,
		db:      db,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// CreateEIJob creates a new enrichment information job
func (m *EnrichmentManager) CreateEIJob(jobType, owner string) (*EIJob, error) {
	job := &EIJob{
		ID:        uuid.New().String(),
		Type:      jobType,
		Owner:     owner,
		Status:    "RUNNING",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := m.db.Exec(m.ctx,
		"INSERT INTO ei_jobs (id, type, owner, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
		job.ID, job.Type, job.Owner, job.Status, job.CreatedAt, job.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create EI job: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"job_id":   job.ID,
		"type":     jobType,
		"owner":    owner,
	}).Info("Created EI job")

	return job, nil
}

// GetEIJob retrieves a job by its ID
func (m *EnrichmentManager) GetEIJob(id string) (*EIJob, error) {
	job := &EIJob{}
	err := m.db.QueryRow(m.ctx,
		"SELECT id, type, owner, status, created_at, updated_at FROM ei_jobs WHERE id = $1",
		id).Scan(&job.ID, &job.Type, &job.Owner, &job.Status, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("job with ID %s not found", id)
	}
	return job, nil
}

// GetAllEIJobs returns all jobs
func (m *EnrichmentManager) GetAllEIJobs() []*EIJob {
	rows, err := m.db.Query(m.ctx, "SELECT id, type, owner, status, created_at, updated_at FROM ei_jobs")
	if err != nil {
		m.logger.WithError(err).Error("Failed to get all EI jobs")
		return []*EIJob{}
	}
	defer rows.Close()

	var jobs []*EIJob
	for rows.Next() {
		job := &EIJob{}
		err := rows.Scan(&job.ID, &job.Type, &job.Owner, &job.Status, &job.CreatedAt, &job.UpdatedAt)
		if err != nil {
			m.logger.WithError(err).Error("Failed to scan EI job")
			continue
		}
		jobs = append(jobs, job)
	}
	return jobs
}

// DeleteEIJob deletes a job by its ID
func (m *EnrichmentManager) DeleteEIJob(id string) error {
	cmdTag, err := m.db.Exec(m.ctx, "DELETE FROM ei_jobs WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete EI job: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("job with ID %s not found", id)
	}
	m.logger.WithField("job_id", id).Info("Deleted EI job")
	return nil
}
