package config

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the complete configuration for the Near-RT RIC
type Config struct {
	// Server configuration
	Server ServerConfig `mapstructure:"server"`
	
	// O-RAN interface configurations
	E2 E2Config `mapstructure:"e2"`
	A1 A1Config `mapstructure:"a1"`
	O1 O1Config `mapstructure:"o1"`
	
	// xApp framework configuration
	XApp XAppConfig `mapstructure:"xapp"`
	
	// Database configurations
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	
	// Security configuration
	Security SecurityConfig `mapstructure:"security"`
	
	// Logging and monitoring
	Logging    LoggingConfig    `mapstructure:"logging"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
	
	// Development settings
	Development DevelopmentConfig `mapstructure:"development"`
}

// ServerConfig contains general server settings
type ServerConfig struct {
	Name        string        `mapstructure:"name"`
	Version     string        `mapstructure:"version"`
	Environment string        `mapstructure:"environment"`
	Host        string        `mapstructure:"host"`
	Port        int           `mapstructure:"port"`
	Timeout     time.Duration `mapstructure:"timeout"`
	
	// TLS configuration
	TLS TLSConfig `mapstructure:"tls"`
	
	// Graceful shutdown
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// E2Config contains E2 interface specific configuration
type E2Config struct {
	Enabled             bool          `mapstructure:"enabled"`
	ListenAddress       string        `mapstructure:"listen_address"`
	Port                int           `mapstructure:"port"`
	MaxConnections      int           `mapstructure:"max_connections"`
	ConnectionTimeout   time.Duration `mapstructure:"connection_timeout"`
	HeartbeatInterval   time.Duration `mapstructure:"heartbeat_interval"`
	BufferSize          int           `mapstructure:"buffer_size"`
	WorkerPoolSize      int           `mapstructure:"worker_pool_size"`
	SubscriptionTimeout time.Duration `mapstructure:"subscription_timeout"`
	
	// SCTP specific configuration
	SCTP SCTPConfig `mapstructure:"sctp"`
	
	// ASN.1 configuration
	ASN1 ASN1Config `mapstructure:"asn1"`
	
	// Worker Pool configuration
	WorkerPool WorkerPoolConfig `mapstructure:"worker_pool"`
	
	// E2 Service Model configuration
	ServiceModels []ServiceModelConfig `mapstructure:"service_models"`
}

// A1Config contains A1 interface specific configuration
type A1Config struct {
	Enabled             bool          `mapstructure:"enabled"`
	Host                string        `mapstructure:"host"`
	Port                int           `mapstructure:"port"`
	BasePath            string        `mapstructure:"base_path"`
	RequestTimeout      time.Duration `mapstructure:"request_timeout"`
	MaxRequestSize      int64         `mapstructure:"max_request_size"`
	
	// Authentication configuration
	Auth AuthConfig `mapstructure:"auth"`
	
	// Policy management
	PolicyEngine PolicyEngineConfig `mapstructure:"policy_engine"`
	
	// ML Model management
	MLModel MLModelConfig `mapstructure:"ml_model"`
}

// O1Config contains O1 interface specific configuration
type O1Config struct {
	Enabled       bool   `mapstructure:"enabled"`
	Host          string `mapstructure:"host"`
	Port          int    `mapstructure:"port"`
	Username      string `mapstructure:"username"`
	
	// NETCONF configuration
	NETCONF NETCONFConfig `mapstructure:"netconf"`
	
	// YANG models configuration
	YANG YANGConfig `mapstructure:"yang"`
	
	// FCAPS configuration
	FCAPS FCAPSConfig `mapstructure:"fcaps"`
}

// XAppConfig contains xApp framework configuration
type XAppConfig struct {
	Enabled            bool          `mapstructure:"enabled"`
	RegistryURL        string        `mapstructure:"registry_url"`
	MaxXApps           int           `mapstructure:"max_xapps"`
	DefaultTimeout     time.Duration `mapstructure:"default_timeout"`
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`
	
	// xApp deployment configuration
	Deployment XAppDeploymentConfig `mapstructure:"deployment"`
	
	// Conflict management
	ConflictResolution ConflictResolutionConfig `mapstructure:"conflict_resolution"`
}

