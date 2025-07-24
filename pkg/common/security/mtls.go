package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// MTLSConfig contains mTLS configuration
type MTLSConfig struct {
	CACertPath     string        `json:"ca_cert_path"`
	ServerCertPath string        `json:"server_cert_path"`
	ServerKeyPath  string        `json:"server_key_path"`
	ClientCertPath string        `json:"client_cert_path"`
	ClientKeyPath  string        `json:"client_key_path"`
	CertTTL        time.Duration `json:"cert_ttl"`
	EnableOCSP     bool          `json:"enable_ocsp"`
	CRLCheckURL    string        `json:"crl_check_url"`
	
	// Advanced security features
	MinTLSVersion       uint16   `json:"min_tls_version"`
	CipherSuites        []uint16 `json:"cipher_suites"`
	CurvePreferences    []tls.CurveID `json:"curve_preferences"`
	RequireClientCert   bool     `json:"require_client_cert"`
	VerifyClientCertCN  bool     `json:"verify_client_cert_cn"`
	AllowedClientCNs    []string `json:"allowed_client_cns"`
}

// MTLSManager handles mutual TLS authentication and certificate management
type MTLSManager struct {
	config     *MTLSConfig
	logger     *logrus.Logger
	caCert     *x509.Certificate
	caKey      *rsa.PrivateKey
	serverCert tls.Certificate
	clientCert tls.Certificate
}

// CertificateInfo contains certificate metadata
type CertificateInfo struct {
	CommonName    string    `json:"common_name"`
	Organization  string    `json:"organization"`
	Country       string    `json:"country"`
	ValidFrom     time.Time `json:"valid_from"`
	ValidTo       time.Time `json:"valid_to"`
	SerialNumber  string    `json:"serial_number"`
	KeyUsage      string    `json:"key_usage"`
	ExtKeyUsage   []string  `json:"ext_key_usage"`
	DNSNames      []string  `json:"dns_names"`
	IPAddresses   []net.IP  `json:"ip_addresses"`
}

// NewMTLSManager creates a new mTLS manager with enhanced security
func NewMTLSManager(config *MTLSConfig, baseLogger *logrus.Logger) (*MTLSManager, error) {
	manager := &MTLSManager{
		config: config,
		logger: baseLogger.WithField("component", "mtls-manager"),
	}
	
	// Set secure defaults
	if config.MinTLSVersion == 0 {
		config.MinTLSVersion = tls.VersionTLS13
	}
	
	if len(config.CipherSuites) == 0 {
		config.CipherSuites = []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		}
	}
	
	if len(config.CurvePreferences) == 0 {
		config.CurvePreferences = []tls.CurveID{
			tls.X25519,
			tls.CurveP384,
			tls.CurveP256,
		}
	}
	
	// Initialize certificates
	if err := manager.initializeCertificates(); err != nil {
		return nil, fmt.Errorf("failed to initialize certificates: %w", err)
	}
	
	manager.logger.Info("mTLS manager initialized with enhanced security")
	return manager, nil
}

// initializeCertificates loads or generates certificates
func (m *MTLSManager) initializeCertificates() error {
	// Load CA certificate
	if err := m.loadCACertificate(); err != nil {
		return fmt.Errorf("failed to load CA certificate: %w", err)
	}
	
	// Load server certificate
	if err := m.loadServerCertificate(); err != nil {
		return fmt.Errorf("failed to load server certificate: %w", err)
	}
	
	// Load client certificate
	if err := m.loadClientCertificate(); err != nil {
		return fmt.Errorf("failed to load client certificate: %w", err)
	}
	
	return nil
}

// loadCACertificate loads or generates CA certificate
func (m *MTLSManager) loadCACertificate() error {
	if _, err := os.Stat(m.config.CACertPath); os.IsNotExist(err) {
		m.logger.Info("CA certificate not found, generating new CA")
		return m.generateCACertificate()
	}
	
	// Load existing CA certificate
	certPEM, err := os.ReadFile(m.config.CACertPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}
	
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to parse CA certificate PEM")
	}
	
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}
	
	// Load CA private key
	keyPath := m.config.CACertPath[:len(m.config.CACertPath)-4] + "-key.pem"
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read CA private key: %w", err)
	}
	
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("failed to parse CA private key PEM")
	}
	
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA private key: %w", err)
	}
	
	m.caCert = cert
	m.caKey = key
	
	m.logger.WithFields(logrus.Fields{
		"subject":     cert.Subject.CommonName,
		"valid_from":  cert.NotBefore,
		"valid_to":    cert.NotAfter,
		"serial":      cert.SerialNumber,
	}).Info("CA certificate loaded successfully")
	
	return nil
}

