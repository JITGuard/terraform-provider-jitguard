# Using the provider locally

Three steps: install the provider binary, create an API key, and write a
Terraform config.

See the [index](index.md) for the full provider reference.

## 1. Install the provider

```sh
./scripts/download-provider.sh
```

This fetches the provider from the GitHub Releases for
`barneyparker/terraform-provider-jitguard`, verifies its SHA256 checksum, and
installs it to:

```
~/.terraform.d/plugins/registry.terraform.io/barneyparker/jitguard/0.1.0/linux_amd64/
```

`terraform init` finds this location automatically — no `-plugin-dir` or
`.terraformrc` needed. Pass a version to install something specific:

```sh
./scripts/download-provider.sh 0.2.0
```

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
  required_providers {
    jitguard = {
      source  = "registry.terraform.io/barneyparker/jitguard"
      version = "0.1.0"
    }
  }
}

provider "jitguard" {
  endpoint = "https://app.jitguard.com"  # use the non-prod URL to test safely
}

resource "jitguard_approval_rule" "example" {
  name           = "SRE Emergency"
  auto_approve   = true
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

Delete the installed directory:

```sh
rm -rf ~/.terraform.d/plugins/registry.terraform.io/barneyparker/jitguard
```

## Troubleshooting

- **`Could not retrieve the list of available versions`** — the release tag or
  asset name doesn't match; check the tags on
  `github.com/barneyparker/terraform-provider-jitguard/releases`.
- **`no available releases match`** — you asked for a version that isn't
  published yet; run `./scripts/download-provider.sh` to confirm what's
  current, or bump your `required_providers` constraint.
- **`Missing API key`** — `JITGUARD_API_KEY` isn't set and no `api_key` is
  in the provider block.
