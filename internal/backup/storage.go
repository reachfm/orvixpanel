/**
 * Storage backend interface for backup providers.
 */

package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// BackendType represents the type of storage backend.
type BackendType string

const (
	BackendLocal  BackendType = "local"
	BackendS3      BackendType = "s3"
	BackendMinio  BackendType = "minio"
	BackendWasabi BackendType = "wasabi"
)

// StorageProvider defines the interface for backup storage backends.
type StorageProvider interface {
	// Name returns the backend name.
	Name() string

	// Initialize configures the storage provider.
	Initialize(ctx context.Context, config map[string]string) error

	// Upload uploads a file to the storage backend.
	Upload(ctx context.Context, key string, data io.Reader, size int64) error

	// Download downloads a file from the storage backend.
	Download(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete deletes a file from the storage backend.
	Delete(ctx context.Context, key string) error

	// Exists checks if a file exists in the storage backend.
	Exists(ctx context.Context, key string) (bool, error)

	// List lists all files with a given prefix.
	List(ctx context.Context, prefix string) ([]string, error)

	// GetSize returns the size of a file.
	GetSize(ctx context.Context, key string) (int64, error)

	// GetURL returns a URL for accessing the file.
	GetURL(ctx context.Context, key string) (string, error)

	// Close closes the storage provider.
	Close() error
}

// StorageFactory creates storage providers based on backend type.
type StorageFactory struct {
	providers map[BackendType]StorageProvider
}

// NewStorageFactory creates a new storage factory.
func NewStorageFactory() *StorageFactory {
	return &StorageFactory{
		providers: make(map[BackendType]StorageProvider),
	}
}

// Register registers a storage provider for a backend type.
func (f *StorageFactory) Register(backend BackendType, provider StorageProvider) {
	f.providers[backend] = provider
}

// Create creates a storage provider for the given backend type.
func (f *StorageFactory) Create(backend BackendType) (StorageProvider, error) {
	provider, ok := f.providers[backend]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidBackend, backend)
	}
	return provider, nil
}

// LocalStorageProvider implements local filesystem storage.
type LocalStorageProvider struct {
	basePath string
}

// NewLocalStorageProvider creates a new local storage provider.
func NewLocalStorageProvider() *LocalStorageProvider {
	return &LocalStorageProvider{}
}

// Name returns the backend name.
func (p *LocalStorageProvider) Name() string {
	return "local"
}

// Initialize configures the local storage provider.
func (p *LocalStorageProvider) Initialize(ctx context.Context, config map[string]string) error {
	basePath := config["base_path"]
	if basePath == "" {
		basePath = "/var/lib/orvixpanel/backups"
	}

	// Ensure directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	p.basePath = basePath
	return nil
}

// resolvePath resolves a key to a full path.
func (p *LocalStorageProvider) resolvePath(key string) string {
	return filepath.Join(p.basePath, key)
}

// Upload uploads a file to local storage.
func (p *LocalStorageProvider) Upload(ctx context.Context, key string, data io.Reader, size int64) error {
	path := p.resolvePath(key)

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Copy data
	written, err := io.Copy(f, data)
	if err != nil {
		os.Remove(path)
		return fmt.Errorf("failed to write data: %w", err)
	}

	// Verify size
	if written != size {
		os.Remove(path)
		return fmt.Errorf("size mismatch: expected %d, wrote %d", size, written)
	}

	return nil
}

// Download downloads a file from local storage.
func (p *LocalStorageProvider) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	path := p.resolvePath(key)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrBackupNotFound
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return f, nil
}

// Delete deletes a file from local storage.
func (p *LocalStorageProvider) Delete(ctx context.Context, key string) error {
	path := p.resolvePath(key)

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Exists checks if a file exists.
func (p *LocalStorageProvider) Exists(ctx context.Context, key string) (bool, error) {
	path := p.resolvePath(key)

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat file: %w", err)
	}

	return true, nil
}

// List lists all files with a given prefix.
func (p *LocalStorageProvider) List(ctx context.Context, prefix string) ([]string, error) {
	searchPath := p.resolvePath(prefix)
	dir := filepath.Dir(searchPath)
	pattern := filepath.Base(searchPath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var results []string
	for _, entry := range entries {
		matched, _ := filepath.Match(pattern, entry.Name())
		if matched {
			relPath, _ := filepath.Rel(p.basePath, filepath.Join(dir, entry.Name()))
			results = append(results, filepath.ToSlash(relPath))
		}
	}

	return results, nil
}

// GetSize returns the size of a file.
func (p *LocalStorageProvider) GetSize(ctx context.Context, key string) (int64, error) {
	path := p.resolvePath(key)

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, ErrBackupNotFound
		}
		return 0, fmt.Errorf("failed to stat file: %w", err)
	}

	return info.Size(), nil
}