// generateCACertificate generates a new CA certificate
func (m *MTLSManager) generateCACertificate() error {
	// Generate CA private key
	caKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate CA private key: %w", err)
	}
	
	// Create CA certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:       []string{"O-RAN Near-RT RIC"},
			OrganizationalUnit: []string{"Certificate Authority"},
			Country:            []string{"US"},
			Province:           []string{"CA"},
			Locality:           []string{"San Francisco"},
			CommonName:         "O-RAN RIC CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(m.config.CertTTL),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            2,
		MaxPathLenZero:        false,
	}
	
	// Generate CA certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %w", err)
	}
	
	// Parse the generated certificate
	caCert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return fmt.Errorf("failed to parse generated CA certificate: %w", err)
	}
	
	// Save CA certificate
	certOut, err := os.Create(m.config.CACertPath)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate file: %w", err)
	}
	defer certOut.Close()
	
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}
	
	// Save CA private key
	keyPath := m.config.CACertPath[:len(m.config.CACertPath)-4] + "-key.pem"
	keyOut, err := os.Create(keyPath)
	if err != nil {
		return fmt.Errorf("failed to create CA key file: %w", err)
	}
	defer keyOut.Close()
	
	keyBytes := x509.MarshalPKCS1PrivateKey(caKey)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return fmt.Errorf("failed to write CA private key: %w", err)
	}
	
	// Set file permissions
	if err := os.Chmod(keyPath, 0600); err != nil {
		return fmt.Errorf("failed to set CA key permissions: %w", err)
	}
	
	m.caCert = caCert
	m.caKey = caKey
	
	m.logger.WithFields(logrus.Fields{
		"subject":    caCert.Subject.CommonName,
		"valid_from": caCert.NotBefore,
		"valid_to":   caCert.NotAfter,
		"key_size":   4096,
	}).Info("CA certificate generated successfully")
	
	return nil
}

// loadServerCertificate loads or generates server certificate
func (m *MTLSManager) loadServerCertificate() error {
	if _, err := os.Stat(m.config.ServerCertPath); os.IsNotExist(err) {
		m.logger.Info("Server certificate not found, generating new certificate")
		return m.generateServerCertificate()
	}
	
	cert, err := tls.LoadX509KeyPair(m.config.ServerCertPath, m.config.ServerKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load server certificate: %w", err)
	}
	
	m.serverCert = cert
	
	m.logger.Info("Server certificate loaded successfully")
	return nil
}

// generateServerCertificate generates a new server certificate
func (m *MTLSManager) generateServerCertificate() error {
	// Generate server private key
	serverKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate server private key: %w", err)
	}
	
	// Create server certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization:       []string{"O-RAN Near-RT RIC"},
			OrganizationalUnit: []string{"Server"},
			Country:            []string{"US"},
			Province:           []string{"CA"},
			Locality:           []string{"San Francisco"},
			CommonName:         "oran-ric-server",
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(m.config.CertTTL),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"oran-ric-server", "localhost", "oran-nearrt-ric.svc.cluster.local"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	
	// Generate server certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, m.caCert, &serverKey.PublicKey, m.caKey)
	if err != nil {
		return fmt.Errorf("failed to create server certificate: %w", err)
	}
	
	// Save server certificate
	certOut, err := os.Create(m.config.ServerCertPath)
	if err != nil {
		return fmt.Errorf("failed to create server certificate file: %w", err)
	}
	defer certOut.Close()
	
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return fmt.Errorf("failed to write server certificate: %w", err)
	}
	
	// Save server private key
	keyOut, err := os.Create(m.config.ServerKeyPath)
	if err != nil {
		return fmt.Errorf("failed to create server key file: %w", err)
	}
	defer keyOut.Close()
	
	keyBytes := x509.MarshalPKCS1PrivateKey(serverKey)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return fmt.Errorf("failed to write server private key: %w", err)
	}
	
	// Set file permissions
	if err := os.Chmod(m.config.ServerKeyPath, 0600); err != nil {
		return fmt.Errorf("failed to set server key permissions: %w", err)
	}
	
	// Load the generated certificate
	cert, err := tls.LoadX509KeyPair(m.config.ServerCertPath, m.config.ServerKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load generated server certificate: %w", err)
	}
	
	m.serverCert = cert
	
	m.logger.Info("Server certificate generated successfully")
	return nil
}

