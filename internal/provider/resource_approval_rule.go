package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure resource.Resource is implemented.
var _ resource.Resource = (*approvalRuleResource)(nil)

// Ensure ValidateConfig is implemented (cross-attribute plan validation).
var _ resource.ResourceWithValidateConfig = (*approvalRuleResource)(nil)

type approvalRuleResource struct {
	client *Client
}

// NewApprovalRuleResource returns the jitguard_approval_rule resource.
func NewApprovalRuleResource() resource.Resource {
	return &approvalRuleResource{}
}

// ValidateConfig rejects a requires-approval rule with no approver groups
// before any API call, mirroring the machine API's own validation.
func (r *approvalRuleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model approvalRuleModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if model.AutoApprove.ValueBool() {
		return
	}
	if model.ApproverGroupIds.IsNull() || len(model.ApproverGroupIds.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("approver_group_ids"),
			"approver_group_ids required",
			"approver_group_ids is required when auto_approve is false (requires_approval).",
		)
	}
}

type approvalRuleModel struct {
	RuleID           types.String `tfsdk:"rule_id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	AutoApprove      types.Bool   `tfsdk:"auto_approve"`
	ApproverGroupIds types.Set    `tfsdk:"approver_group_ids"`
	OncallApproval   types.Bool   `tfsdk:"oncall_approval"`
	SSOGroupIds      types.Set    `tfsdk:"sso_group_ids"`
	Accounts         types.Set    `tfsdk:"accounts"`
	PermissionSets   types.Set    `tfsdk:"permission_sets"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (r *approvalRuleResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "jitguard_approval_rule"
}

func (r *approvalRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An approval rule. Create, read, update, and delete via the " +
			"API-key-secured JIT Guard machine API. Updates are full replacements " +
			"guarded by an If-Match/ETag; a concurrent edit on the server fails the apply.",
		Attributes: map[string]schema.Attribute{
			"rule_id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique rule identifier (assigned by the API).",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-readable rule name.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Free-form description.",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the rule is active.",
			},
			"auto_approve": schema.BoolAttribute{
				Required:    true,
				Description: "Whether matching requests are granted immediately (true) or require approval (false).",
			},
			"approver_group_ids": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(basetypes.NewSetValueMust(types.StringType, []attr.Value{})),
				ElementType: types.StringType,
				Description: "SSO group IDs that can approve when auto_approve is false.",
			},
			"oncall_approval": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Auto-approve the requester when they are on call (only applies when auto_approve is false).",
			},
			"sso_group_ids": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "SSO groups that may request access under this rule.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp (ISO 8601).",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last-update timestamp (ISO 8601); doubles as the ETag for optimistic concurrency.",
			},
		},
		Blocks: map[string]schema.Block{
			"accounts": schema.SetNestedBlock{
				Description: "Accounts matched by this rule. One of account_id, account_tag, or ou_descendant per entry.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "Condition type: account_id, account_tag, or ou_descendant.",
							Validators: []validator.String{
								stringvalidator.OneOf("account_id", "account_tag", "ou_descendant"),
							},
						},
						"account_id": schema.StringAttribute{
							Optional:    true,
							Description: "AWS account ID — required when type is account_id.",
							Validators: []validator.String{
								stringvalidator.RegexMatches(regexp.MustCompile(`^(\d{12})?$`), "account_id must be a 12-digit AWS account ID"),
							},
						},
						"tag_key": schema.StringAttribute{
							Optional:    true,
							Description: "Account tag key — required when type is account_tag.",
						},
						"tag_value": schema.StringAttribute{
							Optional:    true,
							Description: "Account tag value — required when type is account_tag.",
						},
						"ou_id": schema.StringAttribute{
							Optional:    true,
							Description: "Organizational unit ID — required when type is ou_descendant.",
							Validators: []validator.String{
								stringvalidator.RegexMatches(regexp.MustCompile(`^(ou-[0-9a-z]{4,32}-[0-9a-z]{8,32})?$`), "ou_id must be an AWS OU ID (ou-...)"),
							},
						},
					},
				},
			},
			"permission_sets": schema.SetNestedBlock{
				Description: "SSO permission sets matched by this rule. One of permission_set_arn or permission_set_tag per entry.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "Condition type: permission_set_arn or permission_set_tag.",
							Validators: []validator.String{
								stringvalidator.OneOf("permission_set_arn", "permission_set_tag"),
							},
						},
						"arn": schema.StringAttribute{
							Optional:    true,
							Description: "SSO permission set ARN — required when type is permission_set_arn. Use \"*\" for any.",
							Validators: []validator.String{
								stringvalidator.RegexMatches(regexp.MustCompile(`^(\*|arn:aws:sso:::permissionSet/ssoins-[0-9a-f]{8,32}/ps-[0-9a-f]{8,32})?$`), "arn must be an SSO permission set ARN or \"*\""),
							},
						},
						"tag_key": schema.StringAttribute{
							Optional:    true,
							Description: "Permission set tag key — required when type is permission_set_tag.",
						},
						"tag_value": schema.StringAttribute{
							Optional:    true,
							Description: "Permission set tag value — required when type is permission_set_tag.",
						},
					},
				},
			},
		},
	}
}

