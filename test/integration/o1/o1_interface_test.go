package o1_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/hctsai1006/near-rt-ric/pkg/o1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/ssh"
)

// O1IntegrationTestSuite contains O1 interface integration tests
type O1IntegrationTestSuite struct {
	ctx               context.Context
	cancel            context.CancelFunc
	o1Interface       *o1.O1Interface
	postgresContainer testcontainers.Container
	dbURL             string
	netconfClient     *NetconfSSHClient
	testServerHost    string
	testServerPort    int
	suite.Suite
}

// NetconfSSHClient represents a simple NETCONF SSH client for testing
type NetconfSSHClient struct {
	conn      net.Conn
	sshClient *ssh.Client
	session   *ssh.Session
	stdin     io.WriteCloser
	stdout    io.Reader
	stderr    io.Reader
}

// SetupSuite runs before all tests
func (suite *O1IntegrationTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Setup test containers
	suite.setupPostgres()
	suite.setupO1Interface()

	// Wait for interface to be ready
	time.Sleep(5 * time.Second)

	// Setup NETCONF client
	suite.setupNetconfClient()
}

// TearDownSuite runs after all tests
func (suite *O1IntegrationTestSuite) TearDownSuite() {
	if suite.netconfClient != nil {
		suite.netconfClient.Close()
	}
	if suite.o1Interface != nil {
		suite.o1Interface.Stop(suite.ctx)
	}
	if suite.postgresContainer != nil {
		suite.postgresContainer.Terminate(suite.ctx)
	}
	suite.cancel()
}

func (suite *O1IntegrationTestSuite) setupPostgres() {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "near_rt_ric_test",
			"POSTGRES_USER":     "test_user",
			"POSTGRES_PASSWORD": "test_password",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	container, err := testcontainers.GenericContainer(suite.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(suite.T(), err)

	suite.postgresContainer = container

	host, err := container.Host(suite.ctx)
	require.NoError(suite.T(), err)

	port, err := container.MappedPort(suite.ctx, "5432")
	require.NoError(suite.T(), err)

	suite.dbURL = fmt.Sprintf("postgres://test_user:test_password@%s:%s/near_rt_ric_test?sslmode=disable", host, port.Port())
}

func (suite *O1IntegrationTestSuite) setupO1Interface() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	metrics := monitoring.NewMetricsCollector("near_rt_ric", "o1")

	// Generate test SSH keys
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(suite.T(), err)

	suite.testServerHost = "127.0.0.1"
	suite.testServerPort = 2830 // Non-standard port for testing

	cfg := &config.O1Config{
		NETCONF: config.NetconfConfig{
			Server: config.NetconfServerConfig{
				Host: suite.testServerHost,
				Port: suite.testServerPort,
				SSH: config.SSHConfig{
					Enabled:           true,
					HostKeyFile:       "", // Will be generated
					AuthorizedKeysDir: "/tmp/test-keys",
				},
				TLS: config.TLSConfig{
					Enabled:  false, // Disable TLS for testing
					CertFile: "",
					KeyFile:  "",
				},
			},
			Capabilities: []string{
				"urn:ietf:params:netconf:base:1.0",
				"urn:ietf:params:netconf:base:1.1",
				"urn:o-ran:smo:teiv:1.0",
			},
		},
		Database: config.DatabaseConfig{
			URL:                suite.dbURL,
			MaxConnections:     10,
			MaxIdleConnections: 5,
			ConnTimeout:        30 * time.Second,
		},
		YANG: config.YANGConfig{
			ModulesPath: "test/fixtures/yang-modules",
			Models: []config.YANGModel{
				{
					Name:      "ietf-interfaces",
					Namespace: "urn:ietf:params:xml:ns:yang:ietf-interfaces",
					Version:   "2018-02-20",
				},
				{
					Name:      "o-ran-smo-teiv",
					Namespace: "urn:o-ran:smo:teiv:1.0",
					Version:   "1.0.0",
				},
			},
		},
		FCAPS: config.FCAPSConfig{
			FaultManagement: config.FaultManagementConfig{
				Enabled:           true,
				AlarmBufferSize:   1000,
				HeartbeatInterval: 30 * time.Second,
			},
			ConfigurationManagement: config.ConfigurationManagementConfig{
				Enabled:        true,
				BackupInterval: 24 * time.Hour,
				MaxBackups:     7,
			},
			PerformanceManagement: config.PerformanceManagementConfig{
				Enabled:            true,
				CollectionInterval: 15 * time.Minute,
				MetricsRetention:   30 * 24 * time.Hour,
			},
		},
	}

	suite.o1Interface, err = o1.NewO1Interface(cfg, logger, metrics)
	require.NoError(suite.T(), err)

	// Start O1 interface in a goroutine so it doesn't block
	go func() {
		if err := suite.o1Interface.Start(suite.ctx); err != nil {
			suite.T().Logf("O1 interface start error: %v", err)
		}
	}()

	// Wait for the server to be ready
	suite.waitForServerReady(suite.testServerHost, suite.testServerPort)
}