// GetURL returns a URL for accessing the file.
func (p *LocalStorageProvider) GetURL(ctx context.Context, key string) (string, error) {
	// For local storage, return the file path
	path := p.resolvePath(key)
	return "file://" + path, nil
}

// Close closes the storage provider.
func (p *LocalStorageProvider) Close() error {
	return nil
}

// S3StorageProvider implements S3-compatible storage (AWS S3, MinIO, Wasabi).
type S3StorageProvider struct {
	endpoint        string
	bucket          string
	region          string
	accessKeyID     string
	secretAccessKey string
	pathPrefix      string
	client          interface{} // Would be *s3.S3 in real implementation
}

// NewS3StorageProvider creates a new S3 storage provider.
func NewS3StorageProvider() *S3StorageProvider {
	return &S3StorageProvider{}
}

// Name returns the backend name.
func (p *S3StorageProvider) Name() string {
	return "s3"
}

// Initialize configures the S3 storage provider.
func (p *S3StorageProvider) Initialize(ctx context.Context, config map[string]string) error {
	p.endpoint = config["endpoint"]
	p.bucket = config["bucket"]
	p.region = config["region"]
	p.accessKeyID = config["access_key_id"]
	p.secretAccessKey = config["secret_access_key"]
	p.pathPrefix = config["path_prefix"]

	if p.pathPrefix == "" {
		p.pathPrefix = "backups"
	}

	// In production, this would initialize the AWS S3 client
	// For now, we validate the config is present
	if p.endpoint == "" || p.bucket == "" {
		return fmt.Errorf("endpoint and bucket are required for S3 backend")
	}

	return nil
}

// Upload uploads a file to S3.
func (p *S3StorageProvider) Upload(ctx context.Context, key string, data io.Reader, size int64) error {
	// In production, this would use AWS SDK to upload
	// key = filepath.Join(p.pathPrefix, key)
	// S3 upload with multipart support
	return nil
}

// Download downloads a file from S3.
func (p *S3StorageProvider) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	// In production, this would use AWS SDK to download
	return nil, nil
}

// Delete deletes a file from S3.
func (p *S3StorageProvider) Delete(ctx context.Context, key string) error {
	// In production, this would use AWS SDK to delete
	return nil
}

// Exists checks if a file exists in S3.
func (p *S3StorageProvider) Exists(ctx context.Context, key string) (bool, error) {
	// In production, this would use HeadObject
	return false, nil
}

// List lists all files with a given prefix in S3.
func (p *S3StorageProvider) List(ctx context.Context, prefix string) ([]string, error) {
	// In production, this would use ListObjectsV2
	return nil, nil
}

// GetSize returns the size of a file in S3.
func (p *S3StorageProvider) GetSize(ctx context.Context, key string) (int64, error) {
	// In production, this would use HeadObject
	return 0, nil
}

// GetURL returns a signed URL for accessing the file.
func (p *S3StorageProvider) GetURL(ctx context.Context, key string) (string, error) {
	// In production, this would generate a presigned URL
	return "", nil
}

// Close closes the S3 client.
func (p *S3StorageProvider) Close() error {
	return nil
}

// MinIOStorageProvider implements MinIO-specific storage.
type MinIOStorageProvider struct {
	*S3StorageProvider
}

// NewMinIOStorageProvider creates a new MinIO storage provider.
func NewMinIOStorageProvider() *MinIOStorageProvider {
	return &MinIOStorageProvider{
		S3StorageProvider: NewS3StorageProvider(),
	}
}

// Name returns the backend name.
func (p *MinIOStorageProvider) Name() string {
	return "minio"
}

// WasabiStorageProvider implements Wasabi-specific storage.
type WasabiStorageProvider struct {
	*S3StorageProvider
}

// NewWasabiStorageProvider creates a new Wasabi storage provider.
func NewWasabiStorageProvider() *WasabiStorageProvider {
	return &WasabiStorageProvider{
		S3StorageProvider: NewS3StorageProvider(),
	}
}

// Name returns the backend name.
func (p *WasabiStorageProvider) Name() string {
	return "wasabi"
}

// DefaultStorageFactory returns a factory with default providers registered.
func DefaultStorageFactory() *StorageFactory {
	factory := NewStorageFactory()
	factory.Register(BackendLocal, NewLocalStorageProvider())
	factory.Register(BackendS3, NewS3StorageProvider())
	factory.Register(BackendMinio, NewMinIOStorageProvider())
	factory.Register(BackendWasabi, NewWasabiStorageProvider())
	return factory
}