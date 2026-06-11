package health

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCollector(t *testing.T) {
	c, err := NewCollector()
	require.NoError(t, err)
	require.NotNil(t, c)
	require.NotEmpty(t, c.hostname)
}

func TestCollector_Collect(t *testing.T) {
	c, err := NewCollector()
	require.NoError(t, err)

	m, err := c.Collect()
	require.NoError(t, err)
	require.NotNil(t, m)

	// Verify basic fields are populated
	require.NotZero(t, m.Timestamp)
	require.NotEmpty(t, m.Hostname)
	require.NotEmpty(t, m.GoVersion)
	require.NotEmpty(t, m.Uptime)

	// Memory should have values
	require.NotZero(t, m.Memory.TotalKB)
	require.Greater(t, m.Memory.TotalKB, m.Memory.AvailableKB)
	require.GreaterOrEqual(t, m.Memory.UsagePercent, 0.0)
	require.LessOrEqual(t, m.Memory.UsagePercent, 100.0)

	// Load average should have valid values
	require.GreaterOrEqual(t, m.LoadAverage.Load1, 0.0)
	require.GreaterOrEqual(t, m.LoadAverage.Load5, 0.0)
	require.GreaterOrEqual(t, m.LoadAverage.Load15, 0.0)

	// CPU should have values
	require.Greater(t, m.CPU.NumCPU, 0)
	require.Greater(t, m.CPU.NumGoroutine, 0)
}

func TestCollectMemory(t *testing.T) {
	c, err := NewCollector()
	require.NoError(t, err)

	var m MemoryMetrics
	err = c.collectMemory(&m)
	require.NoError(t, err)

	// Memory values should be reasonable
	require.Greater(t, m.TotalKB, uint64(0))
	require.GreaterOrEqual(t, m.AvailableKB, uint64(0))
	require.LessOrEqual(t, m.AvailableKB, m.TotalKB)
	require.Equal(t, m.TotalKB-m.AvailableKB, m.UsedKB)
	require.InDelta(t, float64(m.UsedKB)/float64(m.TotalKB)*100, m.UsagePercent, 0.1)

	// Swap values should be non-negative
	require.GreaterOrEqual(t, m.SwapTotalKB, uint64(0))
	require.LessOrEqual(t, m.SwapFreeKB, m.SwapTotalKB)
}

func TestCollectLoadAvg(t *testing.T) {
	c, err := NewCollector()
	require.NoError(t, err)

	var l LoadMetrics
	err = c.collectLoadAvg(&l)
	require.NoError(t, err)

	// Load averages should be non-negative
	require.GreaterOrEqual(t, l.Load1, 0.0)
	require.GreaterOrEqual(t, l.Load5, 0.0)
	require.GreaterOrEqual(t, l.Load15, 0.0)

	// Load5 should typically be >= Load1 (but not guaranteed)
	// Load15 should typically be >= Load5 (but not guaranteed)
}

func TestGetUptime(t *testing.T) {
	c, err := NewCollector()
	require.NoError(t, err)

	uptime := c.getUptime()
	require.NotEmpty(t, uptime)
	require.NotEqual(t, "unknown", uptime)
}

func TestCollectDisk(t *testing.T) {
	c, err := NewCollector()
	require.NoError(t, err)

	disks := c.collectDisk()
	// Should have at least root mount point
	require.NotEmpty(t, disks)

	for _, d := range disks {
		require.NotEmpty(t, d.MountPoint)
		// TotalKB may be 0 in test environments without /proc mountstats
		// Only validate if we have real data
		if d.TotalKB > 0 {
			require.Greater(t, d.TotalKB, uint64(0))
			require.LessOrEqual(t, d.UsedKB, d.TotalKB)
			require.LessOrEqual(t, d.AvailableKB, d.TotalKB)
			require.InDelta(t, float64(d.UsedKB)/float64(d.TotalKB)*100, d.UsagePercent, 0.1)
		}
		// Values should still be non-negative
		require.GreaterOrEqual(t, d.UsedKB, uint64(0))
		require.GreaterOrEqual(t, d.AvailableKB, uint64(0))
	}
}

func TestCheckService(t *testing.T) {
	// Test with a non-existent service
	status := CheckService("nonexistent-service-xyz123")
	require.Equal(t, "nonexistent-service-xyz123", status.Name)
	// Running should be false for non-existent service
	require.False(t, status.Running)
}

