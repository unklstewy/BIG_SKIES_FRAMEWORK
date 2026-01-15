// Package security provides security engine implementations.
package security

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/models"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/healthcheck"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// TLSSecurityEngine manages TLS/SSL certificates including Let's Encrypt integration.
type TLSSecurityEngine struct {
	db              *pgxpool.Pool
	logger          *zap.Logger
	certCache       map[string]*models.TLSCertificate // keyed by domain
	mu              sync.RWMutex
	acmeClient      *acme.Client
	acmeManager     *autocert.Manager
	renewalInterval time.Duration
	stopChan        chan struct{}
}

// TLSConfig holds configuration for the TLS engine.
type TLSConfig struct {
	ACMEDirectoryURL string   // Let's Encrypt production or staging URL
	Email            string   // Contact email for ACME notifications
	CacheDir         string   // Directory for autocert cache
	Domains          []string // Allowed domains for autocert
}

// NewTLSSecurityEngine creates a new TLS security engine.
func NewTLSSecurityEngine(db *pgxpool.Pool, config *TLSConfig, logger *zap.Logger) *TLSSecurityEngine {
	if logger == nil {
		logger = zap.NewNop()
	}

	engine := &TLSSecurityEngine{
		db:              db,
		logger:          logger.With(zap.String("engine", "tls_security")),
		certCache:       make(map[string]*models.TLSCertificate),
		renewalInterval: 24 * time.Hour,
		stopChan:        make(chan struct{}),
	}

	// Initialize ACME client if config provided
	if config != nil && config.ACMEDirectoryURL != "" {
		engine.acmeManager = &autocert.Manager{
			Prompt:      autocert.AcceptTOS,
			Email:       config.Email,
			Cache:       autocert.DirCache(config.CacheDir),
			HostPolicy:  autocert.HostWhitelist(config.Domains...),
		}

		engine.acmeClient = &acme.Client{
			DirectoryURL: config.ACMEDirectoryURL,
		}

		engine.logger.Info("Initialized ACME client",
			zap.String("directory", config.ACMEDirectoryURL),
			zap.String("email", config.Email))
	}

	return engine
}

// Start begins the certificate renewal monitoring.
func (e *TLSSecurityEngine) Start(ctx context.Context) {
	go e.renewalMonitor(ctx)
	e.logger.Info("TLS security engine started")
}

// Stop stops the certificate renewal monitoring.
func (e *TLSSecurityEngine) Stop() {
	close(e.stopChan)
	e.logger.Info("TLS security engine stopped")
}

// renewalMonitor periodically checks for expiring certificates.
func (e *TLSSecurityEngine) renewalMonitor(ctx context.Context) {
	ticker := time.NewTicker(e.renewalInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.checkExpiringCertificates(ctx)
		}
	}
}

// checkExpiringCertificates checks for certificates expiring within 30 days.
func (e *TLSSecurityEngine) checkExpiringCertificates(ctx context.Context) {
	query := `
		SELECT id, domain, certificate_pem, private_key_pem, expires_at, issuer, created_at, updated_at
		FROM tls_certificates
		WHERE expires_at < $1
	`

	expiryThreshold := time.Now().Add(30 * 24 * time.Hour)
	rows, err := e.db.Query(ctx, query, expiryThreshold)
	if err != nil {
		e.logger.Error("Failed to query expiring certificates", zap.Error(err))
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cert models.TLSCertificate
		err := rows.Scan(&cert.ID, &cert.Domain, &cert.CertificatePEM, &cert.PrivateKeyPEM,
			&cert.ExpiresAt, &cert.Issuer, &cert.CreatedAt, &cert.UpdatedAt)
		if err != nil {
			e.logger.Error("Failed to scan certificate", zap.Error(err))
			continue
		}

		e.logger.Warn("Certificate expiring soon",
			zap.String("domain", cert.Domain),
			zap.Time("expires_at", cert.ExpiresAt))

		// Attempt renewal for Let's Encrypt certificates
		if cert.Issuer == "letsencrypt" {
			e.logger.Info("Attempting to renew certificate", zap.String("domain", cert.Domain))
			// Renewal would be handled by autocert manager automatically
		}
	}
}

// GenerateSelfSignedCertificate creates a self-signed certificate for development.
func (e *TLSSecurityEngine) GenerateSelfSignedCertificate(ctx context.Context, domain string, validDays int) (*models.TLSCertificate, error) {
	// Generate private key
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(time.Duration(validDays) * 24 * time.Hour)

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"BIG SKIES Framework"},
			CommonName:   domain,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{domain},
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Encode private key to PEM
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})

	cert := &models.TLSCertificate{
		ID:             uuid.New().String(),
		Domain:         domain,
		CertificatePEM: string(certPEM),
		PrivateKeyPEM:  string(keyPEM),
		ExpiresAt:      notAfter,
		Issuer:         "self-signed",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Store in database
	if err := e.StoreCertificate(ctx, cert); err != nil {
		return nil, fmt.Errorf("failed to store certificate: %w", err)
	}

	e.logger.Info("Generated self-signed certificate",
		zap.String("domain", domain),
		zap.Time("expires_at", notAfter))

	return cert, nil
}

