// Command periodical-update downloads the upstream China IP lists, merges and
// minimizes their CIDRs, and generates the release MMDB artifacts.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"time"

	"geoip-for-cn/internal/geodb"
	"github.com/oschwald/maxminddb-golang"
)

const databaseType = "GeoIP2-Country"

var sources = []struct {
	name string
	url  string
}{
	{"ipip_net_v4.txt", "https://raw.githubusercontent.com/gaoyifan/china-operator-ip/refs/heads/ip-lists/china.txt"},
	{"ipip_net_v6.txt", "https://raw.githubusercontent.com/gaoyifan/china-operator-ip/refs/heads/ip-lists/china6.txt"},

	{"chunzhen_v4.txt", "https://raw.githubusercontent.com/metowolf/iplist/master/data/country/CN.txt"},

	{"ipverse_v4.txt", "https://raw.githubusercontent.com/ipverse/country-ip-blocks/master/country/cn/ipv4-aggregated.txt"},
	{"ipverse_v6.txt", "https://raw.githubusercontent.com/ipverse/country-ip-blocks/master/country/cn/ipv6-aggregated.txt"},

	{"APNIC_v4.txt", "https://raw.githubusercontent.com/mayaxcn/china-ip-list/master/chnroute.txt"},
	{"APNIC_v6.txt", "https://raw.githubusercontent.com/mayaxcn/china-ip-list/master/chnroute_v6.txt"},
}

type trieNode struct {
	covered  bool
	children [2]*trieNode
}

func main() {
	dist := flag.String("dist", "dist", "artifact output directory")
	purge := flag.Bool("purge", false, "request the CDN purge URL instead of generating artifacts")
	purgeURL := flag.String("purge-url", "", "CDN purge URL")
	verify := flag.Bool("verify", false, "verify generated release artifacts")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var err error
	if *verify {
		err = verifyArtifacts(*dist)
	} else if *purge {
		if *purgeURL == "" {
			err = errors.New("purge URL is empty")
		} else {
			err = requestURL(ctx, *purgeURL)
		}
	} else {
		err = generate(ctx, *dist)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "periodical-update:", err)
		os.Exit(1)
	}
}

type countryRecord struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

func verifyArtifacts(dist string) error {
	ipv4CIDRs := filepath.Join(dist, "CN-ip-cidr.txt")
	ipv6CIDRs := filepath.Join(dist, "CN-ipv6-cidr.txt")
	checks := []struct {
		database  string
		ipVersion uint
		included  []string
		excluded  []string
	}{
		{filepath.Join(dist, "Country-IPv4.mmdb"), 4, []string{ipv4CIDRs}, []string{ipv6CIDRs}},
		{filepath.Join(dist, "Country.mmdb"), 6, []string{ipv4CIDRs, ipv6CIDRs}, nil},
		{filepath.Join(dist, "Country-IPv6.mmdb"), 6, []string{ipv6CIDRs}, []string{ipv4CIDRs}},
	}
	for _, check := range checks {
		if err := verifyDatabase(check.database, check.ipVersion, check.included, check.excluded); err != nil {
			return err
		}
	}
	return nil
}

func verifyDatabase(databasePath string, ipVersion uint, included, excluded []string) error {
	database, err := maxminddb.Open(databasePath)
	if err != nil {
		return fmt.Errorf("open %s: %w", databasePath, err)
	}
	defer database.Close()
	if database.Metadata.IPVersion != ipVersion {
		return fmt.Errorf("verify %s: IP version is %d, want %d", databasePath, database.Metadata.IPVersion, ipVersion)
	}
	for _, cidrPath := range included {
		if err := verifyCIDRFile(database, cidrPath, true, true); err != nil {
			return fmt.Errorf("verify %s: %w", databasePath, err)
		}
	}
	for _, cidrPath := range excluded {
		if err := verifyCIDRFile(database, cidrPath, false, false); err != nil {
			return fmt.Errorf("verify %s: %w", databasePath, err)
		}
	}
	return nil
}

func verifyCIDRFile(database *maxminddb.Reader, path string, wantFound, checkAll bool) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	checked := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(line)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		var record countryRecord
		_, found, lookupErr := database.LookupNetwork(net.IP(prefix.Addr().AsSlice()), &record)
		if lookupErr != nil && wantFound {
			return fmt.Errorf("lookup %s: %w", prefix, lookupErr)
		}
		if found != wantFound {
			return fmt.Errorf("lookup %s found=%t, want %t", prefix, found, wantFound)
		}
		if found && record.Country.ISOCode != "CN" {
			return fmt.Errorf("lookup %s country=%q, want CN", prefix, record.Country.ISOCode)
		}
		checked++
		if !checkAll {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if checked == 0 {
		return fmt.Errorf("%s contains no CIDRs", path)
	}
	return nil
}

