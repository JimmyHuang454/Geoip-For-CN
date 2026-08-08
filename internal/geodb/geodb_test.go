package geodb

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/oschwald/maxminddb-golang"
)

type countryRecord struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

func TestGenerateAddressFamilies(t *testing.T) {
	dir := t.TempDir()
	ipv4Source := filepath.Join(dir, "ipv4.txt")
	ipv6Source := filepath.Join(dir, "ipv6.txt")
	dualSource := filepath.Join(dir, "dual.txt")
	if err := os.WriteFile(ipv4Source, []byte("1.1.1.0/24\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ipv6Source, []byte("2400:3200::/32\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dualSource, []byte("1.1.1.0/24\n2400:3200::/32\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name                string
		source              string
		ipVersion           int
		disableIPv4Aliasing bool
		wantIPv4            bool
		wantIPv6            bool
	}{
		{"IPv4 only", ipv4Source, 4, false, true, false},
		{"dual stack", dualSource, 6, false, true, true},
		{"IPv6 only", ipv6Source, 6, true, false, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			destination := filepath.Join(dir, test.name+".mmdb")
			if err := Generate(test.source, destination, Options{
				DatabaseType:        "GeoIP2-Country",
				IPVersion:           test.ipVersion,
				DisableIPv4Aliasing: test.disableIPv4Aliasing,
			}); err != nil {
				t.Fatal(err)
			}
			database, err := maxminddb.Open(destination)
			if err != nil {
				t.Fatal(err)
			}
			defer database.Close()
			if database.Metadata.IPVersion != uint(test.ipVersion) {
				t.Fatalf("IP version = %d, want %d", database.Metadata.IPVersion, test.ipVersion)
			}
			assertLookup(t, database, "1.1.1.1", test.wantIPv4)
			assertLookup(t, database, "2400:3200::1", test.wantIPv6)
		})
	}
}

func assertLookup(t *testing.T, database *maxminddb.Reader, address string, want bool) {
	t.Helper()
	var record countryRecord
	_, found, err := database.LookupNetwork(net.ParseIP(address), &record)
	if err != nil {
		if !want {
			return
		}
		t.Fatal(err)
	}
	if found != want {
		t.Fatalf("lookup %s found = %t, want %t", address, found, want)
	}
	if found && record.Country.ISOCode != "CN" {
		t.Fatalf("lookup %s country = %q, want CN", address, record.Country.ISOCode)
	}
}
