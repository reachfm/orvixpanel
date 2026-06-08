package ssl

import (
	"fmt"
	"os"
	"path/filepath"
)

// Storage handles certificate file storage operations.
type Storage struct {
	baseDir string
}

// NewStorage creates a new certificate storage handler.
func NewStorage(baseDir string) *Storage {
	return &Storage{baseDir: baseDir}
}

// EnsureDomainDir creates the directory structure for a domain's certificates.
func (s *Storage) EnsureDomainDir(domain string) error {
	dir := filepath.Join(s.baseDir, domain)
	return os.MkdirAll(dir, 0700) // Restricted permissions for security
}

// WriteCertFiles writes certificate files to disk.
// Returns the paths to the written files.
func (s *Storage) WriteCertFiles(domain string, result *IssueResult) (*CertPaths, error) {
	domainDir := filepath.Join(s.baseDir, domain)

	// Ensure directory exists
	if err := os.MkdirAll(domainDir, 0700); err != nil {
		return nil, &Error{Op: "create domain dir", Err: err}
	}

	paths := &CertPaths{
		CertPath:      filepath.Join(domainDir, "cert.pem"),
		KeyPath:       filepath.Join(domainDir, "privkey.pem"),
		ChainPath:     filepath.Join(domainDir, "chain.pem"),
		FullChainPath: filepath.Join(domainDir, "fullchain.pem"),
	}

	// Write certificate (0600 - owner read/write only)
	if err := s.writeFile(paths.CertPath, result.Cert, 0600); err != nil {
		return nil, &Error{Op: "write cert", Err: err}
	}

	// Write private key (0600 - owner read/write only)
	if err := s.writeFile(paths.KeyPath, result.Key, 0600); err != nil {
		return nil, &Error{Op: "write key", Err: err}
	}

	// Write chain (0644 - readable)
	if err := s.writeFile(paths.ChainPath, result.CertChain, 0644); err != nil {
		return nil, &Error{Op: "write chain", Err: err}
	}

	// Write full chain (0644 - readable)
	if err := s.writeFile(paths.FullChainPath, result.FullChain, 0644); err != nil {
		return nil, &Error{Op: "write fullchain", Err: err}
	}

	return paths, nil
}

// writeFile writes content to a file with specified permissions.
func (s *Storage) writeFile(path string, content []byte, mode os.FileMode) error {
	return os.WriteFile(path, content, mode)
}

// ReadCert reads a certificate file.
func (s *Storage) ReadCert(domain string) ([]byte, error) {
	path := filepath.Join(s.baseDir, domain, "cert.pem")
	return os.ReadFile(path)
}

// ReadKey reads a private key file.
func (s *Storage) ReadKey(domain string) ([]byte, error) {
	path := filepath.Join(s.baseDir, domain, "privkey.pem")
	return os.ReadFile(path)
}

// ReadChain reads a certificate chain file.
func (s *Storage) ReadChain(domain string) ([]byte, error) {
	path := filepath.Join(s.baseDir, domain, "chain.pem")
	return os.ReadFile(path)
}

// ReadFullChain reads the full certificate chain.
func (s *Storage) ReadFullChain(domain string) ([]byte, error) {
	path := filepath.Join(s.baseDir, domain, "fullchain.pem")
	return os.ReadFile(path)
}

// DeleteCertFiles removes all certificate files for a domain.
func (s *Storage) DeleteCertFiles(domain string) error {
	domainDir := filepath.Join(s.baseDir, domain)

	// Check if directory exists
	if _, err := os.Stat(domainDir); os.IsNotExist(err) {
		return nil // Nothing to delete
	}

	// Remove all files in the domain directory
	files, err := os.ReadDir(domainDir)
	if err != nil {
		return &Error{Op: "read domain dir", Err: err}
	}

	for _, file := range files {
		if err := os.Remove(filepath.Join(domainDir, file.Name())); err != nil {
			return &Error{Op: "remove file", Err: err}
		}
	}

	// Remove the directory
	if err := os.Remove(domainDir); err != nil {
		return &Error{Op: "remove domain dir", Err: err}
	}

	return nil
}

// FileExists checks if a file exists.
func (s *Storage) FileExists(domain, filename string) bool {
	path := filepath.Join(s.baseDir, domain, filename)
	_, err := os.Stat(path)
	return err == nil
}

// GetCertStats returns statistics about stored certificates.
func (s *Storage) GetCertStats() (*StorageStats, error) {
	stats := &StorageStats{}

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, &Error{Op: "read base dir", Err: err}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			stats.DomainCount++
			domainPath := filepath.Join(s.baseDir, entry.Name())

			// Count files in domain directory
			files, _ := os.ReadDir(domainPath)
			for _, f := range files {
				if !f.IsDir() {
					info, _ := f.Info()
					stats.TotalSize += int64(info.Size())
					stats.FileCount++
				}
			}
		}
	}

	return stats, nil
}

// StorageStats holds statistics about certificate storage.
type StorageStats struct {
	DomainCount int
	FileCount   int
	TotalSize   int64
}

// ImportCertFiles imports an externally-managed certificate.
// Returns the paths where files were stored.
func (s *Storage) ImportCertFiles(domain, certPEM, keyPEM, chainPEM string) (*CertPaths, error) {
	domainDir := filepath.Join(s.baseDir, domain)

	// Ensure directory exists
	if err := os.MkdirAll(domainDir, 0700); err != nil {
		return nil, &Error{Op: "create import dir", Err: err}
	}

	paths := &CertPaths{
		CertPath:      filepath.Join(domainDir, "cert.pem"),
		KeyPath:       filepath.Join(domainDir, "privkey.pem"),
		ChainPath:     filepath.Join(domainDir, "chain.pem"),
		FullChainPath: filepath.Join(domainDir, "fullchain.pem"),
	}

	// Write files with appropriate permissions
	if err := s.writeFile(paths.CertPath, []byte(certPEM), 0644); err != nil {
		return nil, &Error{Op: "import cert", Err: err}
	}

	if err := s.writeFile(paths.KeyPath, []byte(keyPEM), 0600); err != nil {
		return nil, &Error{Op: "import key", Err: err}
	}

	if err := s.writeFile(paths.ChainPath, []byte(chainPEM), 0644); err != nil {
		return nil, &Error{Op: "import chain", Err: err}
	}

	// Create fullchain by combining cert and chain
	fullChain := certPEM + "\n" + chainPEM
	if err := s.writeFile(paths.FullChainPath, []byte(fullChain), 0644); err != nil {
		return nil, &Error{Op: "import fullchain", Err: err}
	}

	return paths, nil
}

// ValidatePaths checks if all required certificate files exist.
func (s *Storage) ValidatePaths(domain string) error {
	domainDir := filepath.Join(s.baseDir, domain)

	files := []string{"cert.pem", "privkey.pem", "fullchain.pem"}
	for _, f := range files {
		path := filepath.Join(domainDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("%w: missing %s", ErrFileNotFound, f)
		}
	}

	return nil
}

// GetDomainPath returns the full path to a domain's certificate directory.
func (s *Storage) GetDomainPath(domain string) string {
	return filepath.Join(s.baseDir, domain)
}