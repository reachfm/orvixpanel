// Disk + inode usage tracker.
//
// The actual `du` and `stat` calls live in
// provision_linux.go (//go:build linux). The pure-Go math
// (byte formatting, %-used, etc.) lives here so the package
// compiles on every platform.
package hosting

import "fmt"

// DiskUsage is the result of a usage probe.
type DiskUsage struct {
	Bytes       int64
	Inodes      int64
	DiskLimitMB int64
	InodeLimit  int64
}

// UsedPercent returns the percentage of the disk limit consumed.
// Returns 0 if no limit is set.
func (d DiskUsage) UsedPercent() float64 {
	if d.DiskLimitMB == 0 {
		return 0
	}
	limitBytes := d.DiskLimitMB * 1024 * 1024
	if limitBytes == 0 {
		return 0
	}
	return float64(d.Bytes) / float64(limitBytes) * 100.0
}

// FormatBytes is a small helper for logs / API responses.
// 1.5 KB / 1.5 MB / 1.5 GB. Returns the input as string if < 1KB.
func FormatBytes(n int64) string {
	const (
		KB = 1024
		MB = 1024 * 1024
		GB = 1024 * 1024 * 1024
	)
	switch {
	case n < KB:
		return fmt.Sprintf("%d B", n)
	case n < MB:
		return fmt.Sprintf("%.1f KB", float64(n)/KB)
	case n < GB:
		return fmt.Sprintf("%.1f MB", float64(n)/MB)
	default:
		return fmt.Sprintf("%.2f GB", float64(n)/GB)
	}
}
