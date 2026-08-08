package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"geoip-for-cn/internal/geodb"
)

func TestVerifyDatabase(t *testing.T) {
	databasePath := createTestDatabase(t)
	checks := []ipCheck{
		{"mainland IPv4", "114.114.114.114", true},
		{"mainland IPv6", "2400:3200::1", true},
		{"another mainland IPv6", "2400:da00::6666", true},
		{"Hong Kong IPv4", "203.198.7.66", false},
		{"Hong Kong IPv6", "2001:2e0::1", false},
		{"Taiwan IPv4", "1.34.0.1", false},
		{"Taiwan IPv6", "2001:288::1", false},
		{"Macao IPv4", "27.109.128.1", false},
		{"Macao IPv6", "2001:f90::1", false},
		{"global IPv6", "2606:4700:4700::1111", false},
		{"private IPv4", "192.168.1.1", false},
		{"private IPv6", "fd00::1", false},
	}
	for _, ipVersion := range []int{0, 4, 6} {
		if err := verifyDatabase(databasePath, ipVersion, checks); err != nil {
			t.Fatalf("IPv%d verification failed: %v", ipVersion, err)
		}
	}
}

func TestVerifyDatabaseDetectsUnexpectedMatch(t *testing.T) {
	databasePath := createTestDatabase(t)
	err := verifyDatabase(databasePath, 4, []ipCheck{{"wrong expectation", "114.114.114.114", false}})
	if err == nil || !strings.Contains(err.Error(), "want false") {
		t.Fatalf("error = %v, want mismatch error", err)
	}
}

func TestVerifyDatabaseRejectsInvalidInput(t *testing.T) {
	databasePath := createTestDatabase(t)
	tests := []struct {
		name      string
		ipVersion int
		checks    []ipCheck
		wantError string
	}{
		{"unsupported family", 5, nil, "unsupported IP version"},
		{"invalid address", 0, []ipCheck{{"invalid", "not-an-ip", false}}, "invalid IP"},
		{"no matching checks", 6, []ipCheck{{"IPv4 only", "192.0.2.1", false}}, "no IPv6 checks"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := verifyDatabase(databasePath, test.ipVersion, test.checks)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want error containing %q", err, test.wantError)
			}
		})
	}
}

func createTestDatabase(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	source := filepath.Join(dir, "cidrs.txt")
	database := filepath.Join(dir, "Country.mmdb")
	if err := os.WriteFile(source, []byte("114.114.114.0/24\n2400:3200::/32\n2400:da00::/32\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := geodb.Generate(source, database, geodb.Options{
		DatabaseType: "GeoIP2-Country",
		IPVersion:    6,
	}); err != nil {
		t.Fatal(err)
	}
	return database
}