func (suite *O1IntegrationTestSuite) setupNetconfClient() {
	// Generate client SSH key for testing and add it to the server's authorized keys
	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(suite.T(), err)

	signer, err := ssh.NewSignerFromKey(clientKey)
	require.NoError(suite.T(), err)

	// Add client public key to authorized keys for the server
	publicKey, err := ssh.NewPublicKey(&clientKey.PublicKey)
	require.NoError(suite.T(), err)
	authorizedKey := ssh.MarshalAuthorizedKey(publicKey)

	authKeysDir := "/tmp/test-keys"
	err = os.MkdirAll(authKeysDir, 0700)
	require.NoError(suite.T(), err)

	authKeysFile := filepath.Join(authKeysDir, "test-client.pub")
	err = os.WriteFile(authKeysFile, authorizedKey, 0644)
	require.NoError(suite.T(), err)

	// SSH client configuration
	config := &ssh.ClientConfig{
		User: "netconf",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Only for testing
		Timeout:         10 * time.Second,
	}

	// Connect to NETCONF server
	addr := fmt.Sprintf("%s:%d", suite.testServerHost, suite.testServerPort)
	client, err := ssh.Dial("tcp", addr, config)
	require.NoError(suite.T(), err)

	// Create NETCONF subsystem session
	session, err := client.NewSession()
	require.NoError(suite.T(), err)

	stdin, err := session.StdinPipe()
	require.NoError(suite.T(), err)

	stdout, err := session.StdoutPipe()
	require.NoError(suite.T(), err)

	stderr, err := session.StderrPipe()
	require.NoError(suite.T(), err)

	// Start NETCONF subsystem
	err = session.RequestSubsystem("netconf")
	require.NoError(suite.T(), err)

	suite.netconfClient = &NetconfSSHClient{
		sshClient: client,
		session:   session,
		stdin:     stdin,
		stdout:    stdout,
		stderr:    stderr,
	}

	// Wait for server hello
	suite.expectHello()
}

// Test NETCONF Hello exchange
func (suite *O1IntegrationTestSuite) TestNetconfHelloExchange() {
	// Send client hello
	clientHello := `<?xml version="1.0" encoding="UTF-8"?>
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <capabilities>
    <capability>urn:ietf:params:netconf:base:1.0</capability>
    <capability>urn:ietf:params:netconf:base:1.1</capability>
  </capabilities>
</hello>
]]>]]>`

	err := suite.sendNetconfMessage(clientHello)
	require.NoError(suite.T(), err)

	// Expect server capabilities in response
	response, err := suite.receiveNetconfMessage()
	require.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "urn:ietf:params:netconf:base:1.0")
	assert.Contains(suite.T(), response, "urn:o-ran:smo:teiv:1.0")
}

