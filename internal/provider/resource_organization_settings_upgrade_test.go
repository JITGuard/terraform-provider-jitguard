package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The default permission set was removed in #1391. States written by earlier
// provider versions still carry default_permission_set_arn, so the resource must
// advertise schema version 1 and provide a v0 upgrader that decodes (and then
// drops) the retired attribute.
func TestOrganizationSettingsStateUpgradeFromV0(t *testing.T) {
	server := providerserver.NewProtocol6(New("test")())()

	schemaResp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}
	resourceSchema, ok := schemaResp.ResourceSchemas["jitguard_organization_settings"]
	if !ok {
		t.Fatal("missing jitguard_organization_settings schema")
	}
	if resourceSchema.Version != 1 {
		t.Fatalf("schema version = %d, want 1", resourceSchema.Version)
	}

	// warm_group_idle_days is additive on schema version 1, so an upgrade from
	// a v0 state must still decode and leave the new attribute null (#1607).
	objectType, ok := resourceSchema.ValueType().(tftypes.Object)
	if !ok {
		t.Fatalf("resource value type = %T, want tftypes.Object", resourceSchema.ValueType())
	}
	if _, ok := objectType.AttributeTypes["warm_group_idle_days"]; !ok {
		t.Fatal("schema is missing warm_group_idle_days")
	}

	// A state shape written by provider <= v0.3.x.
	upgradeResp, err := server.UpgradeResourceState(context.Background(), &tfprotov6.UpgradeResourceStateRequest{
		TypeName: "jitguard_organization_settings",
		Version:  0,
		RawState: &tfprotov6.RawState{
			JSON: []byte(`{"id":"settings","name":"Acme","approval_expiry_hours":12,"default_permission_set_arn":"arn:aws:sso:::permissionSet/ps-1"}`),
		},
	})
	if err != nil {
		t.Fatalf("UpgradeResourceState: %v", err)
	}
	for _, diag := range upgradeResp.Diagnostics {
		if diag.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("upgrade diagnostic: %s: %s", diag.Summary, diag.Detail)
		}
	}
	if upgradeResp.UpgradedState == nil {
		t.Fatal("nil upgraded state")
	}

	value, err := upgradeResp.UpgradedState.Unmarshal(resourceSchema.ValueType())
	if err != nil {
		t.Fatalf("decode upgraded state: %v", err)
	}
	var attrs map[string]tftypes.Value
	if err := value.As(&attrs); err != nil {
		t.Fatalf("read upgraded state attributes: %v", err)
	}
	if _, ok := attrs["default_permission_set_arn"]; ok {
		t.Fatal("upgraded state still carries default_permission_set_arn")
	}

	idleDays, ok := attrs["warm_group_idle_days"]
	if !ok {
		t.Fatal("upgraded state is missing warm_group_idle_days")
	}
	if !idleDays.IsNull() {
		t.Fatalf("warm_group_idle_days = %v, want null after upgrading a v0 state", idleDays)
	}

	var name string
	if err := attrs["name"].As(&name); err != nil {
		t.Fatalf("read name: %v", err)
	}
	if name != "Acme" {
		t.Fatalf("name = %q, want Acme", name)
	}
}
