package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestApprovalRuleModel_toRule_RoundTrip(t *testing.T) {
	accounts, err := accountConditionSetValue(context.Background(), []AccountCondition{
		{Type: "account_id", AccountID: "111111111111"},
		{Type: "account_tag", TagKey: "team", TagValue: "sre"},
	})
	if err != nil {
		t.Fatalf("accountConditionSetValue: %v", err)
	}
	permissionSets, err := permissionSetConditionSetValue(context.Background(), []PermissionSetCondition{
		{Type: "permission_set_arn", ARN: "arn:aws:sso:::permissionSet/ssoins-1111111111111111/ps-22222222222222222222222222222222"},
	})
	if err != nil {
		t.Fatalf("permissionSetConditionSetValue: %v", err)
	}
	ssoIds, err := stringSetValue(context.Background(), []string{"grp-sre"})
	if err != nil {
		t.Fatalf("stringSetValue: %v", err)
	}
	approverIds, err := stringSetValue(context.Background(), []string{})
	if err != nil {
		t.Fatalf("stringSetValue: %v", err)
	}

	m := approvalRuleModel{
		Name:             types.StringValue("SRE Emergency"),
		Description:      types.StringValue("test"),
		Enabled:          types.BoolValue(true),
		AutoApprove:      types.BoolValue(true),
		ApproverGroupIds: approverIds,
		OncallApproval:   types.BoolValue(true),
		SSOGroupIds:      ssoIds,
		Accounts:         accounts,
		PermissionSets:   permissionSets,
	}

	rule, err := m.toRule(context.Background())
	if err != nil {
		t.Fatalf("toRule: %v", err)
	}
	if len(rule.Accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(rule.Accounts))
	}
	if rule.Accounts[0].AccountID != "111111111111" || rule.Accounts[0].Type != "account_id" {
		t.Fatalf("unexpected account[0]: %+v", rule.Accounts[0])
	}
	if rule.Accounts[1].TagKey != "team" {
		t.Fatalf("unexpected account[1]: %+v", rule.Accounts[1])
	}
	if rule.PermissionSets[0].ARN != "arn:aws:sso:::permissionSet/ssoins-1111111111111111/ps-22222222222222222222222222222222" {
		t.Fatalf("unexpected permission set: %+v", rule.PermissionSets[0])
	}
}

func TestApprovalRuleModel_fromRule_RoundTrip(t *testing.T) {
	rule := Rule{
		RuleID:              "rule-1",
		Name:                "SRE Emergency",
		Description:         "test",
		Enabled:             true,
		Outcome:             "auto_approve",
		ApproverGroupIds:    []string{},
		AutoApproveIfOnCall: true,
		SSOGroupIds:         []string{"grp-sre"},
		Accounts: []AccountCondition{
			{Type: "account_id", AccountID: "111111111111"},
		},
		PermissionSets: []PermissionSetCondition{
			{Type: "permission_set_arn", ARN: "arn:aws:sso:::permissionSet/ssoins-1111111111111111/ps-22222222222222222222222222222222"},
		},
		CreatedAt: "2026-08-01T00:00:00Z",
		UpdatedAt: "2026-08-01T00:00:00Z",
	}

	var m approvalRuleModel
	if err := m.fromRule(context.Background(), &rule); err != nil {
		t.Fatalf("fromRule: %v", err)
	}
	if m.Accounts.IsNull() || m.Accounts.IsUnknown() {
		t.Fatal("accounts set should be populated after fromRule")
	}
	if len(m.Accounts.Elements()) != 1 {
		t.Fatalf("expected 1 account element, got %d", len(m.Accounts.Elements()))
	}

	// Now decode back out and verify the values survive.
	back, err := m.toRule(context.Background())
	if err != nil {
		t.Fatalf("toRule after fromRule: %v", err)
	}
	if len(back.Accounts) != 1 || back.Accounts[0].AccountID != "111111111111" {
		t.Fatalf("round-trip accounts mismatch: %+v", back.Accounts)
	}
	if len(back.PermissionSets) != 1 || back.PermissionSets[0].ARN != "arn:aws:sso:::permissionSet/ssoins-1111111111111111/ps-22222222222222222222222222222222" {
		t.Fatalf("round-trip permission sets mismatch: %+v", back.PermissionSets)
	}
}
