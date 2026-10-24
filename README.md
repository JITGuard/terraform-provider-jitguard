# Terraform Provider JIT Guard

Manage [JIT Guard](https://jitguard.com) configuration as code through the
API-key-secured machine API (ADR-0015).

The provider currently manages:

- `jitguard_approval_rule` — create, read, update, delete and import an
  approval rule. Updates are full replacements guarded by `If-Match`/ETag, so
  a concurrent edit on the server fails the apply loudly (412).
- `jitguard_approval_rules` — list the customer's rules, useful for adopting
  existing rules into Terraform.
- `jitguard_organization_settings` — the singleton organization name and
  approval expiry window.
- `jitguard_integration_pagerduty` — the PagerDuty integration. The API key is
  write-only and never stored in state.
- `jitguard_aws_connection` — read JIT Guard's AWS organization connection.
- `jitguard_integrations` — read Slack and PagerDuty integration status.

AWS accounts, OUs, permission sets and groups remain the AWS provider's job;
JIT Guard only exposes its connection to that org.

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
      source  = "jitguard/jitguard"
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
`~/.terraform.d/plugins/registry.terraform.io/jitguard/jitguard/<version>/<os>_<arch>/`.
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

### `jitguard_organization_settings`

The singleton organization settings object. Import with the fixed ID
`settings`; deleting it leaves the server settings unchanged.

```hcl
resource "jitguard_organization_settings" "this" {
  name                  = "Acme Corp"
  approval_expiry_hours = 12
}
```

Omit an attribute to leave the server value unchanged. Permission sets are
chosen per promotion request and matched by `jitguard_approval_rule`; the
organization settings no longer carry a default permission set.

### `jitguard_integration_pagerduty`

The PagerDuty integration. `api_key` is **write-only**, so it is never stored in
state or plan. Bump `api_key_version` to rotate the key.

```hcl
variable "pagerduty_api_key" {
  type      = string
  sensitive = true
}

resource "jitguard_integration_pagerduty" "this" {
  api_key         = var.pagerduty_api_key
  api_key_version = 1
}
```

Slack is OAuth-installed in the web app and is read-only in Terraform.

### `jitguard_aws_connection`

Reads JIT Guard's connection status, management account, SSO region and SSO
instance ARN. Use the AWS provider for account/OU/permission-set/group lookups.

```hcl
data "jitguard_aws_connection" "this" {}

output "jitguard_management_account" {
  value = data.jitguard_aws_connection.this.management_account_id
}
```

### `jitguard_integrations`

Reads Slack and PagerDuty status. The PagerDuty key is only ever exposed as
`masked_api_key`.

```hcl
data "jitguard_integrations" "this" {}

output "slack_team" {
  value = data.jitguard_integrations.this.slack.team_name
}
```

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

Releases are automated and **gated**. The release job only runs when the
repository variable `RELEASES_ENABLED` is `true`; until then, merges to `main`
and `next` still run CI but the release job is skipped. This prevents a release
from being cut before the signing key and Registry are ready.

1. Merge a Conventional-Commit PR to `main`.
2. `semantic-release` computes the next version from the commits, pushes the
   tag, and generates the release notes.
3. `GoReleaser` builds the linux/darwin/windows amd64/arm64 archives, the Registry
   manifest, `SHA256SUMS`, and its GPG signature.
4. `semantic-release` creates a **draft** GitHub Release, uploads every signed
   asset, and then publishes it. The Registry webhook therefore fires only
   after the release is complete.

To cut the first release after enabling the gate, run the **Release** workflow
manually (Actions → Release → Run workflow) on `main`. A rerun does not need an
empty merge commit.

Commit → version mapping: `fix`/`perf` → patch, `feat` → minor,
`BREAKING CHANGE`/`!` → major, `build(deps)`/`chore(deps)` → patch. Other
types (`docs`, `ci`, `test`, `refactor`, `chore`) do not release.

### Prerelease rehearsal

After setting `RELEASES_ENABLED=true`, create a `next` branch from `main` and
run the **Release** workflow on `next` (or push a release-worthy commit to it).
That creates a prerelease (e.g. `v0.2.0-next.1`) without spending a stable
version. Use it to prove the draft-publish handoff and the Registry ingest
before the first `v0.2.x`.

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

### Registry namespace

The Registry namespace is the GitHub organization that owns the repository,
not the personal account used to sign in. Sign in with an account that
administers `JITGuard`, authorize the Terraform Registry GitHub App for the
`JITGuard` organization, and publish
`JITGuard/terraform-provider-jitguard`. The provider address is
`jitguard/jitguard` (Terraform normalizes namespaces and types to lowercase).

Add the signing key under the **JITGuard organization** in
Registry → User Settings → Signing Keys, not under a personal `barneyparker`
namespace. A key registered only on the personal namespace will not validate
this provider's releases.

- Make the repository public; the Registry only ingests public repositories
  named `terraform-provider-{name}`.
- Add topics such as `terraform`, `terraform-provider`, `jitguard`.
- Create the `release` GitHub Environment and restrict it to `main` and
  `next`.
- Register the JITGuard organization's signing public key in the Terraform
  Registry.
- Publish the provider from the Registry UI and verify the webhook.
- Set the repository variable `RELEASES_ENABLED=true` only after the SSM
  parameter is populated and the Registry public key is registered.
- Require the `Verify`, `Test`, and `Conventional PR title` checks, require
  squash merges, and protect `main`.

## License

MPL-2.0. See [LICENSE](LICENSE).
