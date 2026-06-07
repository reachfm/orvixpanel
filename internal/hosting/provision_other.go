//go:build !linux

// Stubs for non-Linux builds. The package compiles but every
// OS-exec method returns ErrUnsupported.
package hosting

// Service is a no-op on non-Linux platforms.
type Service struct {
	Paths Paths
}

func NewService() *Service             { return &Service{Paths: DefaultPaths()} }
func NewServiceWithPaths(p Paths) *Service { return &Service{Paths: p} }

func (s *Service) CreateAccount(username string) (int, error) { return 0, ErrUnsupported }
func (s *Service) SuspendAccount(username string) error       { return ErrUnsupported }
func (s *Service) UnsuspendAccount(username string) error     { return ErrUnsupported }
func (s *Service) DeleteAccount(username string) error        { return ErrUnsupported }
func (s *Service) DiskUsed(path string) (int64, error)        { return 0, ErrUnsupported }
func (s *Service) InodeCount(path string) (int64, error)      { return 0, ErrUnsupported }
func (s *Service) PathInfo(path string) (DiskUsage, error)    { return DiskUsage{}, ErrUnsupported }

// Domain / nginx / php-fpm write paths also stub.
func (s *Service) WriteVHostConfig(username, domain, body string) error { return ErrUnsupported }
func (s *Service) RemoveVHostConfig(username, domain string) error      { return ErrUnsupported }
func (s *Service) WriteFPMPool(username, domain, body string) error    { return ErrUnsupported }
func (s *Service) RemoveFPMPool(username, domain string) error         { return ErrUnsupported }
func (s *Service) ReloadNginx() error                                  { return ErrUnsupported }
func (s *Service) TestNginx() error                                    { return ErrUnsupported }
func (s *Service) ReloadPHP() error                                    { return ErrUnsupported }
func (s *Service) TestPHP() error                                      { return ErrUnsupported }
func (s *Service) ValidateDomain(name string) error                    { return ErrUnsupported }
func (s *Service) DomainOwnedBy(username, domain string) (bool, error)  { return false, ErrUnsupported }
func (s *Service) CreateDomain(username, domain string) error          { return ErrUnsupported }
func (s *Service) DeleteDomain(username, domain string) error          { return ErrUnsupported }
func (s *Service) AtomicSwap(username, domain, newRelease string) error {
	return ErrUnsupported
}
func (s *Service) CurrentRelease(username, domain string) (string, error) {
	return "", ErrUnsupported
}
func (s *Service) ListReleases(username, domain string) ([]string, error) {
	return nil, ErrUnsupported
}
