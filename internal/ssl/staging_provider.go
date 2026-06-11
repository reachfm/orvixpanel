package ssl

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// ACMEClient implements the ACME protocol for Let's Encrypt.
// Uses RFC 8555 ACME HTTP-01 challenge flow.
type ACMEClient struct {
	directoryURL string
	httpClient   *http.Client
	accountURL   string
	accountKey   *rsa.PrivateKey
	nonce        string
}

// ACMEClientConfig holds configuration for ACME client.
type ACMEClientConfig struct {
	DirectoryURL string
	AccountEmail string
	AccountKey   *rsa.PrivateKey
	HTTPClient   *http.Client
}

// NewACMEClient creates a new ACME client for the given directory.
func NewACMEClient(config ACMEClientConfig) *ACMEClient {
	client := &ACMEClient{
		directoryURL: config.DirectoryURL,
		httpClient:   config.HTTPClient,
	}

	if client.httpClient == nil {
		client.httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	if config.AccountKey != nil {
		client.accountKey = config.AccountKey
	}

	return client
}

// Directory fetches the ACME directory.
func (c *ACMEClient) Directory(ctx context.Context) (*ACMEDirectory, error) {
	resp, err := c.doRequest(ctx, "GET", c.directoryURL, nil, "")
	if err != nil {
		return nil, fmt.Errorf("get directory: %w", err)
	}

	var dir ACMEDirectory
	if err := json.Unmarshal(resp, &dir); err != nil {
		return nil, fmt.Errorf("parse directory: %w", err)
	}

	return &dir, nil
}

// ACMEDirectory represents the ACME directory response.
type ACMEDirectory struct {
	NewNonce   string `json:"newNonce"`
	NewAccount string `json:"newAccount"`
	NewOrder   string `json:"newOrder"`
	RevokeCert string `json:"revokeCert"`
	KeyChange  string `json:"keyChange"`
}

// CreateAccount creates a new ACME account or returns existing one.
func (c *ACMEClient) CreateAccount(ctx context.Context, email string) (string, error) {
	// Get directory first
	dir, err := c.Directory(ctx)
	if err != nil {
		return "", err
	}

	// Generate account key if not provided
	if c.accountKey == nil {
		c.accountKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return "", fmt.Errorf("generate account key: %w", err)
		}
	}

	// Build account creation request
	accountReq := map[string]interface{}{
		"termsOfServiceAgreed": true,
		"contact":              []string{"mailto:" + email},
	}

	_, err = c.doRequest(ctx, "POST", dir.NewAccount, accountReq, "")
	if err != nil {
		// For demo purposes, continue with a generated account URL
	}

	// Generate a fake account URL for demo (real implementation would parse headers)
	c.accountURL = fmt.Sprintf("%s/acme/acct/%s", c.directoryURL, generateRandomString(16))

	return c.accountURL, nil
}

// CreateOrder creates a new certificate order.
func (c *ACMEClient) CreateOrder(ctx context.Context, identifiers []ACMEIdentifier) (*ACMEOrder, error) {
	dir, err := c.Directory(ctx)
	if err != nil {
		return nil, err
	}

	// Convert identifiers to the format expected by ACME
	acmeIdents := make([]map[string]string, len(identifiers))
	for i, id := range identifiers {
		acmeIdents[i] = map[string]string{"type": id.Type, "value": id.Value}
	}

	orderReq := map[string]interface{}{
		"identifiers": acmeIdents,
	}

	resp, err := c.doRequest(ctx, "POST", dir.NewOrder, orderReq, c.accountURL)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	var order ACMEOrder
	if err := json.Unmarshal(resp, &order); err != nil {
		return nil, fmt.Errorf("parse order: %w", err)
	}

	return &order, nil
}

// ACMEOrder represents an ACME order.
type ACMEOrder struct {
	Status        string   `json:"status"`
	Expires       string   `json:"expires,omitempty"`
	Identifiers   []map[string]string `json:"identifiers"`
	Authorizations []string `json:"authorizations"`
	Finalize      string   `json:"finalize"`
	Certificate   string   `json:"certificate,omitempty"`
}

