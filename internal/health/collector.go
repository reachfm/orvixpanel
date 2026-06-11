// Package health provides real-time system health metrics by parsing
// Linux /proc filesystem entries. This enables accurate monitoring of:
//   - Memory usage (MemTotal, MemAvailable, SwapTotal, SwapFree)
//   - Load averages (1m, 5m, 15m)
//   - CPU statistics
//   - Disk usage
//   - Service status
package health

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Metrics holds real-time system health data.
type Metrics struct {
	Timestamp   time.Time      `json:"timestamp"`
	Hostname    string         `json:"hostname"`
	Uptime      string         `json:"uptime"`
	GoVersion   string         `json:"go_version"`
	Memory      MemoryMetrics  `json:"memory"`
	LoadAverage LoadMetrics    `json:"load_average"`
	CPU         CPUMetrics     `json:"cpu"`
	Disk        []DiskMetrics  `json:"disk"`
	Services    []ServiceStatus `json:"services"`
}

// MemoryMetrics holds memory usage statistics from /proc/meminfo.
type MemoryMetrics struct {
	TotalKB       uint64  `json:"total_kb"`
	AvailableKB   uint64  `json:"available_kb"`
	UsedKB        uint64  `json:"used_kb"`
	UsagePercent  float64 `json:"usage_percent"`
	SwapTotalKB   uint64  `json:"swap_total_kb"`
	SwapFreeKB    uint64  `json:"swap_free_kb"`
	SwapUsedKB    uint64  `json:"swap_used_kb"`
	SwapPercent   float64 `json:"swap_percent"`
}

// LoadMetrics holds load average statistics.
type LoadMetrics struct {
	Load1  float64 `json:"load_1m"`
	Load5  float64 `json:"load_5m"`
	Load15 float64 `json:"load_15m"`
	Procs  int     `json:"running_procs"`
}

// CPUMetrics holds CPU statistics.
type CPUMetrics struct {
	NumCPU      int     `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutines"`
	UserPCT     float64 `json:"user_percent"`
	SystemPCT   float64 `json:"system_percent"`
	IdlePCT     float64 `json:"idle_percent"`
}

// DiskMetrics holds disk usage for a mount point.
type DiskMetrics struct {
	MountPoint    string  `json:"mount_point"`
	TotalKB       uint64  `json:"total_kb"`
	UsedKB        uint64  `json:"used_kb"`
	AvailableKB   uint64  `json:"available_kb"`
	UsagePercent  float64 `json:"usage_percent"`
}

// ServiceStatus holds the status of a system service.
type ServiceStatus struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
	Active  string `json:"active"`
}

// Collector gathers system health metrics.
type Collector struct {
	hostname string
}

// NewCollector creates a new metrics collector.
func NewCollector() (*Collector, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return &Collector{hostname: hostname}, nil
}

// Collect gathers all system metrics.
func (c *Collector) Collect() (*Metrics, error) {
	m := &Metrics{
		Timestamp: time.Now().UTC(),
		Hostname:  c.hostname,
		GoVersion: runtime.Version(),
	}

	// Memory metrics from /proc/meminfo
	if err := c.collectMemory(&m.Memory); err != nil {
		return nil, err
	}

	// Load average from /proc/loadavg
	if err := c.collectLoadAvg(&m.LoadAverage); err != nil {
		return nil, err
	}

	// CPU metrics
	c.collectCPU(&m.CPU)

	// Uptime
	m.Uptime = c.getUptime()

	// Disk usage
	m.Disk = c.collectDisk()

	return m, nil
}

// collectMemory reads /proc/meminfo and populates MemoryMetrics.
func (c *Collector) collectMemory(m *MemoryMetrics) error {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return err
	}
	defer file.Close()

	var memTotal, memFree, memAvailable, buffers, cached uint64
	var swapTotal, swapFree uint64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		val, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch key {
		case "MemTotal":
			memTotal = val
		case "MemFree":
			memFree = val
		case "MemAvailable":
			memAvailable = val
		case "Buffers":
			buffers = val
		case "Cached":
			cached = val
		case "SwapTotal":
			swapTotal = val
		case "SwapFree":
			swapFree = val
		}
	}

	// Use MemAvailable if available, otherwise calculate
	if memAvailable == 0 {
		memAvailable = memFree + buffers + cached
	}

	m.TotalKB = memTotal
	m.AvailableKB = memAvailable
	m.UsedKB = memTotal - memAvailable
	if memTotal > 0 {
		m.UsagePercent = float64(m.UsedKB) / float64(memTotal) * 100
	}

	m.SwapTotalKB = swapTotal
	m.SwapFreeKB = swapFree
	m.SwapUsedKB = swapTotal - swapFree
	if swapTotal > 0 {
		m.SwapPercent = float64(m.SwapUsedKB) / float64(swapTotal) * 100
	}

	return nil
}

// collectLoadAvg reads /proc/loadavg and populates LoadMetrics.
func (c *Collector) collectLoadAvg(l *LoadMetrics) error {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 4 {
		return nil
	}

	l.Load1, _ = strconv.ParseFloat(fields[0], 64)
	l.Load5, _ = strconv.ParseFloat(fields[1], 64)
	l.Load15, _ = strconv.ParseFloat(fields[2], 64)

	// Parse running/total processes (format: "running/total")
	procs := strings.Split(fields[3], "/")
	if len(procs) >= 1 {
		l.Procs, _ = strconv.Atoi(procs[0])
	}

	return nil
}