// RequestLetsEncryptCertificate requests a certificate from Let's Encrypt.
func (e *TLSSecurityEngine) RequestLetsEncryptCertificate(ctx context.Context, domain string) (*models.TLSCertificate, error) {
	if e.acmeManager == nil {
		return nil, fmt.Errorf("ACME manager not configured")
	}

	// Use autocert to get certificate
	hello := &tls.ClientHelloInfo{
		ServerName: domain,
	}

	cert, err := e.acmeManager.GetCertificate(hello)
	if err != nil {
		e.logger.Error("Failed to get Let's Encrypt certificate",
			zap.Error(err),
			zap.String("domain", domain))
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}

	// Parse certificate
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	
	// Marshal private key (ECDSA or RSA)
	var keyPEM []byte
	if privKey, ok := cert.PrivateKey.(*ecdsa.PrivateKey); ok {
		keyBytes, err := x509.MarshalECPrivateKey(privKey)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal ECDSA private key: %w", err)
		}
		keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	} else {
		return nil, fmt.Errorf("unsupported private key type")
	}

	tlsCert := &models.TLSCertificate{
		ID:             uuid.New().String(),
		Domain:         domain,
		CertificatePEM: string(certPEM),
		PrivateKeyPEM:  string(keyPEM),
		ExpiresAt:      x509Cert.NotAfter,
		Issuer:         "letsencrypt",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Store in database
	if err := e.StoreCertificate(ctx, tlsCert); err != nil {
		return nil, fmt.Errorf("failed to store certificate: %w", err)
	}

	e.logger.Info("Obtained Let's Encrypt certificate",
		zap.String("domain", domain),
		zap.Time("expires_at", x509Cert.NotAfter))

	return tlsCert, nil
}

// StoreCertificate stores a certificate in the database.
func (e *TLSSecurityEngine) StoreCertificate(ctx context.Context, cert *models.TLSCertificate) error {
	query := `
		INSERT INTO tls_certificates (id, domain, certificate_pem, private_key_pem, expires_at, issuer, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (domain) DO UPDATE
		SET certificate_pem = $3, private_key_pem = $4, expires_at = $5, issuer = $6, updated_at = $8
	`

	_, err := e.db.Exec(ctx, query,
		cert.ID, cert.Domain, cert.CertificatePEM, cert.PrivateKeyPEM,
		cert.ExpiresAt, cert.Issuer, cert.CreatedAt, cert.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to store certificate: %w", err)
	}

	// Update cache
	e.mu.Lock()
	e.certCache[cert.Domain] = cert
	e.mu.Unlock()

	return nil
}

// GetCertificate retrieves a certificate for a domain.
func (e *TLSSecurityEngine) GetCertificate(ctx context.Context, domain string) (*models.TLSCertificate, error) {
	// Check cache first
	e.mu.RLock()
	if cert, exists := e.certCache[domain]; exists {
		e.mu.RUnlock()
		return cert, nil
	}
	e.mu.RUnlock()

	// Query database
	cert := &models.TLSCertificate{}
	query := `
		SELECT id, domain, certificate_pem, private_key_pem, expires_at, issuer, created_at, updated_at
		FROM tls_certificates
		WHERE domain = $1
	`

	err := e.db.QueryRow(ctx, query, domain).Scan(
		&cert.ID, &cert.Domain, &cert.CertificatePEM, &cert.PrivateKeyPEM,
		&cert.ExpiresAt, &cert.Issuer, &cert.CreatedAt, &cert.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("certificate not found: %w", err)
	}

	// Update cache
	e.mu.Lock()
	e.certCache[domain] = cert
	e.mu.Unlock()

	return cert, nil
}

// DeleteCertificate deletes a certificate from database and cache.
func (e *TLSSecurityEngine) DeleteCertificate(ctx context.Context, domain string) error {
	query := `DELETE FROM tls_certificates WHERE domain = $1`

	_, err := e.db.Exec(ctx, query, domain)
	if err != nil {
		return fmt.Errorf("failed to delete certificate: %w", err)
	}

	// Remove from cache
	e.mu.Lock()
	delete(e.certCache, domain)
	e.mu.Unlock()

	e.logger.Info("Deleted certificate", zap.String("domain", domain))
	return nil
}

// Check returns the health status of the TLS security engine.
func (e *TLSSecurityEngine) Check(ctx context.Context) *healthcheck.Result {
	// Count certificates and check expiry
	var totalCerts, expiringCerts int
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN expires_at < $1 THEN 1 END) as expiring
		FROM tls_certificates
	`

	expiryThreshold := time.Now().Add(30 * 24 * time.Hour)
	err := e.db.QueryRow(ctx, query, expiryThreshold).Scan(&totalCerts, &expiringCerts)

	status := healthcheck.StatusHealthy
	message := "TLS security engine is operational"

	if err != nil {
		status = healthcheck.StatusUnhealthy
		message = fmt.Sprintf("Database error: %v", err)
		e.logger.Error("Health check failed", zap.Error(err))
	} else if expiringCerts > 0 {
		status = healthcheck.StatusDegraded
		message = fmt.Sprintf("%d certificate(s) expiring within 30 days", expiringCerts)
	}

	return &healthcheck.Result{
		ComponentName: "tls_security_engine",
		Status:        status,
		Message:       message,
		Timestamp:     time.Now(),
		Details: map[string]interface{}{
			"total_certificates":    totalCerts,
			"expiring_certificates": expiringCerts,
			"acme_configured":       e.acmeManager != nil,
			"cached_certificates":   len(e.certCache),
		},
	}
}

// Name returns the name of the engine.
func (e *TLSSecurityEngine) Name() string {
	return "tls_security_engine"
}
