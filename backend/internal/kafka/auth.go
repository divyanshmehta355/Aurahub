package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

type AuthConfig struct {
	User               string
	Password           string
	AuthMethod         string // "scram-sha-256", "scram-sha-512", "plain"
	ServerName         string // SNI Hostname for cloud proxies (e.g. Aiven)
	CACertPath         string
	ClientCertPath     string
	ClientKeyPath      string
	CACert             string
	ClientCert         string
	ClientKey          string
	InsecureSkipVerify bool
}

// BuildTLSAndSASL initializes TLS (including Aiven CA & client certificates) and optional SASL.
func BuildTLSAndSASL(cfg AuthConfig) (sasl.Mechanism, *tls.Config, error) {
	var tlsConfig *tls.Config

	hasCertificates := cfg.CACertPath != "" || cfg.CACert != "" ||
		(cfg.ClientCertPath != "" && cfg.ClientKeyPath != "") ||
		(cfg.ClientCert != "" && cfg.ClientKey != "")

	hasCredentials := cfg.User != "" && cfg.Password != ""

	// If neither certs nor credentials are provided, return nil (plaintext local dev)
	if !hasCertificates && !hasCredentials {
		return nil, nil, nil
	}

	tlsConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		ServerName:         cfg.ServerName,
	}

	// 1. Load CA Certificate if provided
	var caData []byte
	if cfg.CACertPath != "" {
		b, err := readCertFile(cfg.CACertPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read kafka ca cert at %s: %w", cfg.CACertPath, err)
		}
		caData = b
	} else if cfg.CACert != "" {
		caData = []byte(cfg.CACert)
	}

	if len(caData) > 0 {
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caData) {
			return nil, nil, fmt.Errorf("failed to parse ca cert pem")
		}
		tlsConfig.RootCAs = caPool
	}

	// 2. Load Client Certificate & Key (for Aiven mTLS)
	var certData, keyData []byte
	if cfg.ClientCertPath != "" && cfg.ClientKeyPath != "" {
		c, err := readCertFile(cfg.ClientCertPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read kafka client cert at %s: %w", cfg.ClientCertPath, err)
		}
		k, err := readCertFile(cfg.ClientKeyPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read kafka client key at %s: %w", cfg.ClientKeyPath, err)
		}
		certData, keyData = c, k
	} else if cfg.ClientCert != "" && cfg.ClientKey != "" {
		certData = []byte(cfg.ClientCert)
		keyData = []byte(cfg.ClientKey)
	}

	if len(certData) > 0 && len(keyData) > 0 {
		cert, err := tls.X509KeyPair(certData, keyData)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load kafka client x509 key pair: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// 3. Build SASL mechanism if username and password are provided
	var mechanism sasl.Mechanism
	if hasCredentials {
		method := strings.ToLower(strings.TrimSpace(cfg.AuthMethod))
		switch method {
		case "scram-sha-512":
			m, err := scram.Mechanism(scram.SHA512, cfg.User, cfg.Password)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create scram-sha-512 mechanism: %w", err)
			}
			mechanism = m
		case "plain":
			mechanism = plain.Mechanism{
				Username: cfg.User,
				Password: cfg.Password,
			}
		case "scram-sha-256", "":
			m, err := scram.Mechanism(scram.SHA256, cfg.User, cfg.Password)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create scram-sha-256 mechanism: %w", err)
			}
			mechanism = m
		default:
			return nil, nil, fmt.Errorf("unsupported kafka auth method: %s", cfg.AuthMethod)
		}
	}

	return mechanism, tlsConfig, nil
}

func readCertFile(path string) ([]byte, error) {
	if data, err := os.ReadFile(path); err == nil {
		return data, nil
	}
	backendPath := filepath.Join("backend", path)
	if data, err := os.ReadFile(backendPath); err == nil {
		return data, nil
	}
	return nil, fmt.Errorf("certificate file not found at %s or %s", path, backendPath)
}
