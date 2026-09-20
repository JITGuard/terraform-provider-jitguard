VERSION ?= 0.1.0

.PHONY: build test vet docs release download

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

# Regenerate reference docs from the schema (requires tfplugindocs on PATH).
docs:
	tfplugindocs generate
	rm -f docs/data-sources/approval_rules.md.bak docs/resources/approval_rule.md.bak

# Build the release artifacts (per-platform zips + SHA256SUMS) into dist/.
release:
	./scripts/build-release.sh $(VERSION)

# Install a released provider version from GitHub Releases.
download:
	./scripts/download-provider.sh $(VERSION)
