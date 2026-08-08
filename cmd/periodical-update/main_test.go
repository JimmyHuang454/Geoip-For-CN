package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeCIDRs(t *testing.T) {
	tests := []struct {
		name      string
		ipVersion int
		inputs    []string
		want      string
	}{
		{
			name:      "IPv4 adjacent networks",
			ipVersion: 4,
			inputs: []string{
				"10.0.0.0/9\n192.0.2.0/25\n",
				"10.128.0.0/9\n192.0.2.128/25",
			},
			want: "10.0.0.0/8\n192.0.2.0/24\n",
		},
		{
			name:      "IPv6 adjacent networks",
			ipVersion: 6,
			inputs: []string{
				"# IPv6 source\n2001:db8::/33\n2001:db9::/48\n",
				"2001:db8:8000::/33\n2001:db9::/49",
			},
			want: "2001:db8::/32\n2001:db9::/48\n",
		},
		{
			name:      "mixed IPv4 and IPv6 input",
			ipVersion: 0,
			inputs: []string{
				"192.0.2.0/25\n2001:db8::/33\n",
				"2001:db8:8000::/33\n192.0.2.128/25\n",
			},
			want: "192.0.2.0/24\n2001:db8::/32\n",
		},
		{
			name:      "IPv4 filter accepts mixed input",
			ipVersion: 4,
			inputs:    []string{"192.0.2.0/24\n2001:db8::/32\n"},
			want:      "192.0.2.0/24\n",
		},
		{
			name:      "IPv6 filter accepts mixed input",
			ipVersion: 6,
			inputs:    []string{"192.0.2.0/24\n2001:db8::/32\n"},
			want:      "2001:db8::/32\n",
		},
		{
			name:      "duplicates contained networks and host bits",
			ipVersion: 4,
			inputs: []string{
				"\n # comment\n10.1.2.3/8 extra-column\n10.0.0.0/9\n",
				"10.0.0.0/8\n10.20.0.0/16\n",
			},
			want: "10.0.0.0/8\n",
		},
		{
			name:      "IPv4 full address space",
			ipVersion: 4,
			inputs:    []string{"0.0.0.0/1\n128.0.0.0/1\n"},
			want:      "0.0.0.0/0\n",
		},
		{
			name:      "IPv4 full address space",
			ipVersion: 4,
			inputs:    []string{"0.0.0.0/1\n128.0.0.0/1\n127.0.0.1/24"},
			want:      "0.0.0.0/0\n",
		},
		{
			name:      "IPv6 full address space",
			ipVersion: 6,
			inputs:    []string{"::/1\n8000::/1\n"},
			want:      "::/0\n",
		},
		{
			name:      "empty input",
			ipVersion: 4,
			inputs:    []string{"\n# comments only\n"},
			want:      "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			inputs := make([]string, 0, len(test.inputs))
			for index, content := range test.inputs {
				path := filepath.Join(dir, "input-"+string(rune('a'+index))+".txt")
				writeTestFile(t, path, content)
				inputs = append(inputs, path)
			}
			output := filepath.Join(dir, "output.txt")
			if err := mergeCIDRs(inputs, output, test.ipVersion); err != nil {
				t.Fatal(err)
			}
			if got := readTestFile(t, output); got != test.want {
				t.Fatalf("merged list = %q, want %q", got, test.want)
			}
		})
	}
}

func TestMergeCIDRsRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		ipVersion int
		input     string
		wantError string
	}{
		{"invalid CIDR", 4, "not-a-cidr\n", "parse"},
		{"unsupported output family", 5, "192.0.2.0/24\n", "unsupported IP version"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "input.txt")
			writeTestFile(t, input, test.input)
			err := mergeCIDRs([]string{input}, filepath.Join(dir, "output.txt"), test.ipVersion)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want error containing %q", err, test.wantError)
			}
		})
	}
}

func TestMergeCIDRsRejectsMissingInput(t *testing.T) {
	dir := t.TempDir()
	err := mergeCIDRs(
		[]string{filepath.Join(dir, "missing.txt")},
		filepath.Join(dir, "output.txt"),
		4,
	)
	if err == nil || !strings.Contains(err.Error(), "open") {
		t.Fatalf("error = %v, want open error", err)
	}
}