func (r *approvalRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected *Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = client
}

// toRule converts the Terraform model to the API Rule request body.
func (m *approvalRuleModel) toRule(ctx context.Context) (Rule, error) {
	outcome := "requires_approval"
	if m.AutoApprove.ValueBool() {
		outcome = "auto_approve"
	}
	rule := Rule{
		Name:                m.Name.ValueString(),
		Description:         m.Description.ValueString(),
		Enabled:             m.Enabled.ValueBool(),
		Outcome:             outcome,
		AutoApproveIfOnCall: m.OncallApproval.ValueBool(),
	}

	var err error
	if rule.ApproverGroupIds, err = stringSetValues(ctx, m.ApproverGroupIds); err != nil {
		return Rule{}, fmt.Errorf("approver_group_ids: %w", err)
	}
	if rule.SSOGroupIds, err = stringSetValues(ctx, m.SSOGroupIds); err != nil {
		return Rule{}, fmt.Errorf("sso_group_ids: %w", err)
	}
	if rule.Accounts, err = accountConditionSetValues(ctx, m.Accounts); err != nil {
		return Rule{}, fmt.Errorf("accounts: %w", err)
	}
	if rule.PermissionSets, err = permissionSetConditionSetValues(ctx, m.PermissionSets); err != nil {
		return Rule{}, fmt.Errorf("permission_sets: %w", err)
	}
	return rule, nil
}

// fromRule fills the Terraform model from an API rule.
func (m *approvalRuleModel) fromRule(ctx context.Context, rule *Rule) error {
	m.RuleID = types.StringValue(rule.RuleID)
	m.Name = types.StringValue(rule.Name)
	m.Description = types.StringValue(rule.Description)
	m.Enabled = types.BoolValue(rule.Enabled)
	m.AutoApprove = types.BoolValue(rule.Outcome == "auto_approve")
	m.OncallApproval = types.BoolValue(rule.AutoApproveIfOnCall)
	m.CreatedAt = types.StringValue(rule.CreatedAt)
	m.UpdatedAt = types.StringValue(rule.UpdatedAt)

	approverIds, err := stringSetValue(ctx, rule.ApproverGroupIds)
	if err != nil {
		return err
	}
	ssoIds, err := stringSetValue(ctx, rule.SSOGroupIds)
	if err != nil {
		return err
	}
	accounts, err := accountConditionSetValue(ctx, rule.Accounts)
	if err != nil {
		return err
	}
	permissionSets, err := permissionSetConditionSetValue(ctx, rule.PermissionSets)
	if err != nil {
		return err
	}
	m.ApproverGroupIds = approverIds
	m.SSOGroupIds = ssoIds
	m.Accounts = accounts
	m.PermissionSets = permissionSets
	return nil
}

// ifMatch returns the quoted ETag for optimistic concurrency, derived from
// updatedAt (the API's etagFor fallback is createdAt).
func (m *approvalRuleModel) ifMatch() string {
	version := m.UpdatedAt.ValueString()
	if version == "" {
		version = m.CreatedAt.ValueString()
	}
	return `"` + version + `"`
}

func (r *approvalRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan approvalRuleModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := plan.toRule(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Invalid rule", err.Error())
		return
	}

	created, err := r.client.CreateRule(ctx, &rule)
	if err != nil {
		resp.Diagnostics.AddError("Error creating rule", err.Error())
		return
	}

	if err := plan.fromRule(ctx, created); err != nil {
		resp.Diagnostics.AddError("Error storing rule state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *approvalRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state approvalRuleModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetRule(ctx, state.RuleID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading rule", err.Error())
		return
	}
	if rule == nil {
		// Removed on the server — drop from state.
		resp.State.RemoveResource(ctx)
		return
	}
	if err := state.fromRule(ctx, rule); err != nil {
		resp.Diagnostics.AddError("Error storing rule state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *approvalRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state approvalRuleModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := plan.toRule(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Invalid rule", err.Error())
		return
	}

	updated, err := r.client.UpdateRule(ctx, state.RuleID.ValueString(), state.ifMatch(), &rule)
	if err != nil {
		resp.Diagnostics.AddError("Error updating rule", err.Error())
		return
	}
	if updated == nil {
		// Deleted concurrently — drop from state.
		resp.State.RemoveResource(ctx)
		return
	}
	if err := plan.fromRule(ctx, updated); err != nil {
		resp.Diagnostics.AddError("Error storing rule state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *approvalRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state approvalRuleModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.DeleteRule(ctx, state.RuleID.ValueString(), state.ifMatch()); err != nil {
		resp.Diagnostics.AddError("Error deleting rule", err.Error())
	}
}

func (r *approvalRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("rule_id"), req, resp)
}
