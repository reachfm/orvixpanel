package hosting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateFPMPoolBasic(t *testing.T) {
	body, err := GenerateFPMPool(FPMConfig{
		Username:    "alice",
		Domain:      "alice.test",
		PHPVersion:  "8.5",
	})
	require.NoError(t, err)
	require.Contains(t, body, "[orvix-alice-alice.test]")
	require.Contains(t, body, "user = alice")
	require.Contains(t, body, "group = alice")
	require.Contains(t, body, "listen = /run/php/orvix-alice-alice.test.sock")
	require.Contains(t, body, "pm = ondemand")
	require.Contains(t, body, "pm.max_children = 5")
	require.Contains(t, body, "pm.max_requests = 500")
	require.Contains(t, body, "php_admin_value[memory_limit] = 256M")
	require.Contains(t, body, "php_admin_value[open_basedir]")
	require.Contains(t, body, "listen.owner = www-data")
	require.Contains(t, body, "php_admin_flag[expose_php] = off")
}

func TestGenerateFPMPoolDynamic(t *testing.T) {
	body, err := GenerateFPMPool(FPMConfig{
		Username:      "bob",
		Domain:        "bob.test",
		PM:            "dynamic",
		MaxChildren:   10,
		StartServers:  3,
		MinSpareServers: 2,
		MaxSpareServers: 6,
	})
	require.NoError(t, err)
	require.Contains(t, body, "pm = dynamic")
	require.Contains(t, body, "pm.max_children = 10")
	require.Contains(t, body, "pm.start_servers = 3")
	require.Contains(t, body, "pm.min_spare_servers = 2")
	require.Contains(t, body, "pm.max_spare_servers = 6")
}

func TestGenerateFPMPoolRejectsEmpty(t *testing.T) {
	_, err := GenerateFPMPool(FPMConfig{Username: "", Domain: "x"})
	require.Error(t, err)
	_, err = GenerateFPMPool(FPMConfig{Username: "x", Domain: ""})
	require.Error(t, err)
}

func TestGenerateFPMPoolDefaults(t *testing.T) {
	body, err := GenerateFPMPool(FPMConfig{Username: "x", Domain: "y"})
	require.NoError(t, err)
	// Defaults
	require.Contains(t, body, "pm = ondemand")
	require.Contains(t, body, "pm.max_children = 5")
	require.Contains(t, body, "php_admin_value[memory_limit] = 256M")
	require.Contains(t, body, "php_admin_value[upload_max_filesize] = 64M")
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{int64(1.5 * 1024 * 1024), "1.5 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
	}
	for _, c := range cases {
		require.Equal(t, c.want, FormatBytes(c.in), "in=%d", c.in)
	}
}

func TestDiskUsageUsedPercent(t *testing.T) {
	d := DiskUsage{Bytes: 50 * 1024 * 1024, DiskLimitMB: 100}
	require.InDelta(t, 50.0, d.UsedPercent(), 0.01)
	d2 := DiskUsage{Bytes: 100 * 1024 * 1024, DiskLimitMB: 0}
	require.Equal(t, 0.0, d2.UsedPercent())
}