// loadClientCertificate loads or generates client certificate
func (m *MTLSManager) loadClientCertificate() error {
	if _, err := os.Stat(m.config.ClientCertPath); os.IsNotExist(err) {
		m.logger.Info("Client certificate not found, generating new certificate")
		return m.generateClientCertificate()
	}
	
	cert, err := tls.LoadX509KeyPair(m.config.ClientCertPath, m.config.ClientKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load client certificate: %w", err)
	}
	
	m.clientCert = cert
	
	m.logger.Info("Client certificate loaded successfully")
	return nil
}

// generateClientCertificate generates a new client certificate
func (m *MTLSManager) generateClientCertificate() error {
	// Generate client private key
	clientKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate client private key: %w", err)
	}
	
	// Create client certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			Organization:       []string{"O-RAN Near-RT RIC"},
			OrganizationalUnit: []string{"Client"},
			Country:            []string{"US"},
			Province:           []string{"CA"},
			Locality:           []string{"San Francisco"},
			CommonName:         "oran-ric-client",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(m.config.CertTTL),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	
	// Generate client certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, m.caCert, &clientKey.PublicKey, m.caKey)
	if err != nil {
		return fmt.Errorf("failed to create client certificate: %w", err)
	}
	
	// Save client certificate
	certOut, err := os.Create(m.config.ClientCertPath)
	if err != nil {
		return fmt.Errorf("failed to create client certificate file: %w", err)
	}
	defer certOut.Close()
	
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return fmt.Errorf("failed to write client certificate: %w", err)
	}
	
	// Save client private key
	keyOut, err := os.Create(m.config.ClientKeyPath)
	if err != nil {
		return fmt.Errorf("failed to create client key file: %w", err)
	}
	defer keyOut.Close()
	
	keyBytes := x509.MarshalPKCS1PrivateKey(clientKey)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return fmt.Errorf("failed to write client private key: %w", err)
	}
	
	// Set file permissions
	if err := os.Chmod(m.config.ClientKeyPath, 0600); err != nil {
		return fmt.Errorf("failed to set client key permissions: %w", err)
	}
	
	// Load the generated certificate
	cert, err := tls.LoadX509KeyPair(m.config.ClientCertPath, m.config.ClientKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load generated client certificate: %w", err)
	}
	
	m.clientCert = cert
	
	m.logger.Info("Client certificate generated successfully")
	return nil
}

// GetServerTLSConfig returns TLS configuration for server
func (m *MTLSManager) GetServerTLSConfig() *tls.Config {
	// Create certificate pool with CA
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(m.caCert)
	
	clientAuth := tls.NoClientCert
	if m.config.RequireClientCert {
		clientAuth = tls.RequireAndVerifyClientCert
	}
	
	return &tls.Config{
		Certificates:             []tls.Certificate{m.serverCert},
		ClientAuth:               clientAuth,
		ClientCAs:                caCertPool,
		MinVersion:               m.config.MinTLSVersion,
		CipherSuites:             m.config.CipherSuites,
		CurvePreferences:         m.config.CurvePreferences,
		PreferServerCipherSuites: true,
		SessionTicketsDisabled:   true, // Disable for enhanced security
		Renegotiation:            tls.RenegotiateNever,
		VerifyPeerCertificate:    m.verifyPeerCertificate,
	}
}

// GetClientTLSConfig returns TLS configuration for client
func (m *MTLSManager) GetClientTLSConfig(serverName string) *tls.Config {
	// Create certificate pool with CA
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(m.caCert)
	
	return &tls.Config{
		Certificates:         []tls.Certificate{m.clientCert},
		RootCAs:              caCertPool,
		ServerName:           serverName,
		MinVersion:           m.config.MinTLSVersion,
		CipherSuites:         m.config.CipherSuites,
		CurvePreferences:     m.config.CurvePreferences,
		SessionTicketsDisabled: true,
		Renegotiation:        tls.RenegotiateNever,
	}
}

