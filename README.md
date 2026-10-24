# Terraform Provider JIT Guard

Manage [JIT Guard](https://jitguard.com) configuration as code through the
API-key-secured machine API (ADR-0015).

The provider currently manages:

- `jitguard_approval_rule` — create, read, update, delete and import an
  approval rule. Updates are full replacements guarded by `If-Match`/ETag, so
  a concurrent edit on the server fails the apply loudly (412).
- `jitguard_approval_rules` — list the customer's rules, useful for adopting
  existing rules into Terraform.

Organization settings, AWS connection information and messaging/on-call
integrations are the next machine-API surfaces; they are tracked separately
from the initial approval-rule release.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.11
- A JIT Guard API key (`bg_sk_...`) minted in the web app **Keys** section.
  Only a hash of the key is stored at rest; a lost key must be revoked and
  re-created.

## Installation

The provider is published on the Terraform Registry:

```hcl
terraform {
  required_version = ">= 1.11"
  required_providers {
    jitguard = {
      source  = "JITGuard/jitguard"
      version = "~> 0.2"
    }
  }
}

provider "jitguard" {
  endpoint = "https://app.jitguard.com"
  api_key  = var.jitguard_api_key   # or export JITGUARD_API_KEY
}
```

Then run `terraform init`.

### Local/pre-release install

Before a version is on the Registry, or for offline testing, install the
GitHub Release into Terraform's implicit filesystem mirror:

```sh
./scripts/download-provider.sh 0.2.0
```

This verifies the archive against the release's `SHA256SUMS` and installs it
under
`~/.terraform.d/plugins/registry.terraform.io/JITGuard/jitguard/<version>/<os>_<arch>/`.
See `local-usage.md` for the walkthrough.

## Usage

```hcl
resource "jitguard_approval_rule" "sre_emergency" {
  name          = "SRE Emergency"
  description   = "Emergency access for on-call SRE."
  auto_approve  = true
  enabled       = true
  sso_group_ids = ["grp-sre"]

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

Look up AWS account IDs, OU IDs, permission-set ARNs and Identity Store group
IDs with the
[AWS provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)
rather than hardcoding them; JIT Guard does not own those AWS resources.

### Provider arguments

| Argument      | Type   | Required | Description |
|---------------|--------|----------|-------------|
| `endpoint`    | string | yes      | JIT Guard API base URL, e.g. `https://app.jitguard.com`. Point at a non-production environment to test safely. |
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

## Development

```sh
make build      # go build ./...
make test       # TF_ACC=1 go test ./... (in-process fake API; no credentials)
make vet        # go vet ./...
make docs       # regenerate docs/ from the provider schema
make snapshot   # build registry-shaped artifacts into dist/ with GoReleaser
```

The acceptance tests run the real Terraform plugin protocol against an
in-process fake of the machine API. `TF_ACC=1` is required; CI sets it.

For an end-to-end test against a real environment, mint a `bg_sk_...` key in
the JIT Guard web app and run `terraform plan`/`apply` with `endpoint`
pointing at that environment and `api_key` set.

## Reference documentation

The docs in `docs/` are generated from the provider schema with
[`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs) and are
what the Terraform Registry renders. Regenerate after changing the schema:

```sh
make docs
```

Templates live in `templates/` (index page, resource page, data-source page);
the schema tables are injected via `{{ .SchemaMarkdown }}`. The worked example
lives in `example/main.tf`.

## Releasing

Releases are automated. Do **not** create or move a `v*` tag by hand.

1. Merge a Conventional-Commit PR to `main`.
2. `semantic-release` computes the next version from the commits, pushes the
   tag, and generates the release notes.
3. `GoReleaser` builds the linux/darwin amd64/arm64 archives, the Registry
   manifest, `SHA256SUMS`, and its GPG signature.
4. `semantic-release` creates a **draft** GitHub Release, uploads every signed
   asset, and then publishes it. The Registry webhook therefore fires only
   after the release is complete.

Commit → version mapping: `fix`/`perf` → patch, `feat` → minor,
`BREAKING CHANGE`/`!` → major, `build(deps)`/`chore(deps)` → patch. Other
types (`docs`, `ci`, `test`, `refactor`, `chore`) do not release.

### Prerelease rehearsal

Push to the `next` branch to create a prerelease (e.g. `v0.2.0-next.1`)
without spending a stable version. Use it to prove the draft-publish handoff
and the Registry ingest before the first `v0.2.x`.

### Signing key

The release job reads the GPG signing material from a prod-only SSM
`SecureString`:

```text
/jitguard/prod/terraform-provider-jitguard/gpg-signing-key
```

The value is a JSON document:

```json
{ "privateKey": "-----BEGIN PGP PRIVATE KEY BLOCK-----...", "passphrase": "...", "fingerprint": "..." }
```

It is created by Terraform in `jitguard-core` as `placeholder` with
`ignore_changes = [value]` and `prevent_destroy`, then populated out-of-band.
The public key must be registered in the Terraform Registry before the first
release. If a draft is created but not published, run the **Publish draft
release** workflow with the tag.

## Repository setup (one-time)

- Make the repository public; the Registry only ingests public repositories
  named `terraform-provider-{name}`.
- Add topics such as `terraform`, `terraform-provider`, `jitguard`.
- Create the `release` GitHub Environment and restrict it to `main` and
  `next`.
- Register the signing public key in the Terraform Registry.
- Publish the provider from the Registry UI and verify the webhook.
- Require the `Verify`, `Test`, and `Conventional PR title` checks, require
  squash merges, and protect `main`.

## License

MPL-2.0. See [LICENSE](LICENSE).
