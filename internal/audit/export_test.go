package audit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"github.com/stretchr/testify/require"
)

func TestFormatCSV(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	entry := models.AuditEntry{
		Timestamp:    ts,
		UserID:       "user456",
		UserEmail:    "test@example.com",
		UserRole:     "admin",
		ActorIP:      "192.168.1.1",
		SessionID:    "session789",
		Action:       "vault.read",
		ResourceType: "secret",
		ResourceID:   "secret123",
		ResourceName: "API_KEY",
		Result:       "success",
		DurationMS:   150,
		Detail:       "Read secret",
	}

	csv := FormatCSV(entry)
	require.NotEmpty(t, csv, "FormatCSV returned empty string")
	// Verify basic structure - should have timestamp
	require.Contains(t, csv, "2024-01-15T10:30:00Z")
}

func TestFormatCSV_EscapesQuotes(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	entry := models.AuditEntry{
		Timestamp: ts,
		UserEmail: `user"with"quotes`,
		Action:    "test.action",
		Result:    "success",
	}

	csv := FormatCSV(entry)
	// Should have doubled quotes
	require.Contains(t, csv, `""`, "FormatCSV should escape quotes with double quotes")
}

func TestFormatCSV_EscapesCommas(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	entry := models.AuditEntry{
		Timestamp: ts,
		UserEmail: "user,with,commas",
		Action:    "test.action",
		Result:    "success",
	}

	csv := FormatCSV(entry)
	// Should contain quoted value with escaped comma
	require.Contains(t, csv, `"user,with,commas"`, "FormatCSV should quote value with comma: %s", csv)
}

func TestEscapeCSV(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"simple", "simple"},
		{`with"quote`, `"with""quote"`},
		{"with,comma", `"with,comma"`},
		{" leading", `" leading"`},
	}

	for _, tt := range tests {
		got := escapeCSV(tt.input)
		require.Equal(t, tt.expected, got, "escapeCSV(%q)", tt.input)
	}
}

func TestFormatJSONEntry(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	entry := models.AuditEntry{
		Timestamp:    ts,
		UserID:       "user456",
		UserEmail:    "test@example.com",
		Action:       "vault.read",
		ResourceType: "secret",
		Result:       "success",
		Detail:       "Read secret",
	}

	json, err := formatJSONEntry(entry)
	require.NoError(t, err, "formatJSONEntry returned error")

	// Verify it's valid JSON
	require.True(t, strings.HasPrefix(json, "{") && strings.HasSuffix(json, "}"),
		"formatJSONEntry should return valid JSON object: %s", json)

	// Verify timestamp is RFC3339
	require.Contains(t, json, "2024-01-15T10:30:00Z", "formatJSONEntry should have RFC3339 timestamp")
}

func TestApplyFilters(t *testing.T) {
	db := newTestDB(t)
	a, err := New(context.Background(), db)
	require.NoError(t, err)

	// Seed test data using Record
	for i := 0; i < 5; i++ {
		require.NoError(t, a.Record(context.Background(), Event{
			Action: "test.action",
			Result: "success",
			UserID: "user",
		}))
	}

	tests := []struct {
		name     string
		req      ExportRequest
		expected int
	}{
		{
			name:     "no filters",
			req:      ExportRequest{},
			expected: 5,
		},
		{
			name:     "action filter",
			req:      ExportRequest{Action: "test"},
			expected: 5,
		},
		{
			name:     "result filter",
			req:      ExportRequest{Result: "success"},
			expected: 5,
		},
		{
			name:     "combined filters",
			req:      ExportRequest{Action: "test", Result: "success"},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := db.Model(&models.AuditEntry{})
			applyFilters(q, tt.req)
			var count int64
			q.Count(&count)
			require.Equal(t, tt.expected, int(count))
		})
	}
}

