VERSION ?= 0.1.0

.PHONY: build test vet docs publish download

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

# Publish the provider mirror + docs to S3 (via CloudFront at
# https://jitguard.com/terraform/).
publish:
	VERSION=$(VERSION) ./scripts/publish-mirror.sh

# Install the provider locally from the hosted mirror.
download:
	./scripts/download-provider.sh $(VERSION)
