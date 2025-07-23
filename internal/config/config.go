package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// A1Config holds all configuration for the A1 interface
type A1Config struct {
	ListenAddress string
	ListenPort    int
	TLSEnabled    bool
	TLSCertPath   string
	TLSKeyPath    string
	Auth          AuthConfig
	Database      DatabaseConfig
	LogLevel      string
}

// AuthConfig holds authentication related configuration
type AuthConfig struct {
	Enabled       bool
	PrivateKeyPath string
	PublicKeyPath  string
	TokenExpiry   int
	Issuer        string
	Audience      string
}

// DatabaseConfig holds database related configuration
type DatabaseConfig struct {
	URL      string
	PoolSize int
}

// E2Config holds all configuration for the E2 interface
type E2Config struct {
	ListenAddress        string
	ListenPort           int
	Port                 int
	MaxNodes             int
	MaxConnections       int
	ConnectionTimeout    int
	HeartbeatInterval    int
	BufferSize           int
	SubscriptionTimeout  int
	LogLevel             string
	SCTP                 SCTPConfig
	WorkerPool           WorkerPoolConfig
}

// WorkerPoolConfig holds worker pool configuration
type WorkerPoolConfig struct {
	Size      int
	QueueSize int
}

// ASN1Config holds ASN.1 encoding configuration
type ASN1Config struct {
	EncodingType      string
	LogLevel          string
	Strict            bool
	ValidateOnDecode  bool
}

// SCTPConfig holds SCTP transport configuration
type SCTPConfig struct {
	ListenAddress     string
	ListenPort        int
	ConnectTimeout    int
	ReadTimeout       int
	WriteTimeout      int
	KeepaliveInterval int
	Streams           int
	MaxAttempts       int
	InitialRTO        int
	RTOMin            int
	MaxRTO            int
	MaxConnections    int
	BufferSize        int
	HeartbeatInterval int
	ConnectionTimeout int
	TLS               *TLSConfig
}

// O1Config holds all configuration for the O1 interface
type O1Config struct {
	NETCONF NETCONFConfig
	LogLevel string
}

// NETCONFConfig holds NETCONF server configuration
type NETCONFConfig struct {
	ListenAddress string
	Port          int
	TLSPort       int
	TLS           TLSConfig
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
}

// LoadA1Config loads A1 configuration from environment variables or a .env file
func LoadA1Config() (*A1Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found, using environment variables")
	}

	config := &A1Config{
		ListenAddress: getEnv("A1_LISTEN_ADDRESS", "0.0.0.0"),
		ListenPort:    getEnvAsInt("A1_LISTEN_PORT", 10020),
		TLSEnabled:    getEnvAsBool("A1_TLS_ENABLED", true),
		TLSCertPath:   getEnv("A1_TLS_CERT_PATH", "/certs/tls.crt"),
		TLSKeyPath:    getEnv("A1_TLS_KEY_PATH", "/certs/tls.key"),
		LogLevel:      getEnv("A1_LOG_LEVEL", "info"),
		Auth: AuthConfig{
			Enabled:       getEnvAsBool("A1_AUTH_ENABLED", true),
			PrivateKeyPath: getEnv("A1_AUTH_PRIVATE_KEY_PATH", ""),
			PublicKeyPath:  getEnv("A1_AUTH_PUBLIC_KEY_PATH", ""),
			TokenExpiry:   getEnvAsInt("A1_AUTH_TOKEN_EXPIRY_SEC", 3600),
			Issuer:        getEnv("A1_AUTH_ISSUER", "near-rt-ric"),
			Audience:      getEnv("A1_AUTH_AUDIENCE", "a1-interface"),
		},
		Database: DatabaseConfig{
			URL:      getEnv("A1_DATABASE_URL", "postgres://user:password@localhost:5432/a1db?sslmode=disable"),
			PoolSize: getEnvAsInt("A1_DB_POOL_SIZE", 10),
		},
	}

	return config, nil
}

