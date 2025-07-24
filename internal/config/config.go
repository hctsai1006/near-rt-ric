package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// LoggingConfig holds all configuration for logging
type LoggingConfig struct {
	Level      string
	Format     string
	Output     string
	Structured bool
	File       FileLoggingConfig
	Remote     RemoteLoggingConfig
}

// FileLoggingConfig holds file logging configuration
type FileLoggingConfig struct {
	Enabled    bool
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

// RemoteLoggingConfig holds remote logging configuration
type RemoteLoggingConfig struct {
	Enabled  bool
	Endpoint string
	Protocol string
}

// XAppConfig holds all configuration for xApps
type XAppConfig struct {
	Manager    XAppManagerConfig
	Database   DatabaseConfig
	Cache      CacheConfig
	Kubernetes KubernetesConfig
	Registry   XAppRegistryConfig
	Health     HealthConfig
	LogLevel   string
}

// XAppManagerConfig holds xApp Manager configuration
type XAppManagerConfig struct {
	Host             string
	Port             int
	DeploymentEngine string
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Type  string
	Redis RedisConfig
}

// RedisConfig holds Redis cache configuration
type RedisConfig struct {
	URL         string
	MaxRetries  int
	DialTimeout int
}

// KubernetesConfig holds Kubernetes configuration
type KubernetesConfig struct {
	Namespace      string
	InCluster      bool
	ConfigPath     string
	ServiceAccount string
}

// XAppRegistryConfig holds xApp registry configuration
type XAppRegistryConfig struct {
	Type string
	Helm HelmRegistryConfig
}

// HelmRegistryConfig holds Helm registry configuration
type HelmRegistryConfig struct {
	URL      string
	Username string
	Password string
}

// HealthConfig holds health check configuration
type HealthConfig struct {
	CheckInterval    int
	FailureThreshold int
	TimeoutPerCheck  int
}

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
	StrictIPValidation bool
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
	Enabled            bool
	Port               int
	BindAddress        string
	SSH                SSHConfig
	NETCONF            NETCONFConfig
	YANG               YANGConfig
	FileManagement     FileManagementConfig
	SoftwareManagement SoftwareManagementConfig
	LogLevel           string
}

// SSHConfig holds SSH server configuration
type SSHConfig struct {
	HostKeyPath        string
	AuthorizedKeysPath string
	Timeout            int
	MaxConnections     int
}

// YANGConfig holds YANG models configuration
type YANGConfig struct {
	ModulesPath    string
	ValidateConfig bool
}

// FileManagementConfig holds file management configuration
type FileManagementConfig struct {
	Enabled     bool
	UploadPath  string
	MaxFileSize int
}

// SoftwareManagementConfig holds software management configuration
type SoftwareManagementConfig struct {
	Enabled      bool
	PackagePath  string
	InstallPath  string
	BackupPath   string
	MaxPackages  int
}

// NETCONFConfig holds NETCONF server configuration
type NETCONFConfig struct {
	ListenAddress  string
	Port           int
	TLSPort        int
	TLS            TLSConfig
	Capabilities   []string
	SessionTimeout int
	MaxSessions    int
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
}

// LoadLoggingConfig loads logging configuration from environment variables
func LoadLoggingConfig() (*LoggingConfig, error) {
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found, using environment variables")
	}

	config := &LoggingConfig{
		Level:      getEnv("LOG_LEVEL", "info"),
		Format:     getEnv("LOG_FORMAT", "json"),
		Output:     getEnv("LOG_OUTPUT", "stdout"),
		Structured: getEnvAsBool("LOG_STRUCTURED", true),
		File: FileLoggingConfig{
			Enabled:    getEnvAsBool("LOG_FILE_ENABLED", false),
			Filename:   getEnv("LOG_FILE_FILENAME", "/var/log/near-rt-ric.log"),
			MaxSize:    getEnvAsInt("LOG_FILE_MAX_SIZE_MB", 100),
			MaxBackups: getEnvAsInt("LOG_FILE_MAX_BACKUPS", 3),
			MaxAge:     getEnvAsInt("LOG_FILE_MAX_AGE_DAYS", 7),
			Compress:   getEnvAsBool("LOG_FILE_COMPRESS", false),
		},
		Remote: RemoteLoggingConfig{
			Enabled:  getEnvAsBool("LOG_REMOTE_ENABLED", false),
			Endpoint: getEnv("LOG_REMOTE_ENDPOINT", ""),
			Protocol: getEnv("LOG_REMOTE_PROTOCOL", "udp"),
		},
	}

	return config, nil
}