func TestMetricsJSON(t *testing.T) {
	c, err := NewCollector()
	require.NoError(t, err)

	m, err := c.Collect()
	require.NoError(t, err)

	// Verify JSON serialization works
	jsonBytes, err := json.Marshal(m)
	require.NoError(t, err)
	require.NotEmpty(t, jsonBytes)

	// Verify JSON contains expected fields
	jsonStr := string(jsonBytes)
	require.Contains(t, jsonStr, `"hostname"`)
	require.Contains(t, jsonStr, `"memory"`)
	require.Contains(t, jsonStr, `"load_average"`)
	require.Contains(t, jsonStr, `"cpu"`)
}

func TestMemoryMetrics_JSON(t *testing.T) {
	m := MemoryMetrics{
		TotalKB:      16384000,
		AvailableKB:  8000000,
		UsedKB:       8384000,
		UsagePercent: 51.2,
	}

	jsonBytes, err := json.Marshal(m)
	require.NoError(t, err)
	require.Contains(t, string(jsonBytes), `"total_kb":16384000`)
	require.Contains(t, string(jsonBytes), `"available_kb":8000000`)
}

func TestLoadMetrics_JSON(t *testing.T) {
	l := LoadMetrics{
		Load1:  1.5,
		Load5:  2.0,
		Load15: 1.8,
		Procs:  42,
	}

	jsonBytes, err := json.Marshal(l)
	require.NoError(t, err)
	require.Contains(t, string(jsonBytes), `"load_1m":1.5`)
	require.Contains(t, string(jsonBytes), `"load_5m":2`)
	require.Contains(t, string(jsonBytes), `"running_procs":42`)
}

func TestCPUMetrics_JSON(t *testing.T) {
	cpu := CPUMetrics{
		NumCPU:       8,
		NumGoroutine: 15,
		UserPCT:      25.5,
		SystemPCT:    10.2,
		IdlePCT:      64.3,
	}

	jsonBytes, err := json.Marshal(cpu)
	require.NoError(t, err)
	require.Contains(t, string(jsonBytes), `"num_cpu":8`)
	require.Contains(t, string(jsonBytes), `"user_percent":25.5`)
	require.Contains(t, string(jsonBytes), `"idle_percent":64.3`)
}

func TestMetrics_JSON(t *testing.T) {
	m := &Metrics{
		Hostname:  "test-host",
		Uptime:    "1h30m",
		GoVersion: "go1.21",
		Memory: MemoryMetrics{
			TotalKB:      16384000,
			AvailableKB:  8000000,
			UsedKB:       8384000,
			UsagePercent: 51.2,
		},
		LoadAverage: LoadMetrics{
			Load1:  1.5,
			Load5:  2.0,
			Load15: 1.8,
			Procs:  42,
		},
		CPU: CPUMetrics{
			NumCPU:       8,
			NumGoroutine: 15,
			UserPCT:      25.5,
			SystemPCT:    10.2,
			IdlePCT:      64.3,
		},
	}

	jsonBytes, err := json.Marshal(m)
	require.NoError(t, err)
	require.Contains(t, string(jsonBytes), `"hostname":"test-host"`)
	require.Contains(t, string(jsonBytes), `"uptime":"1h30m"`)
	require.Contains(t, string(jsonBytes), `"go_version":"go1.21"`)
	require.Contains(t, string(jsonBytes), `"memory"`)
	require.Contains(t, string(jsonBytes), `"load_average"`)
	require.Contains(t, string(jsonBytes), `"cpu"`)
}

func TestJSONRoundTrip(t *testing.T) {
	c, err := NewCollector()
	require.NoError(t, err)

	m1, err := c.Collect()
	require.NoError(t, err)

	// Marshal to JSON
	jsonBytes, err := json.Marshal(m1)
	require.NoError(t, err)

	// Unmarshal back
	var m2 Metrics
	err = json.Unmarshal(jsonBytes, &m2)
	require.NoError(t, err)

	// Verify key fields match
	require.Equal(t, m1.Hostname, m2.Hostname)
	require.Equal(t, m1.Uptime, m2.Uptime)
	require.Equal(t, m1.GoVersion, m2.GoVersion)
	require.Equal(t, m1.Memory.TotalKB, m2.Memory.TotalKB)
	require.Equal(t, m1.LoadAverage.Load1, m2.LoadAverage.Load1)
	require.Equal(t, m1.CPU.NumCPU, m2.CPU.NumCPU)
}