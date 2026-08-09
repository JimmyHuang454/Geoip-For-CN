package srs

import (
	"bytes"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateAndRead(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "cn.txt")
	destination := filepath.Join(directory, "cn.srs")
	content := "# China IPs\n1.1.1.0/25\n1.1.1.128/25 note\n2400:3200::/32\n"
	if err := os.WriteFile(source, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Generate(source, destination); err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	ruleSet, err := Read(file)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		address string
		want    bool
	}{
		{"1.1.1.0", true},
		{"1.1.1.255", true},
		{"1.1.2.0", false},
		{"2400:3200::1", true},
		{"2400:3201::1", false},
	}
	for _, test := range tests {
		if got := ruleSet.Contains(netip.MustParseAddr(test.address)); got != test.want {
			t.Errorf("Contains(%s) = %t, want %t", test.address, got, test.want)
		}
	}
}

func TestGenerateMergesOverlappingAndAdjacentCIDRs(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "input.txt")
	destination := filepath.Join(directory, "output.srs")
	if err := os.WriteFile(source, []byte("10.0.0.0/9\n10.128.0.0/9\n10.1.0.0/16\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Generate(source, destination); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	ruleSet, err := Read(file)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(ruleSet.ranges); got != 1 {
		t.Fatalf("range count = %d, want 1", got)
	}
	if got, want := ruleSet.ranges[0].from.String(), "10.0.0.0"; got != want {
		t.Fatalf("range start = %s, want %s", got, want)
	}
	if got, want := ruleSet.ranges[0].to.String(), "10.255.255.255"; got != want {
		t.Fatalf("range end = %s, want %s", got, want)
	}
}

func TestGenerateRejectsInvalidAndEmptyInput(t *testing.T) {
	for _, content := range []string{"not-a-cidr\n", "\n# no entries\n"} {
		directory := t.TempDir()
		source := filepath.Join(directory, "input.txt")
		if err := os.WriteFile(source, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		if err := Generate(source, filepath.Join(directory, "output.srs")); err == nil {
			t.Fatalf("Generate(%q) succeeded, want error", content)
		}
	}
}

func TestReadRejectsInvalidFile(t *testing.T) {
	if _, err := Read(bytes.NewReader([]byte("not-srs"))); err == nil {
		t.Fatal("Read succeeded, want error")
	}
}
