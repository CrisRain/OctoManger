// Package tlsmgr manages TLS certificates for the API server.
// It loads the certificate and private key from the system-config store,
// and auto-generates a self-signed certificate when none is present.
package tlsmgr

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"

	"octomanger/backend/internal/service"
)

const (
	configKeyCert = "ssl.certificate"
	configKeyKey  = "ssl.private_key"
	cacheTTL      = 5 * time.Minute
)

// Manager loads and caches TLS certificates from the system config store.
type Manager struct {
	svc   service.SystemConfigService
	mu    sync.RWMutex
	cert  *tls.Certificate
	expAt time.Time
}

func New(svc service.SystemConfigService) *Manager {
	return &Manager{svc: svc}
}

// EnsureCert loads the certificate from DB. If none exists, it generates a
// self-signed certificate, stores it, and caches it.
func (m *Manager) EnsureCert(ctx context.Context) error {
	cert, err := m.loadFromDB(ctx)
	if err != nil {
		return err
	}
	if cert != nil {
		m.setCached(cert)
		return nil
	}
	// No cert stored — generate a self-signed one.
	certPEM, keyPEM, err := generateSelfSigned()
	if err != nil {
		return err
	}
	if err := m.saveToDB(ctx, certPEM, keyPEM); err != nil {
		return err
	}
	tlsCert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return err
	}
	m.setCached(&tlsCert)
	return nil
}

// GetCertificate satisfies tls.Config.GetCertificate and reloads from DB when
// the cache expires, enabling hot-reload after the user uploads a new cert.
func (m *Manager) GetCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	m.mu.RLock()
	if m.cert != nil && time.Now().Before(m.expAt) {
		c := m.cert
		m.mu.RUnlock()
		return c, nil
	}
	m.mu.RUnlock()

	cert, err := m.loadFromDB(context.Background())
	if err != nil || cert == nil {
		// Return cached cert even if stale rather than failing the handshake.
		m.mu.RLock()
		c := m.cert
		m.mu.RUnlock()
		return c, nil
	}
	m.setCached(cert)
	return cert, nil
}

// TLSConfig returns a *tls.Config that uses this manager for certificate selection.
// The initial certificate is also placed in Certificates[0] so connections
// without SNI still work.
func (m *Manager) TLSConfig() *tls.Config {
	m.mu.RLock()
	initial := m.cert
	m.mu.RUnlock()

	cfg := &tls.Config{
		GetCertificate: m.GetCertificate,
	}
	if initial != nil {
		cfg.Certificates = []tls.Certificate{*initial}
	}
	return cfg
}

func (m *Manager) setCached(c *tls.Certificate) {
	m.mu.Lock()
	m.cert = c
	m.expAt = time.Now().Add(cacheTTL)
	m.mu.Unlock()
}

func (m *Manager) loadFromDB(ctx context.Context) (*tls.Certificate, error) {
	certRaw, err := m.svc.Get(ctx, configKeyCert)
	if err != nil {
		return nil, nil // not found or error — treat as missing
	}
	keyRaw, err := m.svc.Get(ctx, configKeyKey)
	if err != nil {
		return nil, nil
	}

	var certPEM, keyPEM string
	if json.Unmarshal(certRaw, &certPEM) != nil || strings.TrimSpace(certPEM) == "" {
		return nil, nil
	}
	if json.Unmarshal(keyRaw, &keyPEM) != nil || strings.TrimSpace(keyPEM) == "" {
		return nil, nil
	}

	tlsCert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return nil, err
	}
	return &tlsCert, nil
}

func (m *Manager) saveToDB(ctx context.Context, certPEM, keyPEM string) error {
	certJSON, _ := json.Marshal(certPEM)
	if err := m.svc.Set(ctx, configKeyCert, certJSON); err != nil {
		return err
	}
	keyJSON, _ := json.Marshal(keyPEM)
	return m.svc.Set(ctx, configKeyKey, keyJSON)
}

// generateSelfSigned creates a self-signed ECDSA P-256 certificate valid for
// 10 years, including localhost / 127.0.0.1 / ::1 as SANs.
func generateSelfSigned() (certPEM, keyPEM string, err error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", err
	}

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "octomanger-self-signed"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &privKey.PublicKey, privKey)
	if err != nil {
		return "", "", err
	}

	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))

	keyDER, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return "", "", err
	}
	keyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))

	return certPEM, keyPEM, nil
}
