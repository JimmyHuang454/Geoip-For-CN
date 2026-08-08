package main

import (
	"flag"
	"fmt"
	"net"
	"net/netip"
	"os"

	"github.com/oschwald/geoip2-golang"
)

type ipCheck struct {
	name    string
	address string
	wantCN  bool
}

var defaultChecks = []ipCheck{
	// Well-known public DNS resolvers in mainland China.
	{"114DNS", "114.114.114.114", true},
	{"AliDNS", "223.5.5.5", true},
	{"DNSPod", "119.29.29.29", true},
	{"Baidu DNS", "180.76.76.76", true},
	{"AliDNS IPv6", "2400:3200::1", true},
	{"AliDNS IPv6 secondary", "2400:3200:baba::1", true},
	{"DNSPod IPv6", "2402:4e00::", true},
	{"Baidu DNS IPv6", "2400:da00::6666", true},
	{"360 Secure DNS IPv6", "240c::6666", true},
	{"360 Secure DNS IPv6 secondary", "240c::6644", true},

	// Well-known global DNS resolvers outside mainland China.
	{"Cloudflare IPv6", "2606:4700:4700::1111", false},
	{"Cloudflare IPv6 secondary", "2606:4700:4700::1001", false},
	{"Google IPv6", "2001:4860:4860::8888", false},
	{"Google IPv6 secondary", "2001:4860:4860::8844", false},
	{"Quad9 IPv6", "2620:fe::fe", false},
	{"Quad9 IPv6 secondary", "2620:fe::9", false},
	{"OpenDNS IPv6", "2620:119:35::35", false},
	{"OpenDNS IPv6 secondary", "2620:119:53::53", false},

	// https://www.cloudflare.com/ips/
	{"Cloudflare ipv4 Range", "103.21.244.1", false},
	{"Cloudflare ipv4 Range", "2400:cb00::", false},

	// Well-known public DNS resolvers NOT in mainland China.
	{"GoogleDNS", "8.8.8.8", false},

	{"NL", "185.49.33.8", false},
	{"US", "21.23.33.1", false},

	// Hong Kong IPv4 and IPv6 addresses must not be classified as mainland China.
	{"PCCW Hong Kong", "203.198.7.66", false},
	{"HKBN Hong Kong", "1.36.0.1", false},
	{"PCCW Business Hong Kong", "210.177.255.138", false},
	{"Hong Kong IPv6", "2001:2e0::1", false},

	// Taiwan IPv4 and IPv6 addresses must not be classified as mainland China.
	{"Taiwan IPv4", "1.34.0.1", false},
	{"HiNet Taiwan IPv4", "168.95.1.1", false},
	{"Taiwan IPv6", "2001:288::1", false},

	// Macao IPv4 and IPv6 addresses must not be classified as mainland China.
	{"Macao IPv4", "27.109.128.1", false},
	{"Macao IPv6", "2001:f90::1", false},

	// Private, loopback, link-local, and unique-local addresses must stay empty.
	{"RFC1918 10/8", "10.0.0.1", false},
	{"RFC1918 172.16/12", "172.16.0.1", false},
	{"RFC1918 192.168/16", "192.168.1.1", false},
	{"IPv4 loopback", "127.0.0.1", false},
	{"IPv4 link-local", "169.254.1.1", false},
	{"IPv6 unique-local", "fd00::1", false},
	{"IPv6 link-local", "fe80::1", false},
	{"IPv6 loopback", "::1", false},
}

func main() {
	databasePath := flag.String("database", "dist/Country.mmdb", "GeoIP2 database to verify")
	ipVersion := flag.Int("ip-version", 0, "checks to run: 0 for dual stack, 4 for IPv4, or 6 for IPv6")
	flag.Parse()

	if err := verifyDatabase(*databasePath, *ipVersion, defaultChecks); err != nil {
		fmt.Fprintln(os.Stderr, "verify-ip:", err)
		os.Exit(1)
	}
	fmt.Printf("verified %s successfully\n", *databasePath)
}

func verifyDatabase(databasePath string, ipVersion int, checks []ipCheck) error {
	if ipVersion != 0 && ipVersion != 4 && ipVersion != 6 {
		return fmt.Errorf("unsupported IP version %d", ipVersion)
	}
	database, err := geoip2.Open(databasePath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	checked := 0
	for _, check := range checks {
		address, err := netip.ParseAddr(check.address)
		if err != nil {
			return fmt.Errorf("%s has invalid IP %q: %w", check.name, check.address, err)
		}
		if (ipVersion == 4 && !address.Is4()) || (ipVersion == 6 && !address.Is6()) {
			continue
		}
		record, err := database.Country(net.IP(address.AsSlice()))
		if err != nil {
			return fmt.Errorf("look up %s (%s): %w", check.name, address, err)
		}
		gotCN := record.Country.IsoCode == "CN"
		if gotCN != check.wantCN {
			return fmt.Errorf("%s (%s) CN match=%t, want %t", check.name, address, gotCN, check.wantCN)
		}
		checked++
	}
	if checked == 0 {
		return fmt.Errorf("no IPv%d checks were run", ipVersion)
	}
	return nil
}
