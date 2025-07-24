package netconf

type Config struct {
	ListenAddress string
	Port          int
	TLSPort       int
	TLS           TLSConfig
}

type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
}