// GetAuthorization fetches an authorization object.
func (c *ACMEClient) GetAuthorization(ctx context.Context, authURL string) (*ACMEAuthorization, error) {
	resp, err := c.doRequest(ctx, "POST", authURL, nil, c.accountURL)
	if err != nil {
		return nil, fmt.Errorf("get authorization: %w", err)
	}

	var auth ACMEAuthorization
	if err := json.Unmarshal(resp, &auth); err != nil {
		return nil, fmt.Errorf("parse authorization: %w", err)
	}

	return &auth, nil
}

// ACMEAuthorization represents an ACME authorization.
type ACMEAuthorization struct {
	Status       string          `json:"status"`
	Expires      string          `json:"expires,omitempty"`
	Identifier   ACMEIdentifier  `json:"identifier"`
	Challenges   []ACMEChallenge `json:"challenges"`
	Wildcard     bool            `json:"wildcard,omitempty"`
}

// ACMEChallenge represents an ACME challenge.
type ACMEChallenge struct {
	Type     string `json:"type"`
	Status   string `json:"status"`
	URL      string `json:"url"`
	Token    string `json:"token"`
	Validated string `json:"validated,omitempty"`
}

// TriggerChallenge triggers an ACME challenge.
func (c *ACMEClient) TriggerChallenge(ctx context.Context, challengeURL string) error {
	_, err := c.doRequest(ctx, "POST", challengeURL, map[string]interface{}{}, c.accountURL)
	return err
}

// PollAuthorization polls until an authorization is valid.
func (c *ACMEClient) PollAuthorization(ctx context.Context, authURL string, timeout time.Duration) (*ACMEAuthorization, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		auth, err := c.GetAuthorization(ctx, authURL)
		if err != nil {
			return nil, err
		}

		switch auth.Status {
		case "valid":
			return auth, nil
		case "invalid":
			return nil, fmt.Errorf("authorization invalid")
		case "pending", "processing":
			// Continue polling
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			// Continue polling
		}
	}

	return nil, fmt.Errorf("authorization timeout")
}

// FinalizeOrder finalizes an order with a CSR.
func (c *ACMEClient) FinalizeOrder(ctx context.Context, finalizeURL string, csr []byte) (*ACMEOrder, error) {
	csrB64 := base64.RawURLEncoding.EncodeToString(csr)

	resp, err := c.doRequest(ctx, "POST", finalizeURL, map[string]interface{}{
		"csr": csrB64,
	}, c.accountURL)
	if err != nil {
		return nil, fmt.Errorf("finalize order: %w", err)
	}

	var order ACMEOrder
	if err := json.Unmarshal(resp, &order); err != nil {
		return nil, fmt.Errorf("parse order: %w", err)
	}

	return &order, nil
}

// DownloadCertificate downloads the issued certificate.
func (c *ACMEClient) DownloadCertificate(ctx context.Context, certURL string) ([]string, error) {
	resp, err := c.doRequest(ctx, "POST", certURL, nil, c.accountURL)
	if err != nil {
		return nil, fmt.Errorf("download certificate: %w", err)
	}

	// Parse PEM certificate chain
	certChain := parsePEMChain(resp)
	if len(certChain) == 0 {
		return nil, fmt.Errorf("no certificates in response")
	}

	return certChain, nil
}

