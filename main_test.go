package main

import (
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"geoip-for-cn/internal/srs"
)

func TestGenerateInfersSRSFormat(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "input.txt")
	destination := filepath.Join(directory, "cn.srs")
	if err := os.WriteFile(source, []byte("1.1.1.0/24\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generate(source, destination, "auto"); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	ruleSet, err := srs.Read(file)
	if err != nil {
		t.Fatal(err)
	}
	if !ruleSet.Contains(netip.MustParseAddr("1.1.1.1")) {
		t.Fatal("generated SRS does not contain source CIDR")
	}
}

func TestGenerateRejectsUnknownFormat(t *testing.T) {
	if err := generate("input.txt", "output.bin", "auto"); err == nil {
		t.Fatal("generate succeeded, want unsupported format error")
	}
}