// LoadE2Config loads E2 configuration from environment variables
func LoadE2Config() (*E2Config, error) {
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found, using environment variables")
	}

	config := &E2Config{
		ListenAddress:       getEnv("E2_LISTEN_ADDRESS", "0.0.0.0"),
		ListenPort:          getEnvAsInt("E2_LISTEN_PORT", 36421),
		Port:                getEnvAsInt("E2_PORT", 36421), 
		MaxNodes:            getEnvAsInt("E2_MAX_NODES", 100),
		MaxConnections:      getEnvAsInt("E2_MAX_CONNECTIONS", 100),
		ConnectionTimeout:   getEnvAsInt("E2_CONNECTION_TIMEOUT", 30),
		HeartbeatInterval:   getEnvAsInt("E2_HEARTBEAT_INTERVAL", 10),
		BufferSize:          getEnvAsInt("E2_BUFFER_SIZE", 65536),
		SubscriptionTimeout: getEnvAsInt("E2_SUBSCRIPTION_TIMEOUT", 300),
		LogLevel:            getEnv("E2_LOG_LEVEL", "info"),
		SCTP: SCTPConfig{
			ListenAddress:     getEnv("E2_SCTP_LISTEN_ADDRESS", "0.0.0.0"),
			ListenPort:        getEnvAsInt("E2_SCTP_LISTEN_PORT", 36421),
			ConnectTimeout:    getEnvAsInt("E2_SCTP_CONNECT_TIMEOUT", 30),
			ReadTimeout:       getEnvAsInt("E2_SCTP_READ_TIMEOUT", 30),
			WriteTimeout:      getEnvAsInt("E2_SCTP_WRITE_TIMEOUT", 30),
			KeepaliveInterval: getEnvAsInt("E2_SCTP_KEEPALIVE_INTERVAL", 10),
			Streams:           getEnvAsInt("E2_SCTP_STREAMS", 1),
			MaxAttempts:       getEnvAsInt("E2_SCTP_MAX_ATTEMPTS", 3),
			InitialRTO:        getEnvAsInt("E2_SCTP_INITIAL_RTO", 3),
			RTOMin:            getEnvAsInt("E2_SCTP_RTO_MIN", 1),
			MaxRTO:            getEnvAsInt("E2_SCTP_MAX_RTO", 60),
			MaxConnections:    getEnvAsInt("E2_SCTP_MAX_CONNECTIONS", 100),
			BufferSize:        getEnvAsInt("E2_SCTP_BUFFER_SIZE", 65536),
			HeartbeatInterval: getEnvAsInt("E2_SCTP_HEARTBEAT_INTERVAL", 10),
			ConnectionTimeout: getEnvAsInt("E2_SCTP_CONNECTION_TIMEOUT", 30),
		},
		WorkerPool: WorkerPoolConfig{
			Size:      getEnvAsInt("E2_WORKER_POOL_SIZE", 10),
			QueueSize: getEnvAsInt("E2_WORKER_POOL_QUEUE_SIZE", 1000),
		},
	}

	return config, nil
}

// LoadO1Config loads O1 configuration from environment variables
func LoadO1Config() (*O1Config, error) {
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found, using environment variables")
	}

	config := &O1Config{
		LogLevel: getEnv("O1_LOG_LEVEL", "info"),
		NETCONF: NETCONFConfig{
			ListenAddress: getEnv("O1_NETCONF_LISTEN_ADDRESS", "0.0.0.0"),
			Port:          getEnvAsInt("O1_NETCONF_PORT", 830),
			TLSPort:       getEnvAsInt("O1_NETCONF_TLS_PORT", 6513),
			TLS: TLSConfig{
				Enabled:  getEnvAsBool("O1_NETCONF_TLS_ENABLED", true),
				CertFile: getEnv("O1_NETCONF_TLS_CERT_FILE", "/certs/tls.crt"),
				KeyFile:  getEnv("O1_NETCONF_TLS_KEY_FILE", "/certs/tls.key"),
			},
		},
	}

	return config, nil
}

// Helper functions to read environment variables

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return fallback
}
