package audit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.AuditEntry{}))
	return db
}

func seedRows(t *testing.T, a *Auditor) {
	t.Helper()
	for i := 0; i < 5; i++ {
		require.NoError(t, a.Record(context.Background(), Event{
			Action:     "domain.create",
			Result:     "success",
			UserID:     "u1",
			UserEmail:  "u1@x",
			ActorIP:    "1.2.3.4",
			ResourceID: "d1",
		}))
	}
	require.NoError(t, a.Record(context.Background(), Event{
		Action:     "vault.read",
		Result:     "denied",
		UserID:     "u2",
		ActorIP:    "5.6.7.8",
	}))
	require.NoError(t, a.Record(context.Background(), Event{
		Action:     "license.expired",
		Result:     "failure",
		UserID:     "u3",
		ActorIP:    "9.9.9.9",
	}))
}

func TestSearchByAction(t *testing.T) {
	db := newTestDB(t)
	a, err := New(context.Background(), db)
	require.NoError(t, err)
	seedRows(t, a)

	resp, err := a.Search(context.Background(), SearchRequest{
		Action: "domain.create",
		Limit:  100,
	}, "")
	require.NoError(t, err)
	require.Equal(t, int64(5), resp.Total)
	require.Len(t, resp.Rows, 5)
}

func TestSearchByResult(t *testing.T) {
	db := newTestDB(t)
	a, err := New(context.Background(), db)
	require.NoError(t, err)
	seedRows(t, a)

	resp, err := a.Search(context.Background(), SearchRequest{
		Result: "denied",
		Limit:  100,
	}, "")
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.Total)
	require.Equal(t, "vault.read", resp.Rows[0].Action)
}

func TestSearchPagination(t *testing.T) {
	db := newTestDB(t)
	a, err := New(context.Background(), db)
	require.NoError(t, err)
	seedRows(t, a)

	page1, err := a.Search(context.Background(), SearchRequest{Limit: 2, Offset: 0}, "")
	require.NoError(t, err)
	require.Len(t, page1.Rows, 2)
	require.Equal(t, 2, page1.NextOffset)

	page2, err := a.Search(context.Background(), SearchRequest{Limit: 2, Offset: 2}, "")
	require.NoError(t, err)
	require.Len(t, page2.Rows, 2)
}

func TestSearchTimeRange(t *testing.T) {
	db := newTestDB(t)
	a, err := New(context.Background(), db)
	require.NoError(t, err)
	seedRows(t, a)

	now := time.Now().UTC()
	future := now.Add(1 * time.Hour)
	resp, err := a.Search(context.Background(), SearchRequest{
		Until:  &future,
		Limit:  100,
	}, "")
	require.NoError(t, err)
	require.Equal(t, int64(7), resp.Total)

	past := now.Add(-1 * time.Hour)
	resp, err = a.Search(context.Background(), SearchRequest{
		Since: &past,
		Limit: 100,
	}, "")
	require.NoError(t, err)
	require.Equal(t, int64(7), resp.Total)
}

func TestSanitizeAction(t *testing.T) {
	require.Equal(t, "vaultread", SanitizeAction("vault%read"))
	require.Equal(t, "vaultread", SanitizeAction("vault_read"))
	require.Equal(t, "vault.read", SanitizeAction("  vault.read  "))
}

func TestFormatCEF(t *testing.T) {
	row := models.AuditEntry{
		Action:       "domain.create",
		Result:       "success",
		UserID:       "u1",
		UserEmail:    "u1@x",
		ActorIP:      "1.2.3.4",
		SessionID:    "sess-1",
		ResourceType: "domain",
		ResourceID:   "d-100",
		Timestamp:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	line := FormatCEF(row, "Acme", "Panel", "1.0.0")
	require.Contains(t, line, "CEF:0|Acme|Panel|1.0.0|domain.create|Domain create|3|")
	require.Contains(t, line, "src=1.2.3.4")
	require.Contains(t, line, "suser=u1")
	require.Contains(t, line, "outcome=success")
}

func TestFormatCEFEscapesPipes(t *testing.T) {
	row := models.AuditEntry{
		Action:    "user.note",
		Result:    "success",
		Detail:    "user said hi|bye",
		Timestamp: time.Now(),
	}
	line := FormatCEF(row, "", "", "")
	require.Contains(t, line, `user said hi\|bye`)
}

func TestFormatSyslog(t *testing.T) {
	frame := FormatSyslog("panel01", "CEF:0|V|P|1|x|y|5|z=1")
	require.True(t, strings.HasPrefix(frame, "<"))
	require.Contains(t, frame, "panel01")
	require.Contains(t, frame, "orvix-audit:")
	require.Contains(t, frame, "CEF:0|V|P|1|x|y|5|z=1")
}

func TestExportToFile(t *testing.T) {
	db := newTestDB(t)
	a, err := New(context.Background(), db)
	require.NoError(t, err)
	seedRows(t, a)

	path := filepath.Join(t.TempDir(), "audit.cef")
	resp, err := a.Export(context.Background(), ExportRequest{
		Transport: "file",
		FilePath:  path,
	})
	require.NoError(t, err)
	require.Equal(t, 7, resp.Exported)
	require.Equal(t, 0, resp.Errors)

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	require.Len(t, lines, 7)
	for _, ln := range lines {
		require.True(t, strings.HasPrefix(ln, "CEF:0|"))
	}
}

func TestExportToUDPLoopback(t *testing.T) {
	db := newTestDB(t)
	a, err := New(context.Background(), db)
	require.NoError(t, err)
	seedRows(t, a)

	// Start a UDP listener on an ephemeral port.
	addr, err := listenUDP(t)
	require.NoError(t, err)
	defer addr.Close()

	host, port, err := splitHostPort(addr.LocalAddr().String())
	require.NoError(t, err)

	resp, err := a.Export(context.Background(), ExportRequest{
		Transport: "syslog_udp",
		Host:      host,
		Port:      port,
	})
	require.NoError(t, err)
	require.Equal(t, 7, resp.Exported)
	require.Equal(t, 0, resp.Errors)

	// Read back at least one frame.
	addr.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 8192)
	n, _, err := addr.ReadFromUDP(buf)
	require.NoError(t, err)
	require.Greater(t, n, 0)
	require.Contains(t, string(buf[:n]), "CEF:0|")
}

func TestExportUnknownTransport(t *testing.T) {
	db := newTestDB(t)
	a, err := New(context.Background(), db)
	require.NoError(t, err)
	seedRows(t, a)

	_, err = a.Export(context.Background(), ExportRequest{Transport: "sneaker_net"})
	require.Error(t, err)
}
