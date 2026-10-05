# JIT Guard Provider

Manage JIT Guard approval rules as code. The provider talks to the
API-key-secured JIT Guard machine API; the API key is created and revoked
in the JIT Guard web app (Keys section).

Use the navigation to the left to read about the available resources and
data sources.

## Reference

- [jitguard_approval_rule](resources/approval_rule.md) — manage an approval
  rule (create / read / update / delete / import).
- [jitguard_approval_rules](data-sources/approval_rules.md) — list the
  customer's approval rules.

## Using the provider locally

See [local-usage.md](local-usage.md) for installing the provider, minting an
API key, and writing your first configuration.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.5
- A JIT Guard API key (`bg_sk_...`) minted in the web app **Keys** section.
  Only a hash of the key is stored at rest; a lost key must be revoked and
  re-created.

## Installation

The provider is published as GitHub Releases on
`github.com/JITGuard/terraform-provider-jitguard`. It is not on the public
Terraform Registry, so install it into Terraform's implicit filesystem mirror:

```sh
./scripts/download-provider.sh
```

This downloads the release archive, verifies it against the release's
`SHA256SUMS`, and installs the binary into the implicit filesystem-mirror
location
(`~/.terraform.d/plugins/registry.terraform.io/JITGuard/jitguard/<version>/<os>_<arch>/`),
so `terraform init` finds it automatically. Pass a version with
`./scripts/download-provider.sh 0.2.0`.

## Configuration

```hcl
terraform {
  required_version = ">= 1.5"
  required_providers {
    jitguard = {
      source  = "registry.terraform.io/JITGuard/jitguard"
      version = "0.1.0"
    }
  }
}

provider "jitguard" {
  endpoint = "https://app.jitguard.com"
  api_key  = var.jitguard_api_key   # or export JITGUARD_API_KEY
}
```

~> **Do not commit API keys.** Prefer `JITGUARD_API_KEY` or a secret store.
