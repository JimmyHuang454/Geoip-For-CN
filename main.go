package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"geoip-for-cn/internal/geodb"
	"geoip-for-cn/internal/srs"
	log "github.com/sirupsen/logrus"
)

var (
	srcFile      string
	dstFile      string
	databaseType string
	ipVersion    int
	outputFormat string
)

func init() {
	flag.StringVar(&srcFile, "s", "ipip_cn.txt", "specify source ip list file")
	flag.StringVar(&dstFile, "d", "Country.mmdb", "specify destination file")
	flag.StringVar(&outputFormat, "format", "auto", "output format: auto, mmdb, or srs")
	flag.StringVar(&databaseType, "t", "GeoIP2-Country", "specify MaxMind database type")
	flag.IntVar(&ipVersion, "ip-version", 6, "specify MMDB IP version (4 or 6)")
}

func main() {
	flag.Parse()
	if err := generate(srcFile, dstFile, outputFormat); err != nil {
		log.Fatal(err)
	}
}

func generate(source, destination, format string) error {
	if format == "auto" {
		format = strings.TrimPrefix(strings.ToLower(filepath.Ext(destination)), ".")
	}
	switch format {
	case "mmdb":
		return geodb.Generate(source, destination, geodb.Options{
			DatabaseType: databaseType,
			IPVersion:    ipVersion,
		})
	case "srs":
		return srs.Generate(source, destination)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}