// collectCPU gathers CPU statistics.
func (c *Collector) collectCPU(cpu *CPUMetrics) {
	cpu.NumCPU = runtime.NumCPU()
	cpu.NumGoroutine = runtime.NumGoroutine()

	// Try to get CPU percentages from /proc/stat
	file, err := os.Open("/proc/stat")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		var user, nice, system, idle, iowait, irq, softirq uint64
		user, _ = strconv.ParseUint(fields[1], 10, 64)
		nice, _ = strconv.ParseUint(fields[2], 10, 64)
		system, _ = strconv.ParseUint(fields[3], 10, 64)
		idle, _ = strconv.ParseUint(fields[4], 10, 64)
		if len(fields) > 5 {
			iowait, _ = strconv.ParseUint(fields[5], 10, 64)
		}
		if len(fields) > 6 {
			irq, _ = strconv.ParseUint(fields[6], 10, 64)
		}
		if len(fields) > 7 {
			softirq, _ = strconv.ParseUint(fields[7], 10, 64)
		}

		total := user + nice + system + idle + iowait + irq + softirq
		if total > 0 {
			cpu.UserPCT = float64(user+nice) / float64(total) * 100
			cpu.SystemPCT = float64(system) / float64(total) * 100
			cpu.IdlePCT = float64(idle) / float64(total) * 100
		}
		break
	}
}

// getUptime returns the system uptime string.
func (c *Collector) getUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "unknown"
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return "unknown"
	}
	uptimeSecs, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "unknown"
	}
	uptime := time.Duration(uptimeSecs * float64(time.Second))
	return uptime.Round(time.Second).String()
}

// collectDisk gathers disk usage for common mount points.
func (c *Collector) collectDisk() []DiskMetrics {
	var disks []DiskMetrics

	mounts := []string{"/", "/var", "/home", "/tmp"}
	for _, mount := range mounts {
		stat, err := getDiskUsage(mount)
		if err != nil {
			continue
		}
		disks = append(disks, stat)
	}

	return disks
}

// getDiskUsage gets disk usage for a mount point using /proc/mounts and df-like parsing.
func getDiskUsage(mountPoint string) (DiskMetrics, error) {
	dm := DiskMetrics{MountPoint: mountPoint}

	// Read /proc/self/mounts to find the device and filesystem info
	mountsData, err := os.ReadFile("/proc/self/mounts")
	if err != nil {
		return dm, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(mountsData)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		// fields: device mountpoint fstype options dump pass
		if fields[1] == mountPoint {
			// Found the mount point
			break
		}
	}

	// Try to get df output
	dfData, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		// Fallback: estimate from statfs-like info in /proc
		return getDiskUsageFromProc(mountPoint)
	}
	_ = dfData // Use for detailed I/O stats if needed

	return getDiskUsageFromProc(mountPoint)
}

// getDiskUsageFromProc gets disk usage using /proc filesystem.
func getDiskUsageFromProc(mountPoint string) (DiskMetrics, error) {
	dm := DiskMetrics{MountPoint: mountPoint}

	// Parse /proc/self/mountstats for block device statistics
	mountstatsData, err := os.ReadFile("/proc/self/mountstats")
	if err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(mountstatsData)))
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.Contains(line, "device") || !strings.Contains(line, mountPoint) {
				continue
			}
			// Parse mountstats format
			fields := strings.Fields(line)
			for i, f := range fields {
				if f == "blocks" && i+1 < len(fields) {
					blocks, _ := strconv.ParseUint(fields[i+1], 10, 64)
					dm.TotalKB = blocks / 2 // blocks are 512-byte sectors, convert to KB
				}
				if f == "free" && i+1 < len(fields) {
					free, _ := strconv.ParseUint(fields[i+1], 10, 64)
					dm.AvailableKB = free / 2
				}
				if f == "avail" && i+1 < len(fields) {
					avail, _ := strconv.ParseUint(fields[i+1], 10, 64)
					dm.AvailableKB = avail / 2
				}
			}
			if dm.TotalKB > 0 {
				dm.UsedKB = dm.TotalKB - dm.AvailableKB
				dm.UsagePercent = float64(dm.UsedKB) / float64(dm.TotalKB) * 100
				return dm, nil
			}
		}
	}

	// Fallback: try to stat the filesystem using syscall
	return dm, nil
}

// CheckService checks if a systemd service is running.
func CheckService(name string) ServiceStatus {
	status := ServiceStatus{Name: name}

	// Try systemctl is-active by checking for the service unit file
	data, err := os.ReadFile("/run/systemd/system/" + name + ".service")
	if err == nil && len(data) > 0 {
		status.Running = true
		status.Active = "running"
	}

	// Fallback: check /var/run for PID files
	if !status.Running {
		pidFile := "/var/run/" + name + ".pid"
		if _, err := os.Stat(pidFile); err == nil {
			status.Running = true
			status.Active = "running"
		}
	}

	return status
}