// Test NETCONF Get operation
func (suite *O1IntegrationTestSuite) TestNetconfGetOperation() {
	getRequest := `<?xml version="1.0" encoding="UTF-8"?>
<rpc message-id="1" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get>
    <filter type="xpath" select="/interfaces" xmlns:if="urn:ietf:params:xml:ns:yang:ietf-interfaces"/>
  </get>
</rpc>
]]>]]>`

	err := suite.sendNetconfMessage(getRequest)
	require.NoError(suite.T(), err)

	response, err := suite.receiveNetconfMessage()
	require.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "rpc-reply")
	assert.Contains(suite.T(), response, "message-id=\"1\"")
	// Should contain interface configuration data
	assert.Contains(suite.T(), response, "data")
}

// Test NETCONF Get-Config operation
func (suite *O1IntegrationTestSuite) TestNetconfGetConfigOperation() {
	getConfigRequest := `<?xml version="1.0" encoding="UTF-8"?>
<rpc message-id="2" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get-config>
    <source>
      <running/>
    </source>
    <filter type="subtree">
      <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces"/>
    </filter>
  </get-config>
</rpc>
]]>]]>`

	err := suite.sendNetconfMessage(getConfigRequest)
	require.NoError(suite.T(), err)

	response, err := suite.receiveNetconfMessage()
	require.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "rpc-reply")
	assert.Contains(suite.T(), response, "message-id=\"2\"")
	assert.Contains(suite.T(), response, "data")
}

// Test NETCONF Edit-Config operation
func (suite *O1IntegrationTestSuite) TestNetconfEditConfigOperation() {
	editConfigRequest := `<?xml version="1.0" encoding="UTF-8"?>
<rpc message-id="3" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target>
      <candidate/>
    </target>
    <config>
      <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">
        <interface>
          <name>eth0</name>
          <description>Test interface configuration</description>
          <enabled>true</enabled>
        </interface>
      </interfaces>
    </config>
  </edit-config>
</rpc>
]]>]]>`

	err := suite.sendNetconfMessage(editConfigRequest)
	require.NoError(suite.T(), err)

	response, err := suite.receiveNetconfMessage()
	require.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "rpc-reply")
	assert.Contains(suite.T(), response, "message-id=\"3\"")
	assert.Contains(suite.T(), response, "ok")
}

// Test NETCONF Commit operation
func (suite *O1IntegrationTestSuite) TestNetconfCommitOperation() {
	// First do an edit-config
	suite.TestNetconfEditConfigOperation()

	// Now commit the changes
	commitRequest := `<?xml version="1.0" encoding="UTF-8"?>
<rpc message-id="4" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <commit/>
</rpc>
]]>]]>`

	err := suite.sendNetconfMessage(commitRequest)
	require.NoError(suite.T(), err)

	response, err := suite.receiveNetconfMessage()
	require.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "rpc-reply")
	assert.Contains(suite.T(), response, "message-id=\"4\"")
	assert.Contains(suite.T(), response, "ok")
}

// Test O-RAN specific operations
func (suite *O1IntegrationTestSuite) TestORanSpecificOperations() {
	// Test O-RAN SMO TEIV query
	oranQuery := `<?xml version="1.0" encoding="UTF-8"?>
<rpc message-id="5" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get>
    <filter type="subtree">
      <teiv xmlns="urn:o-ran:smo:teiv:1.0">
        <entities>
          <entity>
            <entity-type>ManagedElement</entity-type>
            <entity-id>near-rt-ric-001</entity-id>
          </entity>
        </entities>
      </teiv>
    </filter>
  </get>
</rpc>
]]>]]>`

	err := suite.sendNetconfMessage(oranQuery)
	require.NoError(suite.T(), err)

	response, err := suite.receiveNetconfMessage()
	require.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "rpc-reply")
	assert.Contains(suite.T(), response, "message-id=\"5\"")
}

// Test Fault Management (FCAPS)
func (suite *O1IntegrationTestSuite) TestFaultManagement() {
	suite.T().Skip("Skipping fault management test. " +
		"This test is logically incorrect as it attempts to send a <notification> " +
		"from the client to the server. NETCONF notifications are sent from the server to the client. " +
		"A proper test would require triggering a fault on the server and listening for the notification, " +
		"which is beyond the scope of this simple client.")
}