// SecurityConfig contains all security-related settings
type SecurityConfig struct {
	// TLS settings
	TLS TLSConfig `mapstructure:"tls"`
	
	// JWT settings
	JWT JWTConfig `mapstructure:"jwt"`
	
	// RBAC settings
	RBAC RBACConfig `mapstructure:"rbac"`
	
	// Certificate management
	Certificates CertificateConfig `mapstructure:"certificates"`
	
	// API Security
	API APISecurityConfig `mapstructure:"api"`
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Database        string        `mapstructure:"database"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_connections"`
	MaxIdleConns    int           `mapstructure:"max_idle_connections"`
	ConnMaxLifetime time.Duration `mapstructure:"connection_max_lifetime"`
	
	// Migration settings
	Migration MigrationConfig `mapstructure:"migration"`
}

// RedisConfig contains Redis connection settings
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	Database int    `mapstructure:"database"`
	
	// Pool settings
	PoolSize     int           `mapstructure:"pool_size"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	Structured bool   `mapstructure:"structured"`
	
	// File logging
	File FileLoggingConfig `mapstructure:"file"`
	
	// Remote logging
	Remote RemoteLoggingConfig `mapstructure:"remote"`
}

// MonitoringConfig contains monitoring and metrics configuration
type MonitoringConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	Path    string `mapstructure:"path"`
	
	// Prometheus configuration
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
	
	// OpenTelemetry configuration
	OpenTelemetry OpenTelemetryConfig `mapstructure:"opentelemetry"`
	
	// Health checks
	HealthCheck HealthCheckConfig `mapstructure:"health_check"`
}

// TLSConfig contains TLS/SSL configuration
type TLSConfig struct {
	Enabled            bool     `mapstructure:"enabled"`
	CertFile           string   `mapstructure:"cert_file"`
	KeyFile            string   `mapstructure:"key_file"`
	CAFile             string   `mapstructure:"ca_file"`
	InsecureSkipVerify bool     `mapstructure:"insecure_skip_verify"`
	MinVersion         string   `mapstructure:"min_version"`
	CipherSuites       []string `mapstructure:"cipher_suites"`
	
	// Client certificate verification
	ClientAuth string `mapstructure:"client_auth"`
}

// SCTPConfig contains SCTP-specific configuration
type SCTPConfig struct {
	Streams           int           `mapstructure:"streams"`
	MaxAttempts       int           `mapstructure:"max_attempts"`
	InitialRTO        time.Duration `mapstructure:"initial_rto"`
	MaxRTO            time.Duration `mapstructure:"max_rto"`
	RTOMin            time.Duration `mapstructure:"rto_min"`
	HeartbeatEnabled  bool          `mapstructure:"heartbeat_enabled"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
}

// ASN1Config contains ASN.1 encoding configuration
type ASN1Config struct {
	Strict          bool `mapstructure:"strict"`
	ValidateOnDecode bool `mapstructure:"validate_on_decode"`
}

// WorkerPoolConfig contains worker pool configuration
type WorkerPoolConfig struct {
	Size      int `mapstructure:"size"`
	QueueSize int `mapstructure:"queue_size"`
}

