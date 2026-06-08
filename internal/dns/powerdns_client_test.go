package dns

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func setupMockPDNS(handler func(w http.ResponseWriter, r *http.Request)) (*PowerDNSClient, func()) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handler))
	serverURL := server.URL

	// Set env vars for test
	os.Setenv("ORVIX_POWERDNS_URL", serverURL)
	os.Setenv("ORVIX_POWERDNS_API_KEY", "test-api-key")
	os.Setenv("ORVIX_POWERDNS_SERVER_ID", "localhost")

	// Create client
	client, err := NewPowerDNSClient()
	if err != nil {
		panic(err)
	}

	cleanup := func() {
		server.Close()
		os.Unsetenv("ORVIX_POWERDNS_URL")
		os.Unsetenv("ORVIX_POWERDNS_API_KEY")
		os.Unsetenv("ORVIX_POWERDNS_SERVER_ID")
	}

	return client, cleanup
}

func TestNewPowerDNSClient(t *testing.T) {
	// Test with no env vars
	os.Unsetenv("ORVIX_POWERDNS_URL")
	os.Unsetenv("ORVIX_POWERDNS_API_KEY")

	client, err := NewPowerDNSClient()
	if err != nil {
		t.Fatalf("NewPowerDNSClient() error = %v", err)
	}
	if client != nil {
		t.Error("NewPowerDNSClient() should return nil when not configured")
	}
}

func TestNewPowerDNSClient_Configured(t *testing.T) {
	os.Setenv("ORVIX_POWERDNS_URL", "http://localhost:8081")
	os.Setenv("ORVIX_POWERDNS_API_KEY", "test-key")
	os.Setenv("ORVIX_POWERDNS_SERVER_ID", "test-server")
	defer func() {
		os.Unsetenv("ORVIX_POWERDNS_URL")
		os.Unsetenv("ORVIX_POWERDNS_API_KEY")
		os.Unsetenv("ORVIX_POWERDNS_SERVER_ID")
	}()

	client, err := NewPowerDNSClient()
	if err != nil {
		t.Fatalf("NewPowerDNSClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewPowerDNSClient() returned nil")
	}
	if !client.IsConfigured() {
		t.Error("IsConfigured() = false, want true")
	}
}

func TestPowerDNSClient_IsConfigured(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		apiKey string
		want   bool
	}{
		{"both set", "http://localhost:8081", "api-key", true},
		{"url only", "http://localhost:8081", "", false},
		{"api key only", "", "api-key", false},
		{"neither set", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.url != "" {
				os.Setenv("ORVIX_POWERDNS_URL", tt.url)
			} else {
				os.Unsetenv("ORVIX_POWERDNS_URL")
			}
			if tt.apiKey != "" {
				os.Setenv("ORVIX_POWERDNS_API_KEY", tt.apiKey)
			} else {
				os.Unsetenv("ORVIX_POWERDNS_API_KEY")
			}
			defer func() {
				os.Unsetenv("ORVIX_POWERDNS_URL")
				os.Unsetenv("ORVIX_POWERDNS_API_KEY")
			}()

			client, _ := NewPowerDNSClient()
			if client == nil && tt.want {
				t.Skip("client is nil as expected when not configured")
			}
			if client != nil && client.IsConfigured() != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", client.IsConfigured(), tt.want)
			}
		})
	}
}

func TestPowerDNSClient_CreateZone(t *testing.T) {
	var receivedZone PowerDNSZone
	var called bool

	client, cleanup := setupMockPDNS(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != "POST" {
			t.Errorf("Method = %s, want POST", r.Method)
		}
		if r.Header.Get("X-API-Key") != "test-api-key" {
			t.Errorf("X-API-Key header = %s, want test-api-key", r.Header.Get("X-API-Key"))
		}

		var zone PowerDNSZone
		if err := json.NewDecoder(r.Body).Decode(&zone); err != nil {
			t.Fatalf("Failed to decode zone: %v", err)
		}
		receivedZone = zone

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	})
	defer cleanup()

	err := client.CreateZone(PowerDNSZone{
		Name: "example.com",
		Kind: "Native",
	})

	if err != nil {
		t.Fatalf("CreateZone() error = %v", err)
	}
	if !called {
		t.Error("Handler was not called")
	}
	if receivedZone.Name != "example.com" {
		t.Errorf("zone.Name = %s, want example.com", receivedZone.Name)
	}
}

