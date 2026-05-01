package database

import (
	"net/netip"
	"testing"
)

func TestNullableString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *string
	}{
		{
			name:  "empty string returns nil",
			input: "",
			want:  nil,
		},
		{
			name:  "non-empty string returns pointer",
			input: "test",
			want:  stringPtr("test"),
		},
		{
			name:  "whitespace string returns pointer",
			input: "  ",
			want:  stringPtr("  "),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nullableString(tt.input)
			if (got == nil) != (tt.want == nil) {
				t.Errorf("nullableString() = %v, want %v", got, tt.want)
				return
			}
			if got != nil && *got != *tt.want {
				t.Errorf("nullableString() = %v, want %v", *got, *tt.want)
			}
		})
	}
}

func TestDerefString(t *testing.T) {
	tests := []struct {
		name  string
		input *string
		want  string
	}{
		{
			name:  "nil returns empty string",
			input: nil,
			want:  "",
		},
		{
			name:  "pointer to string returns value",
			input: stringPtr("test"),
			want:  "test",
		},
		{
			name:  "pointer to empty string returns empty",
			input: stringPtr(""),
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := derefString(tt.input)
			if got != tt.want {
				t.Errorf("derefString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseIPToInet(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		wantErr bool
		wantIP  string
	}{
		{
			name:    "empty string returns nil",
			input:   "",
			wantNil: true,
			wantErr: false,
		},
		{
			name:    "valid IPv4",
			input:   "192.168.1.1",
			wantNil: false,
			wantErr: false,
			wantIP:  "192.168.1.1",
		},
		{
			name:    "valid IPv6",
			input:   "2001:db8::1",
			wantNil: false,
			wantErr: false,
			wantIP:  "2001:db8::1",
		},
		{
			name:    "invalid IP returns error",
			input:   "not-an-ip",
			wantNil: true,
			wantErr: true,
		},
		{
			name:    "invalid format returns error",
			input:   "999.999.999.999",
			wantNil: true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIPToInet(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIPToInet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got == nil) != tt.wantNil {
				t.Errorf("parseIPToInet() got nil = %v, want nil = %v", got == nil, tt.wantNil)
				return
			}
			if !tt.wantNil && !tt.wantErr {
				if got.Addr().String() != tt.wantIP {
					t.Errorf("parseIPToInet() IP = %v, want %v", got.Addr().String(), tt.wantIP)
				}
				// Verify prefix length
				addr, _ := netip.ParseAddr(tt.input)
				expectedBits := 32
				if addr.Is6() {
					expectedBits = 128
				}
				if got.Bits() != expectedBits {
					t.Errorf("parseIPToInet() prefix bits = %v, want %v", got.Bits(), expectedBits)
				}
			}
		})
	}
}

func TestInetToString(t *testing.T) {
	tests := []struct {
		name  string
		input *netip.Prefix
		want  string
	}{
		{
			name:  "nil returns empty string",
			input: nil,
			want:  "",
		},
		{
			name:  "IPv4 prefix returns IP string",
			input: prefixPtr("192.168.1.1/32"),
			want:  "192.168.1.1",
		},
		{
			name:  "IPv6 prefix returns IP string",
			input: prefixPtr("2001:db8::1/128"),
			want:  "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inetToString(tt.input)
			if got != tt.want {
				t.Errorf("inetToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundTripIPConversion(t *testing.T) {
	tests := []struct {
		name string
		ip   string
	}{
		{"IPv4", "10.0.0.1"},
		{"IPv6", "fe80::1"},
		{"localhost IPv4", "127.0.0.1"},
		{"localhost IPv6", "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// String -> Prefix
			prefix, err := parseIPToInet(tt.ip)
			if err != nil {
				t.Fatalf("parseIPToInet() error = %v", err)
			}

			// Prefix -> String
			got := inetToString(prefix)
			if got != tt.ip {
				t.Errorf("round trip failed: got %v, want %v", got, tt.ip)
			}
		})
	}
}

// Helper functions for tests
func stringPtr(s string) *string {
	return &s
}

func prefixPtr(s string) *netip.Prefix {
	p := netip.MustParsePrefix(s)
	return &p
}
