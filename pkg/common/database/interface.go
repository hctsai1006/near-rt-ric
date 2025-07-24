package database

import (
	"context"
	"time"
)

// Database interface defines the database contract
type Database interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Ping(ctx context.Context) error
	HealthCheck() error
}

// Repository interface defines basic CRUD operations
type Repository interface {
	Create(ctx context.Context, entity interface{}) error
	Update(ctx context.Context, id string, entity interface{}) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (interface{}, error)
	List(ctx context.Context, filters map[string]interface{}) ([]interface{}, error)
}

// Connection represents a database connection
type Connection struct {
	Driver   string
	Host     string
	Port     int
	Database string
	Username string
	Password string
	Timeout  time.Duration
}

// Config represents database configuration
type Config struct {
	Connection  Connection
	MaxConns    int
	MinConns    int
	MaxIdleTime time.Duration
	MaxLifetime time.Duration
	SSLMode     string
	SSLCert     string
	SSLKey      string
	SSLRootCert string
}

// MockDatabase provides a mock implementation for testing
type MockDatabase struct {
	connected bool
}

// NewMockDatabase creates a new mock database
func NewMockDatabase() Database {
	return &MockDatabase{}
}

// Connect simulates database connection
func (m *MockDatabase) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

// Disconnect simulates database disconnection
func (m *MockDatabase) Disconnect(ctx context.Context) error {
	m.connected = false
	return nil
}

// Ping simulates database ping
func (m *MockDatabase) Ping(ctx context.Context) error {
	if !m.connected {
		return &DatabaseError{
			Code:    "connection_error",
			Message: "database not connected",
		}
	}
	return nil
}

// HealthCheck simulates database health check
func (m *MockDatabase) HealthCheck() error {
	return m.Ping(context.Background())
}

// DatabaseError represents a database error
type DatabaseError struct {
	Code    string
	Message string
}

func (e *DatabaseError) Error() string {
	return e.Message
}