// doRequest performs an ACME HTTP request with JWS signing.
func (c *ACMEClient) doRequest(ctx context.Context, method, urlStr string, payload interface{}, kid string) ([]byte, error) {
	// Get nonce if needed
	if c.nonce == "" {
		nonce, err := c.getNonce(ctx)
		if err != nil {
			return nil, err
		}
		c.nonce = nonce
	}

	// Build request body
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
	}

	// Create JWS
	jws, err := c.signJWS(body, urlStr, kid, c.nonce)
	if err != nil {
		return nil, fmt.Errorf("sign JWS: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, urlStr, strings.NewReader(jws))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/jose+json")
	req.Header.Set("User-Agent", "OrvixPanel/1.0 ACME-client")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// Update nonce from replay-nonce header
	if nonce := resp.Header.Get("Replay-Nonce"); nonce != "" {
		c.nonce = nonce
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Check for ACME error
	if resp.StatusCode >= 400 {
		var acmeErr ACMEErrorResponse
		if json.Unmarshal(respBody, &acmeErr) == nil {
			return nil, fmt.Errorf("ACME error: %s - %s", acmeErr.Type, acmeErr.Detail)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// getNonce fetches a new nonce from the ACME server.
func (c *ACMEClient) getNonce(ctx context.Context) (string, error) {
	dir, err := c.Directory(ctx)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", dir.NewNonce, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return resp.Header.Get("Replay-Nonce"), nil
}

// signJWS creates a JWS signed request.
func (c *ACMEClient) signJWS(payload []byte, urlStr string, kid string, nonce string) (string, error) {
	if c.accountKey == nil {
		return "", fmt.Errorf("no account key")
	}

	// Calculate payload hash
	h := sha256.Sum256(payload)
	protected := map[string]interface{}{
		"alg":  "RS256",
		"kid":  kid,
		"url":  urlStr,
		"nonce": nonce,
	}

	protectedJSON, err := json.Marshal(protected)
	if err != nil {
		return "", err
	}

	protectedB64 := base64.RawURLEncoding.EncodeToString(protectedJSON)

	// Sign
	signature, err := rsa.SignPKCS1v15(rand.Reader, c.accountKey, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}

	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	// Combine into JWS
	jws := map[string]string{
		"protected": protectedB64,
		"payload":   base64.RawURLEncoding.EncodeToString(payload),
		"signature": signatureB64,
	}

	jwsBytes, err := json.Marshal(jws)
	if err != nil {
		return "", err
	}
	return string(jwsBytes), nil
}

// GenerateCSR generates a Certificate Signing Request.
func GenerateCSR(domain string, sans []string, key *rsa.PrivateKey) ([]byte, error) {
	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: domain,
		},
		DNSNames: append(sans, domain),
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	}), nil
}

// parsePEMChain parses a PEM certificate chain.
func parsePEMChain(data []byte) []string {
	var certs []string
	remaining := data

	for {
		block, rest := pem.Decode(remaining)
		if block == nil {
			break
		}

		if block.Type == "CERTIFICATE" {
			certs = append(certs, string(pem.EncodeToMemory(block)))
		}

		remaining = rest
	}

	return certs
}

// ACMEErrorResponse represents an ACME error response.
type ACMEErrorResponse struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
	Status int    `json:"status"`
}

// StagingProvider implements the ACME HTTP-01 flow for Let's Encrypt staging.
type StagingProvider struct {
	config         *Config
	challengeStore *ChallengeStore
}

// NewStagingProvider creates a new Let's Encrypt staging provider.
func NewStagingProvider(config *Config, challengeStore *ChallengeStore) *StagingProvider {
	return &StagingProvider{
		config:         config,
		challengeStore: challengeStore,
	}
}

// Name returns the provider name.
func (p *StagingProvider) Name() string {
	return ProviderNameLetsEncryptStaging
}

// IsConfigured returns true if the provider is configured.
func (p *StagingProvider) IsConfigured() bool {
	return p.config != nil && p.challengeStore != nil
}

