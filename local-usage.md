# Using the provider locally

Three steps: install the provider, create an API key, and write a Terraform
config.

## 1. Install the provider

The normal path is the Terraform Registry:

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
```

```sh
terraform init
```

For a pre-release version, or when you cannot reach the Registry, install a
GitHub Release into Terraform's implicit filesystem mirror:

```sh
./scripts/download-provider.sh 0.2.0
```

This fetches the release archive from
`github.com/JITGuard/terraform-provider-jitguard`, verifies its SHA256
checksum, and installs it to:

```text
~/.terraform.d/plugins/registry.terraform.io/JITGuard/jitguard/0.2.0/linux_amd64/
```

`terraform init` finds that location automatically — no `-plugin-dir` or
`.terraformrc` needed.

## 2. Create an API key

In the JIT Guard web app, open **Keys** and create a key. It looks like
`bg_sk_live_...`. It's shown once — copy it somewhere safe.

```sh
export JITGUARD_API_KEY=bg_sk_live_...
```

## 3. Write and apply your config

```hcl
# main.tf
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
  endpoint = "https://app.jitguard.com"  # use the non-prod URL to test safely
}

resource "jitguard_approval_rule" "example" {
  name          = "SRE Emergency"
  auto_approve  = true
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

Then:

```sh
terraform init
terraform plan
terraform apply
```

To test against a non-production environment instead of prod, point
`endpoint` at that environment's URL (e.g. `https://app.dev.jitguard.com`)
and use a key minted in that environment.

## Removing the provider

If you installed it from the filesystem mirror, delete the installed
directory:

```sh
rm -rf ~/.terraform.d/plugins/registry.terraform.io/JITGuard/jitguard
```

A Registry install is cached under `.terraform/` and the shared plugin cache;
`terraform init -upgrade` refreshes it.

## Troubleshooting

- **`Could not retrieve the list of available versions`** — check the source
  address is `JITGuard/jitguard` and the version constraint matches a
  published release.
- **`no available releases match`** — the version isn't published yet; check
  `github.com/JITGuard/terraform-provider-jitguard/releases` or relax the
  constraint.
- **`Missing API key`** — `JITGUARD_API_KEY` isn't set and no `api_key` is
  in the provider block.
- **`precondition failed` (412)** — someone edited the rule outside Terraform
  since the last read; re-plan and apply.
