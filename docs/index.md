# JIT Guard Provider

Manage [JIT Guard](https://jitguard.com) configuration as code through the
API-key-secured JIT Guard machine API. The API key is created and revoked in
the JIT Guard web app (**Keys** section).

Use the navigation to the left to read about the available resources and
data sources.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.11
- A JIT Guard API key (`bg_sk_...`) minted in the web app **Keys** section.
  Only a hash of the key is stored at rest; a lost key must be revoked and
  re-created.

## Installation

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

The provider calls the versioned public API at `/api/v1/customer/api/...`.
The unversioned `/api/customer/api/...` alias remains available until
2027-10-25 but is deprecated; new clients and automation should use `/v1`.
`endpoint` remains the origin (for example `https://app.jitguard.com` or a
local test server), so custom endpoints continue to work unchanged.

For a pre-release version or an offline install, use
`scripts/download-provider.sh` from the provider repository. See the
[local usage guide](https://github.com/JITGuard/terraform-provider-jitguard/blob/main/local-usage.md)
for the walkthrough.

## Reference

- [jitguard_approval_rule](resources/approval_rule.md) — manage an approval
  rule (create / read / update / delete / import).
- [jitguard_approval_rules](data-sources/approval_rules.md) — list the
  customer's approval rules.
- [jitguard_organization_settings](resources/organization_settings.md) —
  manage the organization name and approval expiry.
- [jitguard_integration_pagerduty](resources/integration_pagerduty.md) —
  manage the PagerDuty integration.
- [jitguard_aws_connection](data-sources/aws_connection.md) — read JIT Guard's
  AWS connection status.
- [jitguard_integrations](data-sources/integrations.md) — read Slack and
  PagerDuty integration status.

Look up AWS account IDs, OU IDs, permission-set ARNs and Identity Store group
IDs with the
[AWS provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs);
JIT Guard does not own those AWS resources.

~> **Do not commit API keys.** Prefer `JITGUARD_API_KEY` or a secret store.
