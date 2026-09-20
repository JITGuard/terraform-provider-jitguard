package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// accountConditionAttrs is the object element type for the `accounts` set.
// The element type must match the schema's SetNestedBlock nested object.
var accountConditionAttrs = map[string]attr.Type{
	"type":       types.StringType,
	"account_id": types.StringType,
	"tag_key":    types.StringType,
	"tag_value":  types.StringType,
	"ou_id":      types.StringType,
}

// permissionSetConditionAttrs is the object element type for `permission_sets`.
var permissionSetConditionAttrs = map[string]attr.Type{
	"type":      types.StringType,
	"arn":       types.StringType,
	"tag_key":   types.StringType,
	"tag_value": types.StringType,
}

// accountConditionModel is the nested-object element type for the `accounts`
// set. tfsdk tags mirror the schema's SetNestedBlock attributes.
type accountConditionModel struct {
	Type      types.String `tfsdk:"type"`
	AccountID types.String `tfsdk:"account_id"`
	TagKey    types.String `tfsdk:"tag_key"`
	TagValue  types.String `tfsdk:"tag_value"`
	OUId      types.String `tfsdk:"ou_id"`
}

// permissionSetConditionModel is the nested-object element type for
// `permission_sets`.
type permissionSetConditionModel struct {
	Type     types.String `tfsdk:"type"`
	ARN      types.String `tfsdk:"arn"`
	TagKey   types.String `tfsdk:"tag_key"`
	TagValue types.String `tfsdk:"tag_value"`
}

// stringSetValues converts a types.Set of strings to a Go slice.
func stringSetValues(ctx context.Context, set types.Set) ([]string, error) {
	var values []string
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}
	diags := set.ElementsAs(ctx, &values, false)
	if diags.HasError() {
		return nil, fmt.Errorf("cannot decode string set: %s", diags.Errors())
	}
	return values, nil
}

// stringSetValue builds a types.Set from a Go slice of strings.
func stringSetValue(ctx context.Context, values []string) (types.Set, error) {
	if values == nil {
		values = []string{}
	}
	set, diags := types.SetValueFrom(ctx, types.StringType, values)
	if diags.HasError() {
		return types.SetNull(types.StringType), fmt.Errorf("cannot encode string set: %s", diags.Errors())
	}
	return set, nil
}

// accountConditionSetValues decodes a types.Set of account-condition objects
// into a Go slice.
func accountConditionSetValues(ctx context.Context, set types.Set) ([]AccountCondition, error) {
	var models []accountConditionModel
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}
	diags := set.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, fmt.Errorf("cannot decode accounts: %s", diags.Errors())
	}
	conditions := make([]AccountCondition, 0, len(models))
	for _, m := range models {
		conditions = append(conditions, AccountCondition{
			Type:      m.Type.ValueString(),
			AccountID: m.AccountID.ValueString(),
			TagKey:    m.TagKey.ValueString(),
			TagValue:  m.TagValue.ValueString(),
			OUId:      m.OUId.ValueString(),
		})
	}
	return conditions, nil
}

// nullIfEmpty returns a StringNull when v is empty, otherwise a StringValue.
// The API omits unset condition fields (decode → ""), but the Terraform plan
// has null for them; normalising keeps plan and state consistent.
func nullIfEmpty(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}

// accountConditionSetValue builds a types.Set of object values.
func accountConditionSetValue(ctx context.Context, conditions []AccountCondition) (types.Set, error) {
	if conditions == nil {
		conditions = []AccountCondition{}
	}
	models := make([]accountConditionModel, 0, len(conditions))
	for _, c := range conditions {
		models = append(models, accountConditionModel{
			Type:      types.StringValue(c.Type),
			AccountID: nullIfEmpty(c.AccountID),
			TagKey:    nullIfEmpty(c.TagKey),
			TagValue:  nullIfEmpty(c.TagValue),
			OUId:      nullIfEmpty(c.OUId),
		})
	}
	objType := types.ObjectType{AttrTypes: accountConditionAttrs}
	set, diags := types.SetValueFrom(ctx, objType, models)
	if diags.HasError() {
		return types.SetNull(objType), fmt.Errorf("cannot encode accounts: %s", diags.Errors())
	}
	return set, nil
}

// permissionSetConditionSetValues decodes a types.Set of permission-set
// condition objects into a Go slice.
func permissionSetConditionSetValues(ctx context.Context, set types.Set) ([]PermissionSetCondition, error) {
	var models []permissionSetConditionModel
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}
	diags := set.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, fmt.Errorf("cannot decode permission_sets: %s", diags.Errors())
	}
	conditions := make([]PermissionSetCondition, 0, len(models))
	for _, m := range models {
		conditions = append(conditions, PermissionSetCondition{
			Type:     m.Type.ValueString(),
			ARN:      m.ARN.ValueString(),
			TagKey:   m.TagKey.ValueString(),
			TagValue: m.TagValue.ValueString(),
		})
	}
	return conditions, nil
}

// permissionSetConditionSetValue builds a types.Set of object values.
func permissionSetConditionSetValue(ctx context.Context, conditions []PermissionSetCondition) (types.Set, error) {
	if conditions == nil {
		conditions = []PermissionSetCondition{}
	}
	models := make([]permissionSetConditionModel, 0, len(conditions))
	for _, c := range conditions {
		models = append(models, permissionSetConditionModel{
			Type:     types.StringValue(c.Type),
			ARN:      nullIfEmpty(c.ARN),
			TagKey:   nullIfEmpty(c.TagKey),
			TagValue: nullIfEmpty(c.TagValue),
		})
	}
	objType := types.ObjectType{AttrTypes: permissionSetConditionAttrs}
	set, diags := types.SetValueFrom(ctx, objType, models)
	if diags.HasError() {
		return types.SetNull(objType), fmt.Errorf("cannot encode permission_sets: %s", diags.Errors())
	}
	return set, nil
}

// ruleObjectType builds the object type for the data source's `rules` list
// element, mirroring the resource schema's attributes.
func ruleObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"rule_id":            types.StringType,
			"name":               types.StringType,
			"description":        types.StringType,
			"enabled":            types.BoolType,
			"auto_approve":       types.BoolType,
			"approver_group_ids": types.SetType{ElemType: types.StringType},
			"oncall_approval":    types.BoolType,
			"sso_group_ids":      types.SetType{ElemType: types.StringType},
			"accounts":           types.SetType{ElemType: types.ObjectType{AttrTypes: accountConditionAttrs}},
			"permission_sets":    types.SetType{ElemType: types.ObjectType{AttrTypes: permissionSetConditionAttrs}},
			"created_at":         types.StringType,
			"updated_at":         types.StringType,
		},
	}
}
