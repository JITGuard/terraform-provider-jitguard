# Terraform Provider JIT Guard

Manage [JIT Guard](https://jitguard.com) approval rules as code through the
API-key-secured machine API (ADR-0015).

The provider talks to the JIT Guard rules CRUDL endpoints:

- `jitguard_approval_rule` — create, read, update, delete a rule. Updates
  are full replacements guarded by `If-Match`/ETag optimistic concurrency, so
  a concurrent edit on the server fails the apply loudly (412).
- `jitguard_approval_rules` — list the customer's rules (useful for
  adopting existing rules into Terraform).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.5
- A JIT Guard API key (`bg_sk_...`) minted in the web app **Keys** section.
  Only a hash of the key is stored at rest; a lost key must be revoked and
  re-created.

## Installation

The provider is published as GitHub Releases on
`github.com/barneyparker/terraform-provider-jitguard`. It is not on the public
Terraform Registry, so install it into Terraform's implicit filesystem mirror.

### Install locally (recommended)

```sh
./scripts/download-provider.sh
```

This downloads the release archive from GitHub, verifies it against the
release's `SHA256SUMS`, and installs the binary into the implicit
filesystem-mirror location
(`~/.terraform.d/plugins/registry.terraform.io/barneyparker/jitguard/<version>/<os>_<arch>/`),
so `terraform init` finds it automatically — no extra configuration.

You can pass a version: `./scripts/download-provider.sh 0.2.0`.

### Manual install

Download `terraform-provider-jitguard_v<version>_<os>_<arch>.zip` and
`SHA256SUMS` from the release, verify the checksum, and unzip the binary into
the path above.

### Configuration

```hcl
terraform {
  required_version = ">= 1.5"
  required_providers {
    jitguard = {
      source  = "registry.terraform.io/barneyparker/jitguard"
      version = "0.1.0"
    }
  }
}

provider "jitguard" {
  endpoint = "https://app.jitguard.com"
  api_key  = var.jitguard_api_key   # or export JITGUARD_API_KEY
}
```

See `example/main.tf` for a complete configuration.

## Publishing a new version

Tag a release; CI builds the artifacts and publishes a GitHub Release:

```sh
git tag v0.2.0 && git push origin v0.2.0
```

`scripts/build-release.sh` builds
`terraform-provider-jitguard_v<version>_{os}_{arch}.zip` for
linux/darwin × amd64/arm64, plus a `SHA256SUMS` file. To build locally:

```sh
VERSION=0.2.0 make release
```

If the repository is later made public, these same assets are what the
Terraform Registry needs to publish the provider under `barneyparker/jitguard`.

## Reference documentation

The docs in `docs/` are generated from the provider schema with
[`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs) and are
what the Terraform Registry renders if/when the provider is published.

Regenerate after changing the schema:

```sh
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
export PATH="$PATH:$(go env GOPATH)/bin"
tfplugindocs generate
```

Templates live in `templates/` (index page, resource page, data-source page);
the schema tables are injected via `{{ .SchemaMarkdown }}`. The worked example
lives in `example/main.tf`.

## Usage

```hcl
resource "jitguard_approval_rule" "sre_emergency" {
  name           = "SRE Emergency"
  description    = "Emergency access for on-call SRE."
  auto_approve   = true
  enabled        = true
  sso_group_ids  = ["grp-sre"]

  accounts {
    type       = "account_id"
    account_id = "111111111111"
  }

  permission_sets {
    type = "permission_set_arn"
    arn  = "arn:aws:sso:::permissionSet/ssoins-1111111111111111/ps-22222222222222222222222222222222"
  }
}
```

### Provider arguments

| Argument      | Type   | Required | Description |
|---------------|--------|----------|-------------|
| `endpoint`    | string | yes      | JIT Guard API base URL, e.g. `https://jitguard.example.com`. Point at a non-production environment to test safely. |
| `api_key`     | string | yes      | API key `bg_sk_...` minted in the web app. Falls back to the `JITGUARD_API_KEY` environment variable. |
| `skip_verify` | bool   | no       | Skip TLS certificate verification. Intended only for local HTTPS testing against a self-signed endpoint. |

### `jitguard_approval_rule`

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `rule_id` | string | computed | Unique rule identifier assigned by the API. |
| `name` | string | yes | Human-readable rule name. |
| `description` | string | no | Free-form description (default `""`). |
| `enabled` | bool | no | Whether the rule is active (default `true`). |
| `auto_approve` | bool | yes | `true` grants matching requests immediately; `false` requires approval. |
| `approver_group_ids` | set(string) | no | SSO groups that can approve when `auto_approve = false`. |
| `oncall_approval` | bool | no | Auto-approve on-call requesters (default `true`; applies when `auto_approve = false`). |
| `sso_group_ids` | set(string) | yes | SSO groups that may request access under this rule. |
| `accounts` | block | yes | Account conditions (below). |
| `permission_sets` | block | yes | Permission-set conditions (below). |
| `created_at` | string | computed | Creation timestamp (ISO 8601). |
| `updated_at` | string | computed | Last-update timestamp; used as the ETag for optimistic concurrency. |

#### `accounts`

One of `account_id`, `account_tag`, or `ou_descendant` per block:

| Attribute | Required when | Description |
|-----------|---------------|-------------|
| `type` | always | `account_id`, `account_tag`, or `ou_descendant`. |
| `account_id` | `type = "account_id"` | AWS account ID. |
| `tag_key` / `tag_value` | `type = "account_tag"` | Account tag key/value. |
| `ou_id` | `type = "ou_descendant"` | Organizational unit ID. |

#### `permission_sets`

One of `permission_set_arn` or `permission_set_tag` per block:

| Attribute | Required when | Description |
|-----------|---------------|-------------|
| `type` | always | `permission_set_arn` or `permission_set_tag`. |
| `arn` | `type = "permission_set_arn"` | SSO permission set ARN; use `"*"` for any. |
| `tag_key` / `tag_value` | `type = "permission_set_tag"` | Permission-set tag key/value. |

### `jitguard_approval_rules`

A data source returning a `rules` list of the same objects as the resource.
Useful for discovering rules before adopting them.

## Local testing (acceptance tests)

The provider ships `go test` unit + framework acceptance tests that run
against an in-process fake of the machine API, so they need no credentials or
network:

```sh
go test ./...
```

For an end-to-end test against a real environment, mint a `bg_sk_...` key in
the JIT Guard web app and run `terraform plan`/`apply` with `endpoint`
pointing at that environment and `api_key` set.

## Registry publication

Not yet published. Before publishing to the Terraform Registry, the local +
acceptance tests must be reliable (this repo's CI gate). Build, sign, and
publish via `goreleaser` following the
[Hashicorp publishing workflow](https://developer.hashicorp.com/terraform/registry/providers/publishing).