// ServiceModelConfig contains E2 Service Model configuration
type ServiceModelConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	OID         string `mapstructure:"oid"`
	Description string `mapstructure:"description"`
	Enabled     bool   `mapstructure:"enabled"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Type     string `mapstructure:"type"`
	Provider string `mapstructure:"provider"`
	
	// OAuth2/OIDC settings
	OAuth2 OAuth2Config `mapstructure:"oauth2"`
}

// JWTConfig contains JWT-specific configuration
type JWTConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	SigningMethod string        `mapstructure:"signing_method"`
	SecretKey     string        `mapstructure:"secret_key"`
	PublicKeyFile string        `mapstructure:"public_key_file"`
	Issuer        string        `mapstructure:"issuer"`
	Audience      string        `mapstructure:"audience"`
	ExpiryTime    time.Duration `mapstructure:"expiry_time"`
}

// DevelopmentConfig contains development-specific settings
type DevelopmentConfig struct {
	EnableProfiling bool `mapstructure:"enable_profiling"`
	EnableDebugAPI  bool `mapstructure:"enable_debug_api"`
	MockE2Nodes     bool `mapstructure:"mock_e2_nodes"`
	MockXApps       bool `mapstructure:"mock_xapps"`
}

// Additional sub-configs for complex structures
type PolicyEngineConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	CacheSize     int           `mapstructure:"cache_size"`
	CacheTTL      time.Duration `mapstructure:"cache_ttl"`
	MaxPolicies   int           `mapstructure:"max_policies"`
}

type MLModelConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	StoragePath  string `mapstructure:"storage_path"`
	MaxModelSize int64  `mapstructure:"max_model_size"`
	MaxModels    int    `mapstructure:"max_models"`
}

type NETCONFConfig struct {
	Port            int           `mapstructure:"port"`
	CallHome        bool          `mapstructure:"call_home"`
	SessionTimeout  time.Duration `mapstructure:"session_timeout"`
	HelloTimeout    time.Duration `mapstructure:"hello_timeout"`
}

type YANGConfig struct {
	ModulePath   string   `mapstructure:"module_path"`
	Modules      []string `mapstructure:"modules"`
	ValidationEnabled bool `mapstructure:"validation_enabled"`
}

type FCAPSConfig struct {
	FaultManagement        bool `mapstructure:"fault_management"`
	ConfigurationManagement bool `mapstructure:"configuration_management"`
	AccountingManagement   bool `mapstructure:"accounting_management"`
	PerformanceManagement  bool `mapstructure:"performance_management"`
	SecurityManagement     bool `mapstructure:"security_management"`
}

type XAppDeploymentConfig struct {
	Namespace      string            `mapstructure:"namespace"`
	Registry       string            `mapstructure:"registry"`
	ImagePullPolicy string           `mapstructure:"image_pull_policy"`
	Resources      ResourceLimits    `mapstructure:"resources"`
	Labels         map[string]string `mapstructure:"labels"`
}

type ConflictResolutionConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Strategy  string `mapstructure:"strategy"`
	Priority  string `mapstructure:"priority"`
}

type ResourceLimits struct {
	CPU    string `mapstructure:"cpu"`
	Memory string `mapstructure:"memory"`
}

type RBACConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	PolicyFile  string `mapstructure:"policy_file"`
	DefaultRole string `mapstructure:"default_role"`
}

type CertificateConfig struct {
	AutoRotation bool          `mapstructure:"auto_rotation"`
	RotationInterval time.Duration `mapstructure:"rotation_interval"`
	CertificateDir string        `mapstructure:"certificate_dir"`
}

type APISecurityConfig struct {
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	CORS      CORSConfig      `mapstructure:"cors"`
}

type RateLimitConfig struct {
	Enabled    bool          `mapstructure:"enabled"`
	RequestsPerSecond int    `mapstructure:"requests_per_second"`
	BurstSize  int           `mapstructure:"burst_size"`
	TTL        time.Duration `mapstructure:"ttl"`
}

type CORSConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
}

type MigrationConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Directory string `mapstructure:"directory"`
	Table     string `mapstructure:"table"`
}

type FileLoggingConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
	Compress   bool   `mapstructure:"compress"`
}

type RemoteLoggingConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Endpoint string `mapstructure:"endpoint"`
	Protocol string `mapstructure:"protocol"`
}

type PrometheusConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Namespace string `mapstructure:"namespace"`
	Subsystem string `mapstructure:"subsystem"`
}

type OpenTelemetryConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	Endpoint     string `mapstructure:"endpoint"`
	ServiceName  string `mapstructure:"service_name"`
	SampleRate   float64 `mapstructure:"sample_rate"`
}

type HealthCheckConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Interval time.Duration `mapstructure:"interval"`
	Timeout  time.Duration `mapstructure:"timeout"`
	Path     string        `mapstructure:"path"`
}

type OAuth2Config struct {
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	AuthURL      string   `mapstructure:"auth_url"`
	TokenURL     string   `mapstructure:"token_url"`
	Scopes       []string `mapstructure:"scopes"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configFile string) (*Config, error) {
	config := &Config{}
	
	// Set default values
	setDefaults(config)
	
	// Configure viper
	viper.SetConfigType("yaml")
	
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath("/etc/near-rt-ric/")
		viper.AddConfigPath("$HOME/.near-rt-ric/")
		viper.AddConfigPath("./config/")
		viper.AddConfigPath(".")
	}
	
	// Environment variable support
	viper.SetEnvPrefix("RIC")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	
	// Read configuration file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Continue with defaults and environment variables if config file not found
	}
	
	// Unmarshal into config struct
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	// Resolve sensitive values from environment
	if err := resolveSensitiveValues(config); err != nil {
		return nil, fmt.Errorf("failed to resolve sensitive values: %w", err)
	}
	
	return config, nil
}

