package geoip

import "testing"

func TestLookupCountryISO(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		wantCode string
		wantOK   bool
	}{
		{name: "mainland china public dns (114DNS)", ip: "114.114.114.114", wantCode: "CN", wantOK: true},
		{name: "google public dns", ip: "8.8.8.8", wantCode: "US", wantOK: true},
		{name: "private ip", ip: "192.168.1.1", wantOK: false},
		{name: "loopback", ip: "127.0.0.1", wantOK: false},
		{name: "invalid", ip: "not-an-ip", wantOK: false},
		{name: "empty", ip: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, ok := LookupCountryISO(tt.ip)
			if ok != tt.wantOK {
				t.Fatalf("LookupCountryISO(%q) ok = %v, want %v (code=%q)", tt.ip, ok, tt.wantOK, code)
			}
			if tt.wantOK && code != tt.wantCode {
				t.Fatalf("LookupCountryISO(%q) code = %q, want %q", tt.ip, code, tt.wantCode)
			}
		})
	}
}