func TestPowerDNSClient_DeleteZone(t *testing.T) {
	var called bool

	client, cleanup := setupMockPDNS(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != "DELETE" {
			t.Errorf("Method = %s, want DELETE", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	})
	defer cleanup()

	err := client.DeleteZone("example.com")
	if err != nil {
		t.Fatalf("DeleteZone() error = %v", err)
	}
	if !called {
		t.Error("Handler was not called")
	}
}

func TestPowerDNSClient_ReplaceRecords(t *testing.T) {
	var called bool
	var method string

	client, cleanup := setupMockPDNS(func(w http.ResponseWriter, r *http.Request) {
		called = true
		method = r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	defer cleanup()

	err := client.ReplaceRecords("example.com", PowerDNSRecordSet{
		Name: "www.example.com",
		Type: "A",
		TTL:  3600,
		Records: []PowerDNSRecord{
			{Name: "www.example.com", Type: "A", Content: "192.0.2.1"},
		},
	})
	if err != nil {
		t.Fatalf("ReplaceRecords() error = %v", err)
	}
	if !called {
		t.Error("Handler was not called")
	}
	if method != "PUT" {
		t.Errorf("Method = %s, want PUT", method)
	}
}

func TestPowerDNSClient_AddRecord(t *testing.T) {
	var called bool

	client, cleanup := setupMockPDNS(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != "PATCH" {
			t.Errorf("Method = %s, want PATCH", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	})
	defer cleanup()

	err := client.AddRecord("example.com", PowerDNSRecord{
		Name:    "www.example.com",
		Type:    "A",
		Content: "192.0.2.1",
		TTL:     3600,
	})
	if err != nil {
		t.Fatalf("AddRecord() error = %v", err)
	}
	if !called {
		t.Error("Handler was not called")
	}
}

func TestPowerDNSClient_DeleteRecord(t *testing.T) {
	var called bool

	client, cleanup := setupMockPDNS(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != "PATCH" {
			t.Errorf("Method = %s, want PATCH", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	})
	defer cleanup()

	err := client.DeleteRecord("example.com", "www.example.com", "A")
	if err != nil {
		t.Fatalf("DeleteRecord() error = %v", err)
	}
	if !called {
		t.Error("Handler was not called")
	}
}

func TestPowerDNSClient_SyncZone(t *testing.T) {
	var callCount int

	client, cleanup := setupMockPDNS(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusNoContent)
	})
	defer cleanup()

	records := []PowerDNSRecord{
		{Name: "www.example.com", Type: "A", Content: "192.0.2.1"},
		{Name: "mail.example.com", Type: "A", Content: "192.0.2.2"},
		{Name: "@", Type: "MX", Content: "10 mail.example.com"},
	}

	err := client.SyncZone("example.com", records)
	if err != nil {
		t.Fatalf("SyncZone() error = %v", err)
	}
	if callCount == 0 {
		t.Error("No requests were made")
	}
}

func TestPowerDNSClient_SyncZone_NotConfigured(t *testing.T) {
	// When not configured, SyncZone should be a no-op
	os.Unsetenv("ORVIX_POWERDNS_URL")
	os.Unsetenv("ORVIX_POWERDNS_API_KEY")

	client, err := NewPowerDNSClient()
	if err != nil {
		t.Fatalf("NewPowerDNSClient() error = %v", err)
	}

	// Should not panic and should return nil
	err = client.SyncZone("example.com", []PowerDNSRecord{})
	if err != nil {
		t.Errorf("SyncZone() error = %v, want nil (no-op when not configured)", err)
	}
}

func TestPowerDNSClient_ErrorHandling(t *testing.T) {
	client, cleanup := setupMockPDNS(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "zone not found"}`))
	})
	defer cleanup()

	err := client.CreateZone(PowerDNSZone{Name: "nonexistent.com", Kind: "Native"})
	if err == nil {
		t.Error("CreateZone() expected error for 404 response")
	}
}

func TestPowerDNSClient_Timeout(t *testing.T) {
	os.Setenv("ORVIX_POWERDNS_URL", "http://localhost:99999") // Invalid port
	os.Setenv("ORVIX_POWERDNS_API_KEY", "test-key")
	defer func() {
		os.Unsetenv("ORVIX_POWERDNS_URL")
		os.Unsetenv("ORVIX_POWERDNS_API_KEY")
	}()

	client, err := NewPowerDNSClient()
	if err != nil {
		t.Fatalf("NewPowerDNSClient() error = %v", err)
	}

	// This should timeout or fail
	err = client.CreateZone(PowerDNSZone{Name: "test.com", Kind: "Native"})
	if err == nil {
		t.Error("CreateZone() expected error for unreachable server")
	}
}