func TestDownloadFollowsRedirectAndSetsUserAgent(t *testing.T) {
	const body = "192.0.2.0/24\n"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/redirect":
			http.Redirect(writer, request, "/data", http.StatusFound)
		case "/data":
			if got, want := request.Header.Get("User-Agent"), "GeoIP2-CN-periodical-update"; got != want {
				t.Errorf("User-Agent = %q, want %q", got, want)
			}
			io.WriteString(writer, body)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "download.txt")
	writeTestFile(t, destination, "old data")
	if err := download(context.Background(), server.Client(), server.URL+"/redirect", destination); err != nil {
		t.Fatal(err)
	}
	if got := readTestFile(t, destination); got != body {
		t.Fatalf("downloaded data = %q, want %q", got, body)
	}
	if _, err := os.Stat(destination + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temporary file was not removed: %v", err)
	}
}

func TestDownloadRejectsHTTPErrorWithoutReplacingDestination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "download.txt")
	writeTestFile(t, destination, "existing data")
	err := download(context.Background(), server.Client(), server.URL, destination)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("error = %v, want HTTP 503 error", err)
	}
	if got := readTestFile(t, destination); got != "existing data" {
		t.Fatalf("destination was replaced with %q", got)
	}
}

func TestDownloadRemovesTemporaryFileAfterCopyError(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "download.txt")
	writeTestFile(t, destination, "existing data")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(errorReader{}),
		}, nil
	})}

	err := download(context.Background(), client, "https://example.invalid/list.txt", destination)
	if err == nil || !strings.Contains(err.Error(), "save") {
		t.Fatalf("error = %v, want save error", err)
	}
	if got := readTestFile(t, destination); got != "existing data" {
		t.Fatalf("destination was replaced with %q", got)
	}
	if _, err := os.Stat(destination + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temporary file was not removed: %v", err)
	}
}

func TestRequestURL(t *testing.T) {
	requested := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requested = true
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := requestURL(context.Background(), server.URL); err != nil {
		t.Fatal(err)
	}
	if !requested {
		t.Fatal("server was not requested")
	}
}

func TestRequestURLRejectsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	err := requestURL(context.Background(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("error = %v, want HTTP 502 error", err)
	}
}

func TestGenerateAndVerifyArtifacts(t *testing.T) {
	data := map[string]string{
		"/ipv4-a": "1.1.1.0/25\n2400:3210::/32\n",
		"/ipv4-b": "1.1.1.128/25\n",
		"/ipv6":   "# mixed generated test data\n2400:3200::/33\n1.1.1.64/26\n2400:3200:8000::/33\n",
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		content, ok := data[request.URL.Path]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		io.WriteString(writer, content)
	}))
	defer server.Close()

	originalSources := sources
	sources = []struct {
		name string
		url  string
	}{
		{"ipip_net.txt", server.URL + "/ipv4-a"},
		{"chunzhen.txt", server.URL + "/ipv4-b"},
		{"ipverse_ipv6.txt", server.URL + "/ipv6"},
	}
	defer func() { sources = originalSources }()

	dist := t.TempDir()
	legacyPath := filepath.Join(dist, "ipip_net.txt")
	writeTestFile(t, legacyPath, "stale data")
	if err := generate(context.Background(), dist); err != nil {
		t.Fatal(err)
	}
	if err := verifyArtifacts(dist); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy download was not removed: %v", err)
	}
	if got, want := readTestFile(t, filepath.Join(dist, "data", "ipip_net.txt")), data["/ipv4-a"]; got != want {
		t.Fatalf("downloaded IPv4 data = %q, want %q", got, want)
	}
	if got, want := readTestFile(t, filepath.Join(dist, "CN-ip-cidr.txt")), "1.1.1.0/24\n"; got != want {
		t.Fatalf("IPv4 CIDRs = %q, want %q", got, want)
	}
	if got, want := readTestFile(t, filepath.Join(dist, "CN-ipv6-cidr.txt")), "2400:3200::/32\n2400:3210::/32\n"; got != want {
		t.Fatalf("IPv6 CIDRs = %q, want %q", got, want)
	}
	for _, name := range []string{"Country-IPv4.mmdb", "Country.mmdb", "Country-IPv6.mmdb"} {
		info, err := os.Stat(filepath.Join(dist, name))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("injected read failure")
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