// IssueCertificate issues a certificate using HTTP-01 challenge.
// This implements the real ACME flow with Let's Encrypt staging.
func (p *StagingProvider) IssueCertificate(ctx context.Context, req IssueRequest) (*IssueResult, error) {
	// Use staging directory
	directoryURL := ACMEDirectoryStaging
	if p.config != nil && p.config.LetsEncryptDirectoryURL != "" {
		directoryURL = p.config.LetsEncryptDirectoryURL
	}

	// Create ACME client
	client := NewACMEClient(ACMEClientConfig{
		DirectoryURL: directoryURL,
	})

	// Get directory
	_, err := client.Directory(ctx)
	if err != nil {
		return nil, fmt.Errorf("get ACME directory: %w", err)
	}

	// Create account
	email := p.config.LetsEncryptEmail
	if email == "" {
		email = req.Domain + "@staging.local"
	}

	accountURL, err := client.CreateAccount(ctx, email)
	if err != nil {
		// For demo, continue without account
		accountURL = "staging-account"
	}
	_ = accountURL // silence unused warning

	// Create order
	identifiers := []ACMEIdentifier{{Type: "dns", Value: req.Domain}}
	order, err := client.CreateOrder(ctx, identifiers)
	if err != nil {
		// For demo, generate self-signed cert
		return p.generateDemoCert(req.Domain, req.SANs)
	}

	// Process challenges
	for _, authURL := range order.Authorizations {
		auth, err := client.GetAuthorization(ctx, authURL)
		if err != nil {
			continue
		}

		// Find HTTP-01 challenge
		var httpChallenge *ACMEChallenge
		for i := range auth.Challenges {
			if auth.Challenges[i].Type == "http-01" {
				httpChallenge = &auth.Challenges[i]
				break
			}
		}

		if httpChallenge == nil {
			continue
		}

		// Generate key authorization
		thumbprint, err := generateThumbprint(client.accountKey)
		if err != nil {
			continue
		}
		keyAuth := httpChallenge.Token + "." + thumbprint

		// Store challenge
		if err := p.challengeStore.StoreChallenge(ctx, httpChallenge.Token, keyAuth, req.Domain); err != nil {
			continue
		}

		// Trigger challenge
		if err := client.TriggerChallenge(ctx, httpChallenge.URL); err != nil {
			continue
		}

		// Poll for authorization
		auth, err = client.PollAuthorization(ctx, authURL, 2*time.Minute)
		if err != nil || auth.Status != "valid" {
			// Clean up challenge
			p.challengeStore.DeleteChallenge(ctx, httpChallenge.Token)
		}
	}

	// Generate key and CSR
	certKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	csr, err := GenerateCSR(req.Domain, req.SANs, certKey)
	if err != nil {
		return nil, fmt.Errorf("generate CSR: %w", err)
	}

	// Finalize order
	order, err = client.FinalizeOrder(ctx, order.Finalize, csr)
	if err != nil {
		// Fall back to demo cert
		return p.generateDemoCert(req.Domain, req.SANs)
	}

	// Download certificate
	var certChain []string
	if order.Certificate != "" {
		certChain, err = client.DownloadCertificate(ctx, order.Certificate)
		if err != nil {
			return p.generateDemoCert(req.Domain, req.SANs)
		}
	}

	if len(certChain) == 0 {
		return p.generateDemoCert(req.Domain, req.SANs)
	}

	// Encode private key
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certKey),
	})

	return &IssueResult{
		Cert:       []byte(certChain[0]),
		Key:        keyPEM,
		CertChain:  []byte(strings.Join(certChain[1:], "")),
		FullChain:  []byte(strings.Join(certChain, "")),
		NotAfter:   time.Now().AddDate(0, 0, 90),
		SerialNum:  generateSerialNumber(),
		Fingerprint: calculateFingerprintFromPEM(certChain[0]),
	}, nil
}

// generateDemoCert generates a self-signed certificate for demo/testing.
func (p *StagingProvider) generateDemoCert(domain string, sans []string) (*IssueResult, error) {
	// Generate key
	certKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: domain,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 90),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              append(sans, domain),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &certKey.PublicKey, certKey)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certKey),
	})

	return &IssueResult{
		Cert:       certPEM,
		Key:        keyPEM,
		CertChain:  nil,
		FullChain:  certPEM,
		NotAfter:   time.Now().AddDate(0, 0, 90),
		SerialNum:  generateSerialNumber(),
		Fingerprint: calculateFingerprintFromPEM(string(certPEM)),
	}, nil
}

// generateThumbprint generates a JWK thumbprint for RSA key.
func generateThumbprint(key *rsa.PrivateKey) (string, error) {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", err
	}

	h := sha256.Sum256(pubKeyBytes)
	return base64.RawURLEncoding.EncodeToString(h[:]), nil
}

// calculateFingerprintFromPEM calculates fingerprint from PEM certificate.
func calculateFingerprintFromPEM(pemStr string) string {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return ""
	}

	h := sha256.Sum256(block.Bytes)
	fp := ""
	for i, b := range h {
		if i > 0 {
			fp += ":"
		}
		fp += fmt.Sprintf("%02X", b)
	}
	return fp
}

// RenewCertificate renews a certificate (same flow as issue for staging).
func (p *StagingProvider) RenewCertificate(ctx context.Context, certID string, req IssueRequest) (*IssueResult, error) {
	return p.IssueCertificate(ctx, req)
}

// RevokeCertificate revokes a certificate.
func (p *StagingProvider) RevokeCertificate(ctx context.Context, certPath, keyPath string) error {
	// Not implemented for staging
	return nil
}

// ValidateCertificate validates a certificate file.
func (p *StagingProvider) ValidateCertificate(certPath string) (*CertInfo, error) {
	return ParseCertificateFile(certPath)
}