// setDefaults sets default configuration values
func setDefaults(config *Config) {
	// Server defaults
	config.Server.Name = "O-RAN Near-RT RIC"
	config.Server.Version = "1.0.0"
	config.Server.Environment = "production"
	config.Server.Host = "0.0.0.0"
	config.Server.Port = 8080
	config.Server.Timeout = 30 * time.Second
	config.Server.ShutdownTimeout = 30 * time.Second
	
	// E2 interface defaults
	config.E2.Enabled = true
	config.E2.ListenAddress = "0.0.0.0"
	config.E2.Port = 36421 // Standard E2 port
	config.E2.MaxConnections = 100
	config.E2.ConnectionTimeout = 30 * time.Second
	config.E2.HeartbeatInterval = 30 * time.Second
	config.E2.BufferSize = 65536
	config.E2.WorkerPoolSize = 10
	
	// A1 interface defaults
	config.A1.Enabled = true
	config.A1.Host = "0.0.0.0"
	config.A1.Port = 10020 // Standard A1 port
	config.A1.BasePath = "/a1-p"
	config.A1.RequestTimeout = 30 * time.Second
	config.A1.MaxRequestSize = 10 * 1024 * 1024 // 10MB
	
	// O1 interface defaults
	config.O1.Enabled = true
	config.O1.Host = "0.0.0.0"
	config.O1.Port = 830 // Standard NETCONF port
	config.O1.Username = "admin"
	
	// xApp defaults
	config.XApp.Enabled = true
	config.XApp.MaxXApps = 50
	config.XApp.DefaultTimeout = 60 * time.Second
	config.XApp.HealthCheckInterval = 30 * time.Second
	
	// Database defaults
	config.Database.Driver = "postgres"
	config.Database.Host = "localhost"
	config.Database.Port = 5432
	config.Database.Database = "near_rt_ric"
	config.Database.Username = "ric_user"
	config.Database.SSLMode = "require"
	config.Database.MaxOpenConns = 25
	config.Database.MaxIdleConns = 5
	config.Database.ConnMaxLifetime = 5 * time.Minute
	
	// Redis defaults
	config.Redis.Host = "localhost"
	config.Redis.Port = 6379
	config.Redis.Database = 0
	config.Redis.PoolSize = 10
	config.Redis.DialTimeout = 5 * time.Second
	config.Redis.ReadTimeout = 3 * time.Second
	config.Redis.WriteTimeout = 3 * time.Second
	
	// Security defaults
	config.Security.TLS.Enabled = true
	config.Security.TLS.MinVersion = "1.2"
	config.Security.JWT.Enabled = true
	config.Security.JWT.SigningMethod = "RS256"
	config.Security.JWT.ExpiryTime = 1 * time.Hour
	
	// Logging defaults
	config.Logging.Level = "info"
	config.Logging.Format = "json"
	config.Logging.Output = "stdout"
	config.Logging.Structured = true
	
	// Monitoring defaults
	config.Monitoring.Enabled = true
	config.Monitoring.Host = "0.0.0.0"
	config.Monitoring.Port = 9090
	config.Monitoring.Path = "/metrics"
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate port ranges
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}
	
	if config.E2.Port < 1 || config.E2.Port > 65535 {
		return fmt.Errorf("invalid E2 port: %d", config.E2.Port)
	}
	
	if config.A1.Port < 1 || config.A1.Port > 65535 {
		return fmt.Errorf("invalid A1 port: %d", config.A1.Port)
	}
	
	if config.O1.Port < 1 || config.O1.Port > 65535 {
		return fmt.Errorf("invalid O1 port: %d", config.O1.Port)
	}
	
	// Validate TLS configuration
	if config.Security.TLS.Enabled {
		if config.Security.TLS.CertFile == "" {
			return fmt.Errorf("TLS cert file is required when TLS is enabled")
		}
		if config.Security.TLS.KeyFile == "" {
			return fmt.Errorf("TLS key file is required when TLS is enabled")
		}
	}
	
	// Validate JWT configuration
	if config.Security.JWT.Enabled {
		if config.Security.JWT.SigningMethod == "" {
			return fmt.Errorf("JWT signing method is required when JWT is enabled")
		}
	}
	
	// Validate database configuration
	if config.Database.Driver == "" {
		return fmt.Errorf("database driver is required")
	}
	
	return nil
}

