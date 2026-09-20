package provider

import (
	"context"
	"testing"
)

// Reproduce the plan-vs-state correlation failure: the planned accounts set
// element has nulls for unset fields; the provider's state must produce the
// exact same value.
func TestPlanStateCorrelation_Accounts(t *testing.T) {
	// Simulate what Terraform's plan holds for accounts (config only sets
	// type + account_id; the rest are null).
	planned, err := accountConditionSetValue(context.Background(), []AccountCondition{
		{Type: "account_id", AccountID: "111111111111"},
	})
	if err != nil {
		t.Fatalf("build planned: %v", err)
	}

	// Provider creates the rule; API returns account with only the set field
	// (empty optional fields omitted by the fake server).
	apiRule := AccountCondition{Type: "account_id", AccountID: "111111111111"}
	actual, err := accountConditionSetValue(context.Background(), []AccountCondition{apiRule})
	if err != nil {
		t.Fatalf("build actual: %v", err)
	}

	if !actual.Equal(planned) {
		t.Fatalf("planned and actual sets differ.\nplanned: %s\nactual:  %s",
			planned.String(), actual.String())
	}
}

func TestPlanStateCorrelation_PermissionSets(t *testing.T) {
	planned, err := permissionSetConditionSetValue(context.Background(), []PermissionSetCondition{
		{Type: "permission_set_arn", ARN: "arn:aws:sso:::permissionSet/ssoins-1111111111111111/ps-22222222222222222222222222222222"},
	})
	if err != nil {
		t.Fatalf("build planned: %v", err)
	}

	apiRule := PermissionSetCondition{Type: "permission_set_arn", ARN: "arn:aws:sso:::permissionSet/ssoins-1111111111111111/ps-22222222222222222222222222222222"}
	actual, err := permissionSetConditionSetValue(context.Background(), []PermissionSetCondition{apiRule})
	if err != nil {
		t.Fatalf("build actual: %v", err)
	}

	if !actual.Equal(planned) {
		t.Fatalf("planned and actual sets differ.\nplanned: %s\nactual:  %s",
			planned.String(), actual.String())
	}
}
