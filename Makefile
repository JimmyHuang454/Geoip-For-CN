GO ?= go
DIST_DIR ?= dist
RELEASE_DIR ?= release-artifacts
CDN_URL ?=
RELEASE_FILES := \
	Country.mmdb \
	Country-IPv4.mmdb \
	Country-IPv6.mmdb \
	cn.srs \
	cn-ipv4.srs \
	cn-ipv6.srs \
	CN-ip-cidr.txt \
	CN-ipv6-cidr.txt \
	CN-ipv4-and-ipv6-cidr.txt

.PHONY: all deps update release package check test vet build purge

all: update

# Download module dependencies without modifying go.mod or go.sum.
deps:
	$(GO) mod download

# Download the latest IPv4/IPv6 lists and regenerate every release artifact.
update: deps
	$(GO) run ./cmd/periodical-update -dist "$(DIST_DIR)"

# CI entry point: update first, then validate both code and generated databases.
release: update
	$(MAKE) test
	$(MAKE) vet

# CI publishing entry point: validate, then stage only public artifacts.
package: release
	mkdir -p "$(RELEASE_DIR)"
	cp $(addprefix $(DIST_DIR)/,$(RELEASE_FILES)) "$(RELEASE_DIR)/"

check: test vet

test:
	$(GO) test ./...
	$(GO) run ./cmd/periodical-update -dist "$(DIST_DIR)" -verify
	$(GO) run ./verify -database "$(DIST_DIR)/Country-IPv4.mmdb" -ip-version 4
	$(GO) run ./verify -database "$(DIST_DIR)/Country.mmdb" -ip-version 0
	$(GO) run ./verify -database "$(DIST_DIR)/Country-IPv6.mmdb" -ip-version 6

vet:
	$(GO) vet ./...

build: deps
	mkdir -p "$(DIST_DIR)"
	$(GO) build -o "$(DIST_DIR)/geoip-for-cn" .
	$(GO) build -o "$(DIST_DIR)/verify_ip" ./verify
	$(GO) build -o "$(DIST_DIR)/periodical-update" ./cmd/periodical-update

purge:
	@test -n "$(CDN_URL)" || { echo "CDN_URL is required" >&2; exit 1; }
	$(GO) run ./cmd/periodical-update -purge -purge-url "$(CDN_URL)"
