package dns

import (
	"testing"
)

func TestValidateZoneDomain(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		{"valid simple", "example.com", false},
		{"valid subdomain", "www.example.com", false},
		{"valid multi-level", "api.v2.example.com", false},
		{"valid with numbers", "test123.example.com", false},
		{"valid with hyphen", "my-site.example.com", false},
		{"empty", "", true},
		{"too long", "a." + string(make([]byte, 250)) + ".com", true},
		{"invalid chars", "example!.com", true},
		{"starts with hyphen", "-example.com", true},
		{"ends with hyphen", "example-.com", true},
		{"double dot", "example..com", true},
		{"trailing dot", "example.com.", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateZoneDomain(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateZoneDomain() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRecordType(t *testing.T) {
	tests := []struct {
		name       string
		recordType string
		wantErr    bool
	}{
		{"A", "A", false},
		{"AAAA", "AAAA", false},
		{"CNAME", "CNAME", false},
		{"MX", "MX", false},
		{"TXT", "TXT", false},
		{"NS", "NS", false},
		{"SRV", "SRV", false},
		{"CAA", "CAA", false},
		{"lowercase a", "a", false},
		{"invalid type", "TYPE123", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecordType(tt.recordType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecordType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRecordName(t *testing.T) {
	tests := []struct {
		name    string
		rname   string
		wantErr bool
	}{
		{"apex", "@", false},
		{"wildcard", "*", false},
		{"simple", "www", false},
		{"subdomain", "mail", false},
		{"empty", "", true},
		{"too long label", "a" + string(make([]byte, 65)), true},
		{"invalid chars", "www!", true},
		{"starts with hyphen", "-www", true},
		{"ends with hyphen", "www-", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecordName(tt.rname)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecordName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTTL(t *testing.T) {
	tests := []struct {
		name    string
		ttl     int
		wantErr bool
	}{
		{"minimum valid", 60, false},
		{"maximum valid", 86400, false},
		{"default common", 3600, false},
		{"below minimum", 59, true},
		{"above maximum", 86401, true},
		{"zero", 0, true},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTTL(tt.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTTL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		wantErr  bool
	}{
		{"zero", 0, false},
		{"common", 10, false},
		{"max valid", 65535, false},
		{"above max", 65536, true},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePriority(tt.priority)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePriority() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRecordContent_A(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"valid IPv4", "192.0.2.1", false},
		{"valid IPv4 private", "10.0.0.1", false},
		{"valid IPv4 loopback", "127.0.0.1", false},
		{"invalid IPv4", "192.0.2.256", true},
		{"IPv6 instead", "::1", true},
		{"hostname instead", "example.com", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecordContent("A", tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecordContent(A) error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRecordContent_AAAA(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"valid IPv6", "2001:db8::1", false},
		{"valid full IPv6", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false},
		{"loopback", "::1", false},
		{"IPv4 instead", "192.0.2.1", true},
		{"invalid", "not-an-ip", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecordContent("AAAA", tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecordContent(AAAA) error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRecordContent_MX(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"valid", "10 mail.example.com", false},
		{"high priority", "0 mail.example.com", false},
		{"low priority", "100 mail.example.com", false},
		{"no space", "mailexample.com", true},
		{"invalid hostname", "10 mail!.example.com", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecordContent("MX", tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecordContent(MX) error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRecordContent_TXT(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"simple", "v=spf1 include:_spf.example.com ~all", false},
		{"short", "hello", false},
		{"max length", string(make([]byte, 400)), false},
		{"too long", string(make([]byte, 500)), true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecordContent("TXT", tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecordContent(TXT) error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRecordContent_CNAME(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"valid", "example.com", false},
		{"subdomain", "www.example.com", false},
		{"trailing dot", "example.com.", false},
		{"empty", "", true},
		{"invalid", "example!.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecordContent("CNAME", tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecordContent(CNAME) error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRecordContent_SRV(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"valid", "0 5 443 sip.example.com", false},
		{"zero priority", "0 0 443 service.example.com", false},
		{"service unavailable", "0 0 443 .", false},
		{"missing parts", "0 5 443", true},
		{"too many parts", "0 5 443 host extra", true},
		{"invalid port", "0 5 70000 host", true},
		{"invalid priority", "70000 5 443 host", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecordContent("SRV", tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecordContent(SRV) error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRecordContent_CAA(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"valid issue", "0 issue \"letsencrypt.org\"", false},
		{"valid issuewild", "128 issuewild \"ca.example.com\"", false},
		{"valid iodef", "0 iodef \"mailto:security@example.com\"", false},
		{"invalid tag", "0 invalid \"value\"", true},
		{"invalid flags", "256 issue \"value\"", true},
		{"missing parts", "0 issue", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecordContent("CAA", tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecordContent(CAA) error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRecord(t *testing.T) {
	tests := []struct {
		name   string
		record RecordDefinition
		wantErr bool
	}{
		{
			name: "valid A record",
			record: RecordDefinition{
				Name: "www", Type: "A", Content: "192.0.2.1", TTL: 3600, Priority: 0,
			},
			wantErr: false,
		},
		{
			name: "valid MX record",
			record: RecordDefinition{
				Name: "@", Type: "MX", Content: "10 mail.example.com", TTL: 3600, Priority: 10,
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			record: RecordDefinition{
				Name: "www", Type: "INVALID", Content: "192.0.2.1", TTL: 3600, Priority: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid content for type",
			record: RecordDefinition{
				Name: "www", Type: "A", Content: "not-an-ip", TTL: 3600, Priority: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecord(tt.record)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecord() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateZone(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		{"valid", "example.com", false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateZone(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateZone() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}