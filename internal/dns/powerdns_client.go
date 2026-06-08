// Package dns provides DNS zone and record management with optional
// PowerDNS integration.
package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// PowerDNSConfig holds PowerDNS API connection settings.
type PowerDNSConfig struct {
	URL      string // e.g., "http://127.0.0.1:8081"
	APIKey   string
	ServerID string // e.g., "localhost"
	Timeout  time.Duration
}

// PowerDNSClient wraps the PowerDNS API.
type PowerDNSClient struct {
	config   PowerDNSConfig
	client   *http.Client
	baseURL  string
}

// NewPowerDNSClient creates a new PowerDNS client if configured.
func NewPowerDNSClient() (*PowerDNSClient, error) {
	url := os.Getenv("ORVIX_POWERDNS_URL")
	apiKey := os.Getenv("ORVIX_POWERDNS_API_KEY")

	if url == "" || apiKey == "" {
		return nil, nil // Not configured, will run in local-only mode
	}

	serverID := os.Getenv("ORVIX_POWERDNS_SERVER_ID")
	if serverID == "" {
		serverID = "localhost"
	}

	config := PowerDNSConfig{
		URL:      url,
		APIKey:   apiKey,
		ServerID: serverID,
		Timeout:  10 * time.Second,
	}

	return &PowerDNSClient{
		config:  config,
		client:  &http.Client{Timeout: config.Timeout},
		baseURL: fmt.Sprintf("%s/api/v1/servers/%s", url, serverID),
	}, nil
}

// IsConfigured returns true if PowerDNS is properly configured.
func (c *PowerDNSClient) IsConfigured() bool {
	return c != nil && c.config.URL != "" && c.config.APIKey != ""
}

// doRequest performs an authenticated HTTP request to PowerDNS.
func (c *PowerDNSClient) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("powerdns error %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}

// PowerDNSZone represents a zone in PowerDNS API format.
type PowerDNSZone struct {
	Name    string   `json:"name"`
	Kind    string   `json:"kind"` // Native, Master, Slave
	Masters []string `json:"masters,omitempty"`
}

// CreateZone creates a zone in PowerDNS.
func (c *PowerDNSClient) CreateZone(zone PowerDNSZone) error {
	if !c.IsConfigured() {
		return fmt.Errorf("powerdns not configured")
	}
	_, err := c.doRequest("POST", "/zones", zone)
	return err
}

// DeleteZone deletes a zone from PowerDNS.
func (c *PowerDNSClient) DeleteZone(zoneName string) error {
	if !c.IsConfigured() {
		return fmt.Errorf("powerdns not configured")
	}
	_, err := c.doRequest("DELETE", fmt.Sprintf("/zones/%s", zoneName), nil)
	return err
}

// PowerDNSRecord represents a record in PowerDNS API format.
type PowerDNSRecord struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl,omitempty"`
	Disabled bool  `json:"disabled,omitempty"`
}

// PowerDNSRecordSet represents a set of records in PowerDNS API.
type PowerDNSRecordSet struct {
	Name    string          `json:"name"`
	Type    string          `json:"type"`
	TTL     int             `json:"ttl,omitempty"`
	Records []PowerDNSRecord `json:"records"`
}

// ReplaceRecords replaces all records of a given type in a zone.
func (c *PowerDNSClient) ReplaceRecords(zoneName string, recordSet PowerDNSRecordSet) error {
	if !c.IsConfigured() {
		return fmt.Errorf("powerdns not configured")
	}
	path := fmt.Sprintf("/zones/%s/%s/%s", zoneName, recordSet.Type, recordSet.Name)
	_, err := c.doRequest("PUT", path, recordSet.Records)
	return err
}

// AddRecord adds a single record to a zone.
func (c *PowerDNSClient) AddRecord(zoneName string, record PowerDNSRecord) error {
	if !c.IsConfigured() {
		return fmt.Errorf("powerdns not configured")
	}
	path := fmt.Sprintf("/zones/%s", zoneName)
	_, err := c.doRequest("PATCH", path, map[string]interface{}{
		"rrsets": []map[string]interface{}{
			{
				"name":   record.Name,
				"type":   record.Type,
				"ttl":    record.TTL,
				"changetype": "REPLACE",
				"records": []map[string]interface{}{
					{"content": record.Content, "disabled": record.Disabled},
				},
			},
		},
	})
	return err
}

// DeleteRecord removes a record from a zone.
func (c *PowerDNSClient) DeleteRecord(zoneName string, recordName, recordType string) error {
	if !c.IsConfigured() {
		return fmt.Errorf("powerdns not configured")
	}
	path := fmt.Sprintf("/zones/%s", zoneName)
	_, err := c.doRequest("PATCH", path, map[string]interface{}{
		"rrsets": []map[string]interface{}{
			{
				"name":      recordName,
				"type":      recordType,
				"changetype": "DELETE",
			},
		},
	})
	return err
}

// SyncZone syncs a local zone to PowerDNS.
func (c *PowerDNSClient) SyncZone(zoneName string, records []PowerDNSRecord) error {
	if !c.IsConfigured() {
		return nil // No-op if not configured
	}

	// Group records by type
	typeRecords := make(map[string][]PowerDNSRecord)
	for _, r := range records {
		typeRecords[r.Type] = append(typeRecords[r.Type], r)
	}

	// Sync each record type
	for recType, recs := range typeRecords {
		recordSet := PowerDNSRecordSet{
			Name:    zoneName,
			Type:    recType,
			TTL:     3600,
			Records: recs,
		}
		if err := c.ReplaceRecords(zoneName, recordSet); err != nil {
			return fmt.Errorf("sync %s records: %w", recType, err)
		}
	}

	return nil
}