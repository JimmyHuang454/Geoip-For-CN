package main

import (
	"flag"

	"geoip-for-cn/internal/geodb"
	log "github.com/sirupsen/logrus"
)

var (
	srcFile      string
	dstFile      string
	databaseType string
	ipVersion    int
)

func init() {
	flag.StringVar(&srcFile, "s", "ipip_cn.txt", "specify source ip list file")
	flag.StringVar(&dstFile, "d", "Country.mmdb", "specify destination mmdb file")
	flag.StringVar(&databaseType, "t", "GeoIP2-Country", "specify MaxMind database type")
	flag.IntVar(&ipVersion, "ip-version", 6, "specify MMDB IP version (4 or 6)")
	flag.Parse()
}

func main() {
	if err := geodb.Generate(srcFile, dstFile, geodb.Options{
		DatabaseType: databaseType,
		IPVersion:    ipVersion,
	}); err != nil {
		log.Fatal(err)
	}
}