func generate(ctx context.Context, dist string) error {
	if err := os.MkdirAll(dist, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	dataDir := filepath.Join(dist, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create downloaded data directory: %w", err)
	}

	client := &http.Client{Timeout: 45 * time.Second}
	paths := make([]string, 0, len(sources))
	for _, source := range sources {
		legacyPath := filepath.Join(dist, source.name)
		if err := os.Remove(legacyPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove legacy downloaded data %s: %w", legacyPath, err)
		}
		path := filepath.Join(dataDir, source.name)
		if err := download(ctx, client, source.url, path); err != nil {
			return err
		}
		paths = append(paths, path)
	}

	ipv4CIDRs := filepath.Join(dist, "CN-ip-cidr.txt")
	ipv6CIDRs := filepath.Join(dist, "CN-ipv6-cidr.txt")
	dualStackCIDRs := filepath.Join(dist, "CN-ipv4-and-ipv6-cidr.txt")
	if err := mergeCIDRs(paths, ipv4CIDRs, 4); err != nil {
		return err
	}
	if err := mergeCIDRs(paths, ipv6CIDRs, 6); err != nil {
		return err
	}
	if err := mergeCIDRs(paths, dualStackCIDRs, 0); err != nil {
		return err
	}

	artifacts := []struct {
		cidrs               string
		database            string
		ipVersion           int
		disableIPv4Aliasing bool
	}{
		{ipv4CIDRs, filepath.Join(dist, "Country-IPv4.mmdb"), 4, false},
		{dualStackCIDRs, filepath.Join(dist, "Country.mmdb"), 6, false},
		{ipv6CIDRs, filepath.Join(dist, "Country-IPv6.mmdb"), 6, true},
	}
	for _, artifact := range artifacts {
		if err := geodb.Generate(artifact.cidrs, artifact.database, geodb.Options{
			DatabaseType:        databaseType,
			IPVersion:           artifact.ipVersion,
			DisableIPv4Aliasing: artifact.disableIPv4Aliasing,
		}); err != nil {
			return fmt.Errorf("generate %s: %w", filepath.Base(artifact.database), err)
		}
	}
	return nil
}

func download(ctx context.Context, client *http.Client, url, destination string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create download request for %s: %w", url, err)
	}
	request.Header.Set("User-Agent", "GeoIP2-CN-periodical-update")

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("download %s: unexpected HTTP status %s", url, response.Status)
	}

	temporary := destination + ".tmp"
	output, err := os.Create(temporary)
	if err != nil {
		return fmt.Errorf("create %s: %w", temporary, err)
	}
	_, copyErr := io.Copy(output, response.Body)
	closeErr := output.Close()
	if copyErr != nil {
		os.Remove(temporary)
		return fmt.Errorf("save %s: %w", destination, copyErr)
	}
	if closeErr != nil {
		os.Remove(temporary)
		return fmt.Errorf("close %s: %w", temporary, closeErr)
	}
	if err := os.Rename(temporary, destination); err != nil {
		os.Remove(temporary)
		return fmt.Errorf("replace %s: %w", destination, err)
	}
	return nil
}

func requestURL(ctx context.Context, url string) error {
	client := &http.Client{Timeout: 45 * time.Second}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create purge request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request CDN purge: %w", err)
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		return fmt.Errorf("read CDN purge response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("request CDN purge: unexpected HTTP status %s", response.Status)
	}
	return nil
}

func mergeCIDRs(inputs []string, destination string, ipVersion int) error {
	if ipVersion != 0 && ipVersion != 4 && ipVersion != 6 {
		return fmt.Errorf("unsupported IP version %d", ipVersion)
	}
	roots := map[int]*trieNode{4: {}, 6: {}}
	for _, input := range inputs {
		if err := addFile(roots, input); err != nil {
			return err
		}
	}
	compress(roots[4])
	compress(roots[6])

	output, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create merged CIDR list: %w", err)
	}
	writer := bufio.NewWriter(output)
	var writeErr error
	if ipVersion == 0 || ipVersion == 4 {
		writeErr = writeCIDRs(writer, roots[4], [16]byte{}, 0, 4)
	}
	if writeErr == nil && (ipVersion == 0 || ipVersion == 6) {
		writeErr = writeCIDRs(writer, roots[6], [16]byte{}, 0, 6)
	}
	flushErr := writer.Flush()
	closeErr := output.Close()
	if writeErr != nil {
		return writeErr
	}
	if flushErr != nil {
		return fmt.Errorf("flush merged CIDR list: %w", flushErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close merged CIDR list: %w", closeErr)
	}
	return nil
}

func addFile(roots map[int]*trieNode, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		prefix, err := netip.ParsePrefix(fields[0])
		if err != nil {
			return fmt.Errorf("parse %s:%d: %w", path, lineNumber, err)
		}
		ipVersion := 6
		if prefix.Addr().Is4() {
			ipVersion = 4
		}
		insert(roots[ipVersion], prefix.Masked())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}

func insert(root *trieNode, prefix netip.Prefix) {
	node := root
	var address [16]byte
	if prefix.Addr().Is4() {
		ipv4 := prefix.Addr().As4()
		copy(address[:4], ipv4[:])
	} else {
		address = prefix.Addr().As16()
	}
	for depth := 0; depth < prefix.Bits(); depth++ {
		if node.covered {
			return
		}
		bit := (address[depth/8] >> uint(7-depth%8)) & 1
		if node.children[bit] == nil {
			node.children[bit] = &trieNode{}
		}
		node = node.children[bit]
	}
	node.covered = true
	node.children = [2]*trieNode{}
}

func compress(node *trieNode) bool {
	if node == nil {
		return false
	}
	if node.covered {
		return true
	}
	leftCovered := compress(node.children[0])
	rightCovered := compress(node.children[1])
	if leftCovered && rightCovered {
		node.covered = true
		node.children = [2]*trieNode{}
	}
	return node.covered
}

func writeCIDRs(writer io.Writer, node *trieNode, address [16]byte, depth, ipVersion int) error {
	if node == nil {
		return nil
	}
	if node.covered {
		var addr netip.Addr
		if ipVersion == 4 {
			addr = netip.AddrFrom4([4]byte(address[:4]))
		} else {
			addr = netip.AddrFrom16(address)
		}
		_, err := fmt.Fprintf(writer, "%s/%d\n", addr, depth)
		return err
	}
	if err := writeCIDRs(writer, node.children[0], address, depth+1, ipVersion); err != nil {
		return err
	}
	if node.children[1] != nil {
		address[depth/8] |= byte(1) << uint(7-depth%8)
	}
	return writeCIDRs(writer, node.children[1], address, depth+1, ipVersion)
}