// resolveSensitiveValues resolves sensitive configuration values from environment variables
func resolveSensitiveValues(config *Config) error {
	// Database password
	if config.Database.Password == "" {
		config.Database.Password = os.Getenv("RIC_DATABASE_PASSWORD")
	}
	
	// Redis password
	if config.Redis.Password == "" {
		config.Redis.Password = os.Getenv("RIC_REDIS_PASSWORD")
	}
	
	// JWT secret key
	if config.Security.JWT.SecretKey == "" {
		config.Security.JWT.SecretKey = os.Getenv("RIC_JWT_SECRET_KEY")
	}
	
	// OAuth2 client secret
	if config.A1.Auth.OAuth2.ClientSecret == "" {
		config.A1.Auth.OAuth2.ClientSecret = os.Getenv("RIC_OAUTH2_CLIENT_SECRET")
	}
	
	// O1 password (if not using key-based auth)
	if password := os.Getenv("RIC_O1_PASSWORD"); password != "" {
		// Store in a secure way - this is just for demo
		viper.Set("o1.password", password)
	}
	
	return nil
}

// GetTLSConfig converts TLS configuration to *tls.Config
func (t *TLSConfig) GetTLSConfig() (*tls.Config, error) {
	if !t.Enabled {
		return nil, nil
	}
	
	tlsConfig := &tls.Config{
		InsecureSkipVerify: t.InsecureSkipVerify,
		PreferServerCipherSuites: true,
	}
	
	// Set minimum TLS version
	switch t.MinVersion {
	case "1.0":
		tlsConfig.MinVersion = tls.VersionTLS10
	case "1.1":
		tlsConfig.MinVersion = tls.VersionTLS11
	case "1.2":
		tlsConfig.MinVersion = tls.VersionTLS12
	case "1.3":
		tlsConfig.MinVersion = tls.VersionTLS13
	default:
		tlsConfig.MinVersion = tls.VersionTLS12
	}
	
	// Set cipher suites for TLS 1.2 and below
	if len(t.CipherSuites) > 0 {
		tlsConfig.CipherSuites = make([]uint16, 0, len(t.CipherSuites))
		for _, suite := range t.CipherSuites {
			switch suite {
			case "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":
				tlsConfig.CipherSuites = append(tlsConfig.CipherSuites, tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384)
			case "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":
				tlsConfig.CipherSuites = append(tlsConfig.CipherSuites, tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305)
			case "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":
				tlsConfig.CipherSuites = append(tlsConfig.CipherSuites, tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384)
			case "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305":
				tlsConfig.CipherSuites = append(tlsConfig.CipherSuites, tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305)
			}
		}
	} else {
		// Use secure defaults
		tlsConfig.CipherSuites = []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		}
	}
	
	return tlsConfig, nil
}

// GetDatabaseConnectionString builds database connection string
func (d *DatabaseConfig) GetConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.Username, d.Password, d.Database, d.SSLMode)
}

// GetRedisConnectionString builds Redis connection string
func (r *RedisConfig) GetConnectionString() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}