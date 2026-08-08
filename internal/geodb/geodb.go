package geodb

import (
	"bufio"
	"fmt"
	"net"
	"os"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

var chinaRecord = mmdbtype.Map{
	"country": mmdbtype.Map{
		"geoname_id":           mmdbtype.Uint32(1814991),
		"is_in_european_union": mmdbtype.Bool(false),
		"iso_code":             mmdbtype.String("CN"),
		"names": mmdbtype.Map{
			"de":    mmdbtype.String("China"),
			"en":    mmdbtype.String("China"),
			"es":    mmdbtype.String("China"),
			"fr":    mmdbtype.String("Chine"),
			"ja":    mmdbtype.String("中国"),
			"pt-BR": mmdbtype.String("China"),
			"ru":    mmdbtype.String("Китай"),
			"zh-CN": mmdbtype.String("中国"),
		},
	},
}

// Options controls the MaxMind database metadata and address family.
type Options struct {
	DatabaseType        string
	IPVersion           int
	DisableIPv4Aliasing bool
}

// Generate creates a MaxMind database from an IPv4, IPv6, or dual-stack CIDR list.
func Generate(source, destination string, options Options) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open CIDR list: %w", err)
	}
	defer input.Close()

	writer, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType:        options.DatabaseType,
		IPVersion:           options.IPVersion,
		DisableIPv4Aliasing: options.DisableIPv4Aliasing,
		RecordSize:          24,
	})
	if err != nil {
		return fmt.Errorf("create MMDB writer: %w", err)
	}

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		_, network, err := net.ParseCIDR(line)
		if err != nil || network == nil {
			return fmt.Errorf("parse CIDR %q: %w", line, err)
		}
		if err := writer.Insert(network, chinaRecord); err != nil {
			return fmt.Errorf("insert CIDR %q: %w", line, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read CIDR list: %w", err)
	}

	output, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create MMDB: %w", err)
	}
	if _, err := writer.WriteTo(output); err != nil {
		output.Close()
		return fmt.Errorf("write MMDB: %w", err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close MMDB: %w", err)
	}
	return nil
}