// CreateHTTPSServer creates an HTTPS server with mTLS
func (m *MTLSManager) CreateHTTPSServer(addr string, handler http.Handler) *http.Server {
	tlsConfig := m.GetServerTLSConfig()
	
	server := &http.Server{
		Addr:      addr,
		Handler:   handler,
		TLSConfig: tlsConfig,
		
		// Security timeouts
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}
	
	return server
}

// CreateHTTPSClient creates an HTTPS client with mTLS
func (m *MTLSManager) CreateHTTPSClient(serverName string) *http.Client {
	tlsConfig := m.GetClientTLSConfig(serverName)
	
	transport := &http.Transport{
		TLSClientConfig:       tlsConfig,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	
	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}
}

// CreateGRPCServerCredentials creates gRPC server credentials with mTLS
func (m *MTLSManager) CreateGRPCServerCredentials() (credentials.TransportCredentials, error) {
	tlsConfig := m.GetServerTLSConfig()
	return credentials.NewTLS(tlsConfig), nil
}

// CreateGRPCClientCredentials creates gRPC client credentials with mTLS
func (m *MTLSManager) CreateGRPCClientCredentials(serverName string) (credentials.TransportCredentials, error) {
	tlsConfig := m.GetClientTLSConfig(serverName)
	return credentials.NewTLS(tlsConfig), nil
}

// CreateGRPCServer creates a gRPC server with mTLS
func (m *MTLSManager) CreateGRPCServer() (*grpc.Server, error) {
	creds, err := m.CreateGRPCServerCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC server credentials: %w", err)
	}
	
	server := grpc.NewServer(
		grpc.Creds(creds),
		grpc.MaxRecvMsgSize(4*1024*1024), // 4MB
		grpc.MaxSendMsgSize(4*1024*1024), // 4MB
		grpc.ConnectionTimeout(30*time.Second),
	)
	
	return server, nil
}

// CreateGRPCClientConnection creates a gRPC client connection with mTLS
func (m *MTLSManager) CreateGRPCClientConnection(target, serverName string) (*grpc.ClientConn, error) {
	creds, err := m.CreateGRPCClientCredentials(serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client credentials: %w", err)
	}
	
	conn, err := grpc.Dial(target,
		grpc.WithTransportCredentials(creds),
		grpc.WithBlock(),
		grpc.WithTimeout(30*time.Second),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024),
			grpc.MaxCallSendMsgSize(4*1024*1024),
		),
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}
	
	return conn, nil
}

// verifyPeerCertificate performs additional certificate verification
func (m *MTLSManager) verifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if !m.config.VerifyClientCertCN {
		return nil
	}
	
	if len(verifiedChains) == 0 || len(verifiedChains[0]) == 0 {
		return fmt.Errorf("no verified certificate chains")
	}
	
	clientCert := verifiedChains[0][0]
	
	// Check if client CN is in allowed list
	if len(m.config.AllowedClientCNs) > 0 {
		allowed := false
		for _, allowedCN := range m.config.AllowedClientCNs {
			if clientCert.Subject.CommonName == allowedCN {
				allowed = true
				break
			}
		}
		
		if !allowed {
			m.logger.WithFields(logrus.Fields{
				"client_cn":     clientCert.Subject.CommonName,
				"allowed_cns":   m.config.AllowedClientCNs,
			}).Warn("Client certificate CN not in allowed list")
			return fmt.Errorf("client certificate CN not allowed")
		}
	}
	
	m.logger.WithFields(logrus.Fields{
		"client_cn":      clientCert.Subject.CommonName,
		"client_org":     clientCert.Subject.Organization,
		"serial_number":  clientCert.SerialNumber,
		"valid_from":     clientCert.NotBefore,
		"valid_to":       clientCert.NotAfter,
	}).Debug("Client certificate verified")
	
	return nil
}

