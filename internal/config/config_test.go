package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file for testing
	configContent := `
debug: true
log_level: "debug"
listen_address: "0.0.0.0"
http_port: 8080
https_port: 8443

database:
  host: "localhost"
  port: 5432
  name: "test_db"
  user: "test_user"
  ssl_mode: "disable"
  max_connections: 25
  connection_timeout: "30s"

redis:
  address: "localhost:6379"
  database: 0
  pool_size: 10
  timeout: "5s"

a1:
  enabled: true
  interface_version: "1.1.0"
  auth:
    enabled: true
    issuer: "test-issuer"
    audience: "test-audience"
    token_expiry: 3600
  policy:
    max_policies: 100
    policy_timeout: "30s"
  enrichment:
    max_jobs: 50
    job_timeout: "60s"

e2:
  enabled: true
  sctp:
    listen_address: "0.0.0.0"
    listen_port: 36421
    max_connections: 100
    connection_timeout: "30s"
  heartbeat_interval: "30s"
  setup_timeout: "10s"

o1:
  enabled: true
  netconf:
    listen_address: "0.0.0.0"
    port: 830
    tls_port: 6513
    max_sessions: 10
    session_timeout: "300s"
    tls:
      enabled: true
      cert_file: "/etc/certs/server.crt"
      key_file: "/etc/certs/server.key"

xapp:
  registry_url: "http://localhost:8080/api/v1/xapps"
  deployment_timeout: "60s"
  max_xapps: 50

monitoring:
  metrics:
    enabled: true
    port: 9090
    interval: "15s"
  tracing:
    enabled: true
    endpoint: "http://localhost:14268/api/traces"
    sample_rate: 0.1
  logging:
    level: "info"
    format: "json"
    output: "stdout"
`

	// Write config to temporary file
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	t.Run("Load valid config", func(t *testing.T) {
		cfg, err := LoadConfig(tmpFile.Name())
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Test basic settings
		assert.True(t, cfg.Debug)
		assert.Equal(t, "debug", cfg.LogLevel)
		assert.Equal(t, "0.0.0.0", cfg.ListenAddress)
		assert.Equal(t, 8080, cfg.HTTPPort)
		assert.Equal(t, 8443, cfg.HTTPSPort)

		// Test database config
		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, 5432, cfg.Database.Port)
		assert.Equal(t, "test_db", cfg.Database.Name)
		assert.Equal(t, "test_user", cfg.Database.User)
		assert.Equal(t, 25, cfg.Database.MaxConnections)
		assert.Equal(t, 30*time.Second, cfg.Database.ConnectionTimeout)

		// Test Redis config
		assert.Equal(t, "localhost:6379", cfg.Redis.Address)
		assert.Equal(t, 0, cfg.Redis.Database)
		assert.Equal(t, 10, cfg.Redis.PoolSize)
		assert.Equal(t, 5*time.Second, cfg.Redis.Timeout)

		// Test A1 config
		assert.True(t, cfg.A1.Enabled)
		assert.Equal(t, "1.1.0", cfg.A1.InterfaceVersion)
		assert.True(t, cfg.A1.Auth.Enabled)
		assert.Equal(t, "test-issuer", cfg.A1.Auth.Issuer)
		assert.Equal(t, "test-audience", cfg.A1.Auth.Audience)
		assert.Equal(t, 3600, cfg.A1.Auth.TokenExpiry)
		assert.Equal(t, 100, cfg.A1.Policy.MaxPolicies)
		assert.Equal(t, 50, cfg.A1.Enrichment.MaxJobs)

		// Test E2 config
		assert.True(t, cfg.E2.Enabled)
		assert.Equal(t, "0.0.0.0", cfg.E2.SCTP.ListenAddress)
		assert.Equal(t, 36421, cfg.E2.SCTP.ListenPort)
		assert.Equal(t, 100, cfg.E2.SCTP.MaxConnections)
		assert.Equal(t, 30*time.Second, cfg.E2.HeartbeatInterval)

		// Test O1 config
		assert.True(t, cfg.O1.Enabled)
		assert.Equal(t, "0.0.0.0", cfg.O1.NETCONF.ListenAddress)
		assert.Equal(t, 830, cfg.O1.NETCONF.Port)
		assert.Equal(t, 6513, cfg.O1.NETCONF.TLSPort)
		assert.True(t, cfg.O1.NETCONF.TLS.Enabled)

		// Test xApp config
		assert.Equal(t, "http://localhost:8080/api/v1/xapps", cfg.XApp.RegistryURL)
		assert.Equal(t, 50, cfg.XApp.MaxXApps)

		// Test monitoring config
		assert.True(t, cfg.Monitoring.Metrics.Enabled)
		assert.Equal(t, 9090, cfg.Monitoring.Metrics.Port)
		assert.True(t, cfg.Monitoring.Tracing.Enabled)
		assert.Equal(t, "info", cfg.Monitoring.Logging.Level)
	})

	t.Run("Load non-existent config", func(t *testing.T) {
		_, err := LoadConfig("non-existent-file.yaml")
		assert.Error(t, err)
	})
}

func TestLoadDefaultConfig(t *testing.T) {
	cfg := LoadDefaultConfig()
	require.NotNil(t, cfg)

	// Test that defaults are set
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "0.0.0.0", cfg.ListenAddress)
	assert.Equal(t, 8080, cfg.HTTPPort)
	assert.Equal(t, 8443, cfg.HTTPSPort)

	// Test database defaults
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "near_rt_ric", cfg.Database.Name)
	assert.Equal(t, "disable", cfg.Database.SSLMode)

	// Test interface defaults
	assert.True(t, cfg.A1.Enabled)
	assert.True(t, cfg.E2.Enabled)
	assert.True(t, cfg.O1.Enabled)

	// Test monitoring defaults
	assert.True(t, cfg.Monitoring.Metrics.Enabled)
	assert.Equal(t, "json", cfg.Monitoring.Logging.Format)
}

func TestConfigValidation(t *testing.T) {
	t.Run("Valid config", func(t *testing.T) {
		cfg := LoadDefaultConfig()
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("Invalid HTTP port", func(t *testing.T) {
		cfg := LoadDefaultConfig()
		cfg.HTTPPort = -1
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP port")
	})

	t.Run("Invalid HTTPS port", func(t *testing.T) {
		cfg := LoadDefaultConfig()
		cfg.HTTPSPort = 70000
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTPS port")
	})

	t.Run("Invalid database port", func(t *testing.T) {
		cfg := LoadDefaultConfig()
		cfg.Database.Port = 0
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Database port")
	})

	t.Run("Invalid log level", func(t *testing.T) {
		cfg := LoadDefaultConfig()
		cfg.LogLevel = "invalid"
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "log level")
	})

	t.Run("Invalid E2 SCTP port", func(t *testing.T) {
		cfg := LoadDefaultConfig()
		cfg.E2.SCTP.ListenPort = -1
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "E2 SCTP port")
	})

	t.Run("Invalid O1 NETCONF port", func(t *testing.T) {
		cfg := LoadDefaultConfig()
		cfg.O1.NETCONF.Port = 70000
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "O1 NETCONF port")
	})
}

