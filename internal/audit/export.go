// Package audit — CEF export (v0.3.0).
//
// CEF (Common Event Format) is an ArcSight / Splunk standard:
//   CEF:0|Device Vendor|Device Product|Device Version|Signature ID|Name|Severity|Extension
//
// The exporter walks the audit chain (or a filtered subset) and
// writes one CEF-formatted line per row, either to a file or over
// syslog (UDP/TCP).
//
// Reference: https://www.microfocus.com/documentation/arcsight/arcsight-smart-connectors/
package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"gorm.io/gorm"
)

// ExportRequest is the body for POST /admin/audit-log/export.
type ExportRequest struct {
	Format        string     `json:"format"`                     // "cef" | "csv" | "json" (default: "cef")
	Transport     string     `json:"transport"`                  // "syslog_udp" | "syslog_tcp" | "file"
	Host          string     `json:"host,omitempty"`
	Port          int        `json:"port,omitempty"`
	FilePath      string     `json:"file_path,omitempty"`        // for transport=file
	Since         *time.Time `json:"since,omitempty"`
	Until         *time.Time `json:"until,omitempty"`
	DeviceVendor  string     `json:"device_vendor,omitempty"`
	DeviceProduct string     `json:"device_product,omitempty"`
	MaxRows       int        `json:"max_rows,omitempty"`         // safety cap; default 10000
	// Filters
	Action       string `json:"action,omitempty"`        // filter by action (prefix match)
	Result       string `json:"result,omitempty"`        // filter by result (success|failure|denied)
	ResourceType string `json:"resource_type,omitempty"` // filter by resource type
	UserID       string `json:"user_id,omitempty"`       // filter by user ID
	TenantID     string `json:"tenant_id,omitempty"`     // filter by tenant ID (for multi-tenant)
	Search       string `json:"search,omitempty"`         // full-text search across detail field
}