// GetCertificateInfo returns information about certificates
func (m *MTLSManager) GetCertificateInfo() map[string]*CertificateInfo {
	info := make(map[string]*CertificateInfo)
	
	// CA certificate info
	if m.caCert != nil {
		info["ca"] = &CertificateInfo{
			CommonName:   m.caCert.Subject.CommonName,
			Organization: m.caCert.Subject.Organization[0],
			Country:      m.caCert.Subject.Country[0],
			ValidFrom:    m.caCert.NotBefore,
			ValidTo:      m.caCert.NotAfter,
			SerialNumber: m.caCert.SerialNumber.String(),
			KeyUsage:     "Certificate Signing",
		}
	}
	
	// Server certificate info
	if len(m.serverCert.Certificate) > 0 {
		cert, _ := x509.ParseCertificate(m.serverCert.Certificate[0])
		if cert != nil {
			info["server"] = &CertificateInfo{
				CommonName:   cert.Subject.CommonName,
				Organization: cert.Subject.Organization[0],
				Country:      cert.Subject.Country[0],
				ValidFrom:    cert.NotBefore,
				ValidTo:      cert.NotAfter,
				SerialNumber: cert.SerialNumber.String(),
				KeyUsage:     "Server Authentication",
				DNSNames:     cert.DNSNames,
				IPAddresses:  cert.IPAddresses,
			}
		}
	}
	
	// Client certificate info
	if len(m.clientCert.Certificate) > 0 {
		cert, _ := x509.ParseCertificate(m.clientCert.Certificate[0])
		if cert != nil {
			info["client"] = &CertificateInfo{
				CommonName:   cert.Subject.CommonName,
				Organization: cert.Subject.Organization[0],
				Country:      cert.Subject.Country[0],
				ValidFrom:    cert.NotBefore,
				ValidTo:      cert.NotAfter,
				SerialNumber: cert.SerialNumber.String(),
				KeyUsage:     "Client Authentication",
			}
		}
	}
	
	return info
}

// ValidateCertificates checks certificate validity
func (m *MTLSManager) ValidateCertificates() []error {
	var errors []error
	now := time.Now()
	
	// Check CA certificate
	if m.caCert != nil {
		if now.After(m.caCert.NotAfter) {
			errors = append(errors, fmt.Errorf("CA certificate expired on %v", m.caCert.NotAfter))
		} else if now.Add(30*24*time.Hour).After(m.caCert.NotAfter) {
			errors = append(errors, fmt.Errorf("CA certificate expires soon on %v", m.caCert.NotAfter))
		}
	}
	
	// Check server certificate
	if len(m.serverCert.Certificate) > 0 {
		cert, err := x509.ParseCertificate(m.serverCert.Certificate[0])
		if err == nil {
			if now.After(cert.NotAfter) {
				errors = append(errors, fmt.Errorf("server certificate expired on %v", cert.NotAfter))
			} else if now.Add(30*24*time.Hour).After(cert.NotAfter) {
				errors = append(errors, fmt.Errorf("server certificate expires soon on %v", cert.NotAfter))
			}
		}
	}
	
	// Check client certificate
	if len(m.clientCert.Certificate) > 0 {
		cert, err := x509.ParseCertificate(m.clientCert.Certificate[0])
		if err == nil {
			if now.After(cert.NotAfter) {
				errors = append(errors, fmt.Errorf("client certificate expired on %v", cert.NotAfter))
			} else if now.Add(30*24*time.Hour).After(cert.NotAfter) {
				errors = append(errors, fmt.Errorf("client certificate expires soon on %v", cert.NotAfter))
			}
		}
	}
	
	return errors
}

// RotateCertificates rotates all certificates
func (m *MTLSManager) RotateCertificates() error {
	m.logger.Info("Starting certificate rotation")
	
	// Generate new CA certificate
	if err := m.generateCACertificate(); err != nil {
		return fmt.Errorf("failed to rotate CA certificate: %w", err)
	}
	
	// Generate new server certificate
	if err := m.generateServerCertificate(); err != nil {
		return fmt.Errorf("failed to rotate server certificate: %w", err)
	}
	
	// Generate new client certificate
	if err := m.generateClientCertificate(); err != nil {
		return fmt.Errorf("failed to rotate client certificate: %w", err)
	}
	
	m.logger.Info("Certificate rotation completed successfully")
	return nil
}