func TestEnvironmentVariableOverrides(t *testing.T) {
	// Test environment variable overrides
	originalValues := map[string]string{
		"LOG_LEVEL":  os.Getenv("LOG_LEVEL"),
		"HTTP_PORT":  os.Getenv("HTTP_PORT"),
		"DB_HOST":    os.Getenv("DB_HOST"),
		"DB_PORT":    os.Getenv("DB_PORT"),
		"REDIS_ADDR": os.Getenv("REDIS_ADDR"),
	}

	// Set test environment variables
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("HTTP_PORT", "9090")
	os.Setenv("DB_HOST", "testdb")
	os.Setenv("DB_PORT", "3306")
	os.Setenv("REDIS_ADDR", "redis:6379")

	defer func() {
		// Restore original values
		for key, value := range originalValues {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	t.Run("Environment variables override defaults", func(t *testing.T) {
		cfg := LoadDefaultConfig()

		// Apply environment variable overrides
		cfg.applyEnvironmentOverrides()

		assert.Equal(t, "debug", cfg.LogLevel)
		assert.Equal(t, 9090, cfg.HTTPPort)
		assert.Equal(t, "testdb", cfg.Database.Host)
		assert.Equal(t, 3306, cfg.Database.Port)
		assert.Equal(t, "redis:6379", cfg.Redis.Address)
	})
}

func TestConfigSerialization(t *testing.T) {
	cfg := LoadDefaultConfig()

	t.Run("Config to JSON", func(t *testing.T) {
		jsonData, err := cfg.ToJSON()
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		// Verify it's valid JSON
		var result map[string]interface{}
		err = json.Unmarshal(jsonData, &result)
		assert.NoError(t, err)
	})

	t.Run("Config to YAML", func(t *testing.T) {
		yamlData, err := cfg.ToYAML()
		require.NoError(t, err)
		assert.NotEmpty(t, yamlData)
		assert.Contains(t, string(yamlData), "log_level")
		assert.Contains(t, string(yamlData), "database")
	})
}

func TestConfigMerging(t *testing.T) {
	baseConfig := LoadDefaultConfig()
	baseConfig.LogLevel = "info"
	baseConfig.HTTPPort = 8080

	overrideConfig := &Config{
		LogLevel:  "debug",
		HTTPSPort: 9443,
		Database: DatabaseConfig{
			Host: "newhost",
		},
	}

	t.Run("Merge configs", func(t *testing.T) {
		merged := baseConfig.MergeWith(overrideConfig)

		// Should have overridden values
		assert.Equal(t, "debug", merged.LogLevel)
		assert.Equal(t, 9443, merged.HTTPSPort)
		assert.Equal(t, "newhost", merged.Database.Host)

		// Should keep base values for non-overridden fields
		assert.Equal(t, 8080, merged.HTTPPort)
		assert.Equal(t, 5432, merged.Database.Port) // Default from base config
	})
}

func TestConfigHelperMethods(t *testing.T) {
	cfg := LoadDefaultConfig()

	t.Run("IsDevelopment", func(t *testing.T) {
		cfg.Debug = true
		assert.True(t, cfg.IsDevelopment())

		cfg.Debug = false
		assert.False(t, cfg.IsDevelopment())
	})

	t.Run("IsProduction", func(t *testing.T) {
		cfg.Debug = false
		assert.True(t, cfg.IsProduction())

		cfg.Debug = true
		assert.False(t, cfg.IsProduction())
	})

	t.Run("GetDatabaseURL", func(t *testing.T) {
		cfg.Database = DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Name:     "testdb",
			User:     "testuser",
			Password: "testpass",
			SSLMode:  "disable",
		}

		url := cfg.GetDatabaseURL()
		expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
		assert.Equal(t, expected, url)
	})

	t.Run("GetRedisURL", func(t *testing.T) {
		cfg.Redis = RedisConfig{
			Address:  "localhost:6379",
			Password: "redispass",
			Database: 1,
		}

		url := cfg.GetRedisURL()
		assert.Contains(t, url, "redis://:redispass@localhost:6379/1")
	})

	t.Run("GetListenAddress", func(t *testing.T) {
		cfg.ListenAddress = "0.0.0.0"
		cfg.HTTPPort = 8080

		addr := cfg.GetListenAddress()
		assert.Equal(t, "0.0.0.0:8080", addr)
	})

	t.Run("GetTLSListenAddress", func(t *testing.T) {
		cfg.ListenAddress = "0.0.0.0"
		cfg.HTTPSPort = 8443

		addr := cfg.GetTLSListenAddress()
		assert.Equal(t, "0.0.0.0:8443", addr)
	})
}

func TestInterfaceConfigurations(t *testing.T) {
	cfg := LoadDefaultConfig()

	t.Run("A1 Interface Configuration", func(t *testing.T) {
		assert.True(t, cfg.A1.Enabled)
		assert.NotEmpty(t, cfg.A1.InterfaceVersion)
		assert.True(t, cfg.A1.Auth.Enabled)
		assert.Greater(t, cfg.A1.Auth.TokenExpiry, 0)
		assert.Greater(t, cfg.A1.Policy.MaxPolicies, 0)
		assert.Greater(t, cfg.A1.Enrichment.MaxJobs, 0)
	})

	t.Run("E2 Interface Configuration", func(t *testing.T) {
		assert.True(t, cfg.E2.Enabled)
		assert.Equal(t, 36421, cfg.E2.SCTP.ListenPort)
		assert.Greater(t, cfg.E2.SCTP.MaxConnections, 0)
		assert.Greater(t, cfg.E2.HeartbeatInterval, 0)
		assert.Greater(t, cfg.E2.SetupTimeout, 0)
	})

	t.Run("O1 Interface Configuration", func(t *testing.T) {
		assert.True(t, cfg.O1.Enabled)
		assert.Equal(t, 830, cfg.O1.NETCONF.Port)
		assert.Equal(t, 6513, cfg.O1.NETCONF.TLSPort)
		assert.Greater(t, cfg.O1.NETCONF.MaxSessions, 0)
		assert.Greater(t, cfg.O1.NETCONF.SessionTimeout, 0)
	})

	t.Run("xApp Configuration", func(t *testing.T) {
		assert.NotEmpty(t, cfg.XApp.RegistryURL)
		assert.Greater(t, cfg.XApp.DeploymentTimeout, 0)
		assert.Greater(t, cfg.XApp.MaxXApps, 0)
	})
}

func TestMonitoringConfiguration(t *testing.T) {
	cfg := LoadDefaultConfig()

	t.Run("Metrics Configuration", func(t *testing.T) {
		assert.True(t, cfg.Monitoring.Metrics.Enabled)
		assert.Equal(t, 9090, cfg.Monitoring.Metrics.Port)
		assert.Greater(t, cfg.Monitoring.Metrics.Interval, 0)
	})

	t.Run("Tracing Configuration", func(t *testing.T) {
		assert.NotEmpty(t, cfg.Monitoring.Tracing.Endpoint)
		assert.GreaterOrEqual(t, cfg.Monitoring.Tracing.SampleRate, 0.0)
		assert.LessOrEqual(t, cfg.Monitoring.Tracing.SampleRate, 1.0)
	})

	t.Run("Logging Configuration", func(t *testing.T) {
		assert.Contains(t, []string{"debug", "info", "warn", "error"}, cfg.Monitoring.Logging.Level)
		assert.Contains(t, []string{"json", "text"}, cfg.Monitoring.Logging.Format)
		assert.NotEmpty(t, cfg.Monitoring.Logging.Output)
	})
}

// Benchmark tests for configuration loading
func BenchmarkLoadDefaultConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cfg := LoadDefaultConfig()
		if cfg == nil {
			b.Fatal("Failed to load default config")
		}
	}
}

func BenchmarkConfigValidation(b *testing.B) {
	cfg := LoadDefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := cfg.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConfigSerialization(b *testing.B) {
	cfg := LoadDefaultConfig()

	b.Run("ToJSON", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := cfg.ToJSON()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ToYAML", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := cfg.ToYAML()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