// ExportResponse is the result envelope.
type ExportResponse struct {
	Exported    int       `json:"exported"`
	Transport   string    `json:"transport"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
	Errors      int       `json:"errors"`
	FirstError  string    `json:"first_error,omitempty"`
}

// CEFVendor / CEFProduct / CEFVersion are the device identity
// baked into every line. Operators can override via ExportRequest.
const (
	CEFVersion  = "0.3.0"
	CEFVendor   = "Orvix"
	CEFProduct  = "Panel"
	SyslogTag   = "orvix-audit"
	Facility    = 1 // LOG_AUTH
	SeverityMap = "5" // default
)

// FormatCEF serializes one audit row as a CEF line. The result does
// not include a trailing newline.
func FormatCEF(r models.AuditEntry, vendor, product, version string) string {
	if vendor == "" {
		vendor = CEFVendor
	}
	if product == "" {
		product = CEFProduct
	}
	if version == "" {
		version = CEFVersion
	}
	signatureID := r.Action
	name := humanizeAction(r.Action)
	severity := severityFromResult(r.Result)
	ts := r.Timestamp.UnixMilli()

	ext := []string{
		fmt.Sprintf("rt=%d", ts),
		fmt.Sprintf("dvchost=orvixpanel"),
	}
	if r.UserID != "" {
		ext = append(ext, fmt.Sprintf("suser=%s", escapeCEF(r.UserID)))
	}
	if r.UserEmail != "" {
		ext = append(ext, fmt.Sprintf("suid=%s", escapeCEF(r.UserEmail)))
	}
	if r.ActorIP != "" {
		ext = append(ext, fmt.Sprintf("src=%s", escapeCEF(r.ActorIP)))
	}
	if r.SessionID != "" {
		ext = append(ext, fmt.Sprintf("cs1=%s cs1Label=session_id", escapeCEF(r.SessionID)))
	}
	if r.ResourceType != "" {
		ext = append(ext, fmt.Sprintf("cs2=%s cs2Label=resource_type", escapeCEF(r.ResourceType)))
	}
	if r.ResourceID != "" {
		ext = append(ext, fmt.Sprintf("cs3=%s cs3Label=resource_id", escapeCEF(r.ResourceID)))
	}
	if r.ResourceName != "" {
		ext = append(ext, fmt.Sprintf("cs4=%s cs4Label=resource_name", escapeCEF(r.ResourceName)))
	}
	ext = append(ext, fmt.Sprintf("outcome=%s", escapeCEF(r.Result)))
	if r.DurationMS > 0 {
		ext = append(ext, fmt.Sprintf("cat=action duration=%d", r.DurationMS))
	}
	if r.Detail != "" {
		ext = append(ext, fmt.Sprintf("msg=%s", escapeCEF(r.Detail)))
	}

	return fmt.Sprintf("CEF:0|%s|%s|%s|%s|%s|%s|%s",
		vendor, product, version, signatureID, name, severity,
		strings.Join(ext, " "))
}

// FormatSyslog wraps a CEF line in a minimal RFC 3164 syslog
// frame: <PRI>TIMESTAMP HOST TAG MSG
func FormatSyslog(host, cefLine string) string {
	pri := Facility*8 + 6 // info
	ts := time.Now().UTC().Format("Jan _2 15:04:05")
	return fmt.Sprintf("<%d>%s %s %s: %s", pri, ts, host, SyslogTag, cefLine)
}

// CSVHeader is the header row for CSV export.
const CSVHeader = "id,timestamp,user_id,user_email,user_role,actor_ip,session_id,action,resource_type,resource_id,resource_name,result,duration_ms,detail"

// FormatCSV serializes one audit row as a CSV line.
func FormatCSV(r models.AuditEntry) string {
	return fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%d,%s",
		escapeCSV(r.ID),
		r.Timestamp.Format(time.RFC3339),
		escapeCSV(r.UserID),
		escapeCSV(r.UserEmail),
		escapeCSV(r.UserRole),
		escapeCSV(r.ActorIP),
		escapeCSV(r.SessionID),
		escapeCSV(r.Action),
		escapeCSV(r.ResourceType),
		escapeCSV(r.ResourceID),
		escapeCSV(r.ResourceName),
		escapeCSV(r.Result),
		r.DurationMS,
		escapeCSV(r.Detail),
	)
}

// ExportCSV exports all rows as CSV format to the writer.
// Returns the number of rows exported and any error.
func (a *Auditor) ExportCSV(ctx context.Context, w func(string) error, req ExportRequest) (int, error) {
	q := a.db.WithContext(ctx).Model(&models.AuditEntry{})
	applyFilters(q, req)
	var rows []models.AuditEntry
	if err := q.Order("timestamp ASC").Limit(req.MaxRows).Find(&rows).Error; err != nil {
		return 0, fmt.Errorf("query rows: %w", err)
	}
	if err := w(CSVHeader); err != nil {
		return 0, err
	}
	count := 0
	for _, r := range rows {
		if err := w(FormatCSV(r)); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// ExportJSON exports all rows as JSON Lines format to the writer.
// Each line is a valid JSON object representing one audit entry.
func (a *Auditor) ExportJSON(ctx context.Context, w func(string) error, req ExportRequest) (int, error) {
	q := a.db.WithContext(ctx).Model(&models.AuditEntry{})
	applyFilters(q, req)
	var rows []models.AuditEntry
	if err := q.Order("timestamp ASC").Limit(req.MaxRows).Find(&rows).Error; err != nil {
		return 0, fmt.Errorf("query rows: %w", err)
	}
	count := 0
	for _, r := range rows {
		line, err := formatJSONEntry(r)
		if err != nil {
			return count, fmt.Errorf("format row: %w", err)
		}
		if err := w(line); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// formatJSONEntry converts an audit entry to a JSON line.
func formatJSONEntry(r models.AuditEntry) (string, error) {
	type jsonEntry struct {
		ID           string `json:"id"`
		Timestamp    string `json:"timestamp"`
		UserID       string `json:"user_id,omitempty"`
		UserEmail    string `json:"user_email,omitempty"`
		UserRole     string `json:"user_role,omitempty"`
		ActorIP      string `json:"actor_ip,omitempty"`
		SessionID    string `json:"session_id,omitempty"`
		Action       string `json:"action"`
		ResourceType string `json:"resource_type,omitempty"`
		ResourceID   string `json:"resource_id,omitempty"`
		ResourceName string `json:"resource_name,omitempty"`
		Result       string `json:"result"`
		DurationMS   int    `json:"duration_ms,omitempty"`
		Detail       string `json:"detail,omitempty"`
	}
	entry := jsonEntry{
		ID:           r.ID,
		Timestamp:    r.Timestamp.Format(time.RFC3339),
		UserID:       r.UserID,
		UserEmail:    r.UserEmail,
		UserRole:     r.UserRole,
		ActorIP:      r.ActorIP,
		SessionID:    r.SessionID,
		Action:       r.Action,
		ResourceType: r.ResourceType,
		ResourceID:   r.ResourceID,
		ResourceName: r.ResourceName,
		Result:       r.Result,
		DurationMS:   r.DurationMS,
		Detail:       r.Detail,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Export runs the export. Returns the result envelope + the first
// error encountered (if any). The audit chain is unaffected.
func (a *Auditor) Export(ctx context.Context, req ExportRequest) (*ExportResponse, error) {
	resp := &ExportResponse{
		Transport:  req.Transport,
		StartedAt:  time.Now().UTC(),
	}
	if req.MaxRows <= 0 {
		req.MaxRows = 10000
	}
	// Default format is CEF
	format := req.Format
	if format == "" {
		format = "cef"
	}

	// 1. Wire transport.
	var write func(line string) error
	switch req.Transport {
	case "file":
		if req.FilePath == "" {
			return resp, fmt.Errorf("file_path required for transport=file")
		}
		f, err := os.OpenFile(req.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return resp, fmt.Errorf("open file: %w", err)
		}
		defer f.Close()
		w := bufio.NewWriter(f)
		defer w.Flush()
		write = func(line string) error {
			_, err := w.WriteString(line + "\n")
			return err
		}
	case "syslog_udp":
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", req.Host, req.Port))
		if err != nil {
			return resp, fmt.Errorf("resolve udp: %w", err)
		}
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			return resp, fmt.Errorf("dial udp: %w", err)
		}
		defer conn.Close()
		host, _ := os.Hostname()
		write = func(line string) error {
			frame := FormatSyslog(host, line)
			_, err := conn.Write([]byte(frame))
			return err
		}
	case "syslog_tcp":
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", req.Host, req.Port))
		if err != nil {
			return resp, fmt.Errorf("resolve tcp: %w", err)
		}
		conn, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			return resp, fmt.Errorf("dial tcp: %w", err)
		}
		defer conn.Close()
		w := bufio.NewWriter(conn)
		defer w.Flush()
		write = func(line string) error {
			_, err := w.WriteString(line + "\n")
			return err
		}
	default:
		return resp, fmt.Errorf("unknown transport: %q", req.Transport)
	}

	// 2. Export based on format.
	switch format {
	case "csv":
		count, err := a.ExportCSV(ctx, write, req)
		resp.Exported = count
		if err != nil {
			resp.FirstError = err.Error()
		}
	case "json":
		count, err := a.ExportJSON(ctx, write, req)
		resp.Exported = count
		if err != nil {
			resp.FirstError = err.Error()
		}
	case "cef":
		fallthrough
	default:
		// CEF export with filters
		q := a.db.WithContext(ctx).Model(&models.AuditEntry{})
		applyFilters(q, req)
		var rows []models.AuditEntry
		if err := q.Order("timestamp ASC").Limit(req.MaxRows).Find(&rows).Error; err != nil {
			return resp, fmt.Errorf("query rows: %w", err)
		}
		for _, r := range rows {
			cef := FormatCEF(r, req.DeviceVendor, req.DeviceProduct, CEFVersion)
			if err := write(cef); err != nil {
				resp.Errors++
				if resp.FirstError == "" {
					resp.FirstError = err.Error()
				}
				continue
			}
			resp.Exported++
		}
	}
	resp.FinishedAt = time.Now().UTC()
	return resp, nil
}

// -----------------------------------------------------------------------------
// CEF helpers
// -----------------------------------------------------------------------------

// escapeCEF escapes | = \ and newlines per the CEF spec.
func escapeCEF(s string) string {
	r := strings.NewReplacer(
		"\\", "\\\\",
		"|", "\\|",
		"=", "\\=",
		"\n", "\\n",
		"\r", "\\r",
	)
	return r.Replace(s)
}

// humanizeAction turns "vault.read" into "Vault read" for the CEF
// `Name` field. Cosmetic.
func humanizeAction(s string) string {
	if s == "" {
		return "Unknown action"
	}
	out := strings.ReplaceAll(s, ".", " ")
	if len(out) > 0 {
		out = strings.ToUpper(out[:1]) + out[1:]
	}
	return out
}

// severityFromResult maps our result string to the CEF severity
// scale (0=highest, 10=lowest).
func severityFromResult(r string) string {
	switch r {
	case "success":
		return "3"
	case "failure":
		return "7"
	case "denied":
		return "5"
	}
	return "5"
}

// escapeCSV escapes a string for CSV output.
// It wraps in quotes if necessary and escapes internal quotes.
func escapeCSV(s string) string {
	if s == "" {
		return ""
	}
	needsQuotes := strings.ContainsAny(s, `",`+"\n") || strings.HasPrefix(s, " ")
	if !needsQuotes {
		return s
	}
	// Escape internal quotes by doubling them
	s = strings.ReplaceAll(s, `"`, `""`)
	return `"` + s + `"`
}

// applyFilters applies the filter options from ExportRequest to the query.
func applyFilters(q *gorm.DB, req ExportRequest) {
	if req.Since != nil {
		q = q.Where("timestamp >= ?", *req.Since)
	}
	if req.Until != nil {
		q = q.Where("timestamp <= ?", *req.Until)
	}
	if req.Action != "" {
		q = q.Where("action LIKE ?", req.Action+"%")
	}
	if req.Result != "" {
		q = q.Where("result = ?", req.Result)
	}
	if req.ResourceType != "" {
		q = q.Where("resource_type = ?", req.ResourceType)
	}
	if req.UserID != "" {
		q = q.Where("user_id = ?", req.UserID)
	}
	if req.Search != "" {
		q = q.Where("detail LIKE ?", "%"+req.Search+"%")
	}
}