// LoadXAppConfig loads xApp configuration from environment variables
func LoadXAppConfig() (*XAppConfig, error) {
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found, using environment variables")
	}

	config := &XAppConfig{
		Manager: XAppManagerConfig{
			Host:             getEnv("XAPP_MANAGER_HOST", "127.0.0.1"),
			Port:             getEnvAsInt("XAPP_MANAGER_PORT", 8088),
			DeploymentEngine: getEnv("XAPP_MANAGER_DEPLOYMENT_ENGINE", "kubernetes"),
		},
		Database: DatabaseConfig{
			URL:      getEnv("XAPP_DATABASE_URL", "postgres://user:password@localhost:5432/xappdb?sslmode=disable"),
			PoolSize: getEnvAsInt("XAPP_DB_POOL_SIZE", 10),
		},
		Cache: CacheConfig{
			Type: getEnv("XAPP_CACHE_TYPE", "redis"),
			Redis: RedisConfig{
				URL:         getEnv("XAPP_REDIS_URL", "redis://localhost:6379/0"),
				MaxRetries:  getEnvAsInt("XAPP_REDIS_MAX_RETRIES", 3),
				DialTimeout: getEnvAsInt("XAPP_REDIS_DIAL_TIMEOUT_SEC", 5),
			},
		},
		Kubernetes: KubernetesConfig{
			Namespace:      getEnv("XAPP_K8S_NAMESPACE", "xapp"),
			InCluster:      getEnvAsBool("XAPP_K8S_IN_CLUSTER", false),
			ConfigPath:     getEnv("XAPP_K8S_CONFIG_PATH", ""),
			ServiceAccount: getEnv("XAPP_K8S_SERVICE_ACCOUNT", "xapp-manager"),
		},
		Registry: XAppRegistryConfig{
			Type: getEnv("XAPP_REGISTRY_TYPE", "helm"),
			Helm: HelmRegistryConfig{
				URL:      getEnv("XAPP_HELM_REGISTRY_URL", "oci://registry-1.docker.io"),
				Username: getEnv("XAPP_HELM_REGISTRY_USERNAME", ""),
				Password: getEnv("XAPP_HELM_REGISTRY_PASSWORD", ""),
			},
		},
		Health: HealthConfig{
			CheckInterval:    getEnvAsInt("XAPP_HEALTH_CHECK_INTERVAL_SEC", 30),
			FailureThreshold: getEnvAsInt("XAPP_HEALTH_FAILURE_THRESHOLD", 3),
			TimeoutPerCheck:  getEnvAsInt("XAPP_HEALTH_TIMEOUT_PER_CHECK_SEC", 10),
		},
		LogLevel: getEnv("XAPP_LOG_LEVEL", "info"),
	}

	return config, nil
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
			StrictIPValidation: getEnvAsBool("A1_AUTH_STRICT_IP_VALIDATION", false),
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
		Enabled:     getEnvAsBool("O1_ENABLED", true),
		Port:        getEnvAsInt("O1_PORT", 830),
		BindAddress: getEnv("O1_BIND_ADDRESS", "0.0.0.0"),
		LogLevel:    getEnv("O1_LOG_LEVEL", "info"),
		SSH: SSHConfig{
			HostKeyPath:        getEnv("O1_SSH_HOST_KEY_PATH", "/etc/near-rt-ric/ssh/ssh_host_rsa_key"),
			AuthorizedKeysPath: getEnv("O1_SSH_AUTHORIZED_KEYS_PATH", "/etc/near-rt-ric/ssh/authorized_keys"),
			Timeout:            getEnvAsInt("O1_SSH_TIMEOUT_SEC", 30),
			MaxConnections:     getEnvAsInt("O1_SSH_MAX_CONNECTIONS", 10),
		},
		NETCONF: NETCONFConfig{
			ListenAddress:  getEnv("O1_NETCONF_LISTEN_ADDRESS", "0.0.0.0"),
			Port:           getEnvAsInt("O1_NETCONF_PORT", 830),
			TLSPort:        getEnvAsInt("O1_NETCONF_TLS_PORT", 6513),
			SessionTimeout: getEnvAsInt("O1_NETCONF_SESSION_TIMEOUT_SEC", 600),
			MaxSessions:    getEnvAsInt("O1_NETCONF_MAX_SESSIONS", 10),
			TLS: TLSConfig{
				Enabled:  getEnvAsBool("O1_NETCONF_TLS_ENABLED", true),
				CertFile: getEnv("O1_NETCONF_TLS_CERT_FILE", "/certs/tls.crt"),
				KeyFile:  getEnv("O1_NETCONF_TLS_KEY_FILE", "/certs/tls.key"),
			},
		},
		YANG: YANGConfig{
			ModulesPath:    getEnv("O1_YANG_MODULES_PATH", "/etc/near-rt-ric/yang"),
			ValidateConfig: getEnvAsBool("O1_YANG_VALIDATE_CONFIG", true),
		},
		FileManagement: FileManagementConfig{
			Enabled:     getEnvAsBool("O1_FILE_MANAGEMENT_ENABLED", true),
			UploadPath:  getEnv("O1_FILE_MANAGEMENT_UPLOAD_PATH", "/var/lib/near-rt-ric/uploads"),
			MaxFileSize: getEnvAsInt("O1_FILE_MANAGEMENT_MAX_FILE_SIZE_MB", 100),
		},
		SoftwareManagement: SoftwareManagementConfig{
			Enabled:      getEnvAsBool("O1_SOFTWARE_MANAGEMENT_ENABLED", true),
			PackagePath:  getEnv("O1_SOFTWARE_MANAGEMENT_PACKAGE_PATH", "/var/lib/near-rt-ric/packages"),
			InstallPath:  getEnv("O1_SOFTWARE_MANAGEMENT_INSTALL_PATH", "/opt/near-rt-ric"),
			BackupPath:   getEnv("O1_SOFTWARE_MANAGEMENT_BACKUP_PATH", "/var/lib/near-rt-ric/backups"),
			MaxPackages:  getEnvAsInt("O1_SOFTWARE_MANAGEMENT_MAX_PACKAGES", 10),
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