terraform {
  required_version = ">= 1.11"
  required_providers {
    jitguard = {
      source  = "registry.terraform.io/jitguard/jitguard"
      version = "~> 0.2"
    }
  }
}

provider "jitguard" {
  endpoint = "https://app.jitguard.com"
  # api_key  = var.jitguard_api_key
}

# 1. Auto-approve by exact account ID + named permission set ARN.
resource "jitguard_approval_rule" "sre_emergency" {
  name          = "SRE Emergency (Example)"
  description   = "On-call SREs are auto-approved into the production account."
  auto_approve  = true
  enabled       = true
  sso_group_ids = ["grp-sre", "grp-oncall"]

  accounts {
    type       = "account_id"
    account_id = "111111111111"
  }

  permission_sets {
    type = "permission_set_arn"
    arn  = "arn:aws:sso:::permissionSet/ssoins-1111111111111111/ps-22222222222222222222222222222222"
  }
}

# 2. Requires approval, matched by account tag + permission set tag.
resource "jitguard_approval_rule" "finance_monthly" {
  name               = "Finance Monthly (Example)"
  description        = "Finance teams need manager approval for tagged prod access."
  auto_approve       = false
  oncall_approval    = true
  enabled            = true
  approver_group_ids = ["grp-finance-managers"]
  sso_group_ids      = ["grp-finance"]

  accounts {
    type      = "account_tag"
    tag_key   = "team"
    tag_value = "finance"
  }

  permission_sets {
    type      = "permission_set_tag"
    tag_key   = "environment"
    tag_value = "prod"
  }
}

# 3. Any user in a child OU, any permission set.
resource "jitguard_approval_rule" "sandbox_ou" {
  name          = "Sandbox OU Access (Example)"
  description   = "Anyone in the sandbox OU can request any permission set."
  auto_approve  = true
  enabled       = true
  sso_group_ids = ["*"]

  accounts {
    type  = "ou_descendant"
    ou_id = "ou-sandb-2c3d4e5f"
  }

  permission_sets {
    type = "permission_set_arn"
    arn  = "*"
  }
}

# 4. Disabled example: requires approval, no on-call bypass.
resource "jitguard_approval_rule" "audit_readonly" {
  name               = "Audit Read-only (Example, disabled)"
  description        = "Read-only access for auditors; requires approval."
  auto_approve       = false
  oncall_approval    = false
  enabled            = false
  approver_group_ids = ["grp-audit-leads"]
  sso_group_ids      = ["grp-auditors"]

  accounts {
    type       = "account_id"
    account_id = "222222222222"
  }

  permission_sets {
    type = "permission_set_arn"
    arn  = "arn:aws:sso:::permissionSet/ssoins-1111111111111111/ps-22222222222222222222222222222222"
  }
}

data "jitguard_approval_rules" "all" {}

output "rule_details" {
  value = data.jitguard_approval_rules.all
}