// Test Performance Management
func (suite *O1IntegrationTestSuite) TestPerformanceManagement() {
	// Query performance metrics
	pmQuery := `<?xml version="1.0" encoding="UTF-8"?>
<rpc message-id="6" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get>
    <filter type="subtree">
      <performance-management xmlns="urn:o-ran:pm:1.0">
        <performance-measurements>
          <measurement-type>e2-interface-throughput</measurement-type>
          <measurement-period>15min</measurement-period>
        </performance-measurements>
      </performance-management>
    </filter>
  </get>
</rpc>
]]>]]>`

	err := suite.sendNetconfMessage(pmQuery)
	require.NoError(suite.T(), err)

	response, err := suite.receiveNetconfMessage()
	require.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "rpc-reply")
	assert.Contains(suite.T(), response, "message-id=\"6\"")
}

// Test invalid operations
func (suite *O1IntegrationTestSuite) TestInvalidOperations() {
	// Test malformed XML
	invalidRequest := `<?xml version="1.0" encoding="UTF-8"?>
<rpc message-id="7" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get>
    <filter type="invalid-type">
      <invalid-element>
    </filter>
  </get>
</rpc>
]]>]]>`

	err := suite.sendNetconfMessage(invalidRequest)
	require.NoError(suite.T(), err)

	response, err := suite.receiveNetconfMessage()
	require.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "rpc-reply")
	assert.Contains(suite.T(), response, "rpc-error")
	assert.Contains(suite.T(), response, "message-id=\"7\"")
}

// Test session management
func (suite *O1IntegrationTestSuite) TestSessionManagement() {
	// Test close-session
	closeSessionRequest := `<?xml version="1.0" encoding="UTF-8"?>
<rpc message-id="8" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <close-session/>
</rpc>
]]>]]>`

	err := suite.sendNetconfMessage(closeSessionRequest)
	require.NoError(suite.T(), err)

	response, err := suite.receiveNetconfMessage()
	require.NoError(suite.T(), err)

	assert.Contains(suite.T(), response, "rpc-reply")
	assert.Contains(suite.T(), response, "message-id=\"8\"")
	assert.Contains(suite.T(), response, "ok")
}

// Helper methods for NETCONF communication

func (suite *O1IntegrationTestSuite) expectHello() {
	response, err := suite.receiveNetconfMessage()
	require.NoError(suite.T(), err)
	assert.Contains(suite.T(), response, "<hello")
}

func (suite *O1IntegrationTestSuite) sendNetconfMessage(message string) error {
	_, err := suite.netconfClient.stdin.Write([]byte(message))
	return err
}

func (suite *O1IntegrationTestSuite) receiveNetconfMessage() (string, error) {
	var buf bytes.Buffer
	readBuf := make([]byte, 4096)
	delimiter := "]]>]]>"

	for {
		n, err := suite.netconfClient.stdout.Read(readBuf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("error reading from stdout: %w", err)
		}
		buf.Write(readBuf[:n])
		if strings.Contains(buf.String(), delimiter) {
			break
		}
	}

	response := buf.String()
	// Return the first full message and trim the delimiter
	message, _, _ := strings.Cut(response, delimiter)
	return message, nil
}

func (nc *NetconfSSHClient) Close() {
	if nc.stdin != nil {
		nc.stdin.Close()
	}
	if nc.session != nil {
		nc.session.Close()
	}
	if nc.sshClient != nil {
		nc.sshClient.Close()
	}
}

func (suite *O1IntegrationTestSuite) waitForServerReady(host string, port int) {
	address := fmt.Sprintf("%s:%d", host, port)
	require.Eventually(suite.T(), func() bool {
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}, 10*time.Second, 200*time.Millisecond, "NETCONF server did not become available")
}

// Test runner
func TestO1IntegrationSuite(t *testing.T) {
	suite.Run(t, new(O1IntegrationTestSuite))
}
