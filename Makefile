VERSION ?= 0.2.0

.PHONY: build test vet docs release snapshot download

build:
	go build ./...

test:
	TF_ACC=1 go test ./...

vet:
	go vet ./...

# Regenerate reference docs from the schema (requires tfplugindocs on PATH).
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.25.0 generate
	rm -f docs/data-sources/approval_rules.md.bak docs/resources/approval_rule.md.bak

# Build and publish a GitHub Release from the current tag (requires goreleaser).
# Normal releases are produced by .github/workflows/release.yml, not this target.
release:
	goreleaser release --clean

# Build registry-shaped artifacts locally without publishing.
snapshot:
	goreleaser release --snapshot --clean

# Install a published provider locally (pre-Registry / offline fallback).
download:
	./scripts/download-provider.sh $(VERSION)