func TestExportCSV(t *testing.T) {
	db := newTestDB(t)
	a, err := New(context.Background(), db)
	require.NoError(t, err)

	// Insert test data
	for i := 0; i < 3; i++ {
		require.NoError(t, a.Record(context.Background(), Event{
			Action: "test.action",
			Result: "success",
			UserID: "user",
		}))
	}

	var lines []string
	w := func(line string) error {
		lines = append(lines, line)
		return nil
	}

	count, err := a.ExportCSV(context.Background(), w, ExportRequest{MaxRows: 100})
	require.NoError(t, err, "ExportCSV returned error")

	require.Equal(t, 3, count, "ExportCSV count mismatch")

	require.Equal(t, 4, len(lines), "ExportCSV lines should be header + 3 rows")

	// Verify header
	require.Equal(t, CSVHeader, lines[0], "First line should be header")
}

func TestExportJSON(t *testing.T) {
	db := newTestDB(t)
	a, err := New(context.Background(), db)
	require.NoError(t, err)

	// Insert test data
	for i := 0; i < 3; i++ {
		require.NoError(t, a.Record(context.Background(), Event{
			Action: "test.action",
			Result: "success",
			UserID: "user",
		}))
	}

	var lines []string
	w := func(line string) error {
		lines = append(lines, line)
		return nil
	}

	count, err := a.ExportJSON(context.Background(), w, ExportRequest{MaxRows: 100})
	require.NoError(t, err, "ExportJSON returned error")

	require.Equal(t, 3, count, "ExportJSON count mismatch")

	require.Equal(t, 3, len(lines), "ExportJSON lines should be 3")

	// Verify each line is valid JSON
	for i, line := range lines {
		require.True(t, strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}"),
			"Line %d should be JSON object: %s", i, line)
	}
}

func TestExport_FileTransportCSV(t *testing.T) {
	db := newTestDB(t)
	a, err := New(context.Background(), db)
	require.NoError(t, err)

	// Insert test data
	for i := 0; i < 2; i++ {
		require.NoError(t, a.Record(context.Background(), Event{
			Action: "test.action",
			Result: "success",
		}))
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_export.csv")

	resp, err := a.Export(context.Background(), ExportRequest{
		Format:    "csv",
		Transport: "file",
		FilePath:  filePath,
	})
	require.NoError(t, err, "Export returned error")

	require.Equal(t, 2, resp.Exported, "Exported count mismatch")

	// Read file and verify
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.NotEmpty(t, data, "Export file should not be empty")
}

func TestExport_FileTransportJSON(t *testing.T) {
	db := newTestDB(t)
	a, err := New(context.Background(), db)
	require.NoError(t, err)

	// Insert test data
	for i := 0; i < 2; i++ {
		require.NoError(t, a.Record(context.Background(), Event{
			Action: "test.action",
			Result: "success",
		}))
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_export.json")

	resp, err := a.Export(context.Background(), ExportRequest{
		Format:    "json",
		Transport: "file",
		FilePath:  filePath,
	})
	require.NoError(t, err, "Export returned error")

	require.Equal(t, 2, resp.Exported, "Exported count mismatch")

	// Read file and verify
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.NotEmpty(t, data, "Export file should not be empty")

	// Each line should be valid JSON
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	require.Len(t, lines, 2)
	for _, line := range lines {
		require.True(t, strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}"),
			"Line should be JSON object: %s", line)
	}
}

func TestExport_WithFilters(t *testing.T) {
	db := newTestDB(t)
	a, err := New(context.Background(), db)
	require.NoError(t, err)

	// Seed mixed data
	require.NoError(t, a.Record(context.Background(), Event{
		Action: "vault.read", Result: "success", UserID: "u1",
	}))
	require.NoError(t, a.Record(context.Background(), Event{
		Action: "vault.write", Result: "success", UserID: "u1",
	}))
	require.NoError(t, a.Record(context.Background(), Event{
		Action: "vault.delete", Result: "denied", UserID: "u2",
	}))

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "filtered.csv")

	// Export only "success" results
	resp, err := a.Export(context.Background(), ExportRequest{
		Format:    "csv",
		Transport: "file",
		FilePath:  filePath,
		Result:    "success",
	})
	require.NoError(t, err)
	require.Equal(t, 2, resp.Exported, "Should export only success results")

	// Export only "vault.read" action
	resp, err = a.Export(context.Background(), ExportRequest{
		Format:    "csv",
		Transport: "file",
		FilePath:  filePath,
		Action:    "vault.read",
	})
	require.NoError(t, err)
	require.Equal(t, 1, resp.Exported, "Should export only vault.read action")
}