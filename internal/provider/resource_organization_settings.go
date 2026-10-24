package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*organizationSettingsResource)(nil)
var _ resource.ResourceWithImportState = (*organizationSettingsResource)(nil)

type organizationSettingsResource struct {
	client *Client
}

// NewOrganizationSettingsResource returns the singleton organization_settings
// resource. The API has one settings object per customer and one PUT, so this
// is one Terraform resource rather than three field-level resources.
func NewOrganizationSettingsResource() resource.Resource {
	return &organizationSettingsResource{}
}

type organizationSettingsModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	DefaultPermissionSetARN types.String `tfsdk:"default_permission_set_arn"`
	ApprovalExpiryHours     types.Int64  `tfsdk:"approval_expiry_hours"`
}

func (r *organizationSettingsResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "jitguard_organization_settings"
}

func (r *organizationSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The customer's JIT Guard organization settings: name, default permission set, " +
			"and approval expiry window. This is a singleton resource; one per customer. Deleting it " +
			"removes it from Terraform state but leaves the server settings unchanged.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Singleton identifier, always \"settings\".",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Organization display name. Omit to leave the server value unchanged.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"default_permission_set_arn": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Default SSO permission set ARN. Set to an empty string to clear it.",
			},
			"approval_expiry_hours": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Hours before an undecided approval request expires (1-720). The server default is 8.",
				Validators: []validator.Int64{
					int64validator.Between(1, 720),
				},
			},
		},
	}
}

func (r *organizationSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *Client, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (m organizationSettingsModel) updateFromConfig() SettingsUpdate {
	var update SettingsUpdate
	if !m.Name.IsNull() && !m.Name.IsUnknown() {
		value := m.Name.ValueString()
		update.Name = &value
	}
	if !m.DefaultPermissionSetARN.IsNull() && !m.DefaultPermissionSetARN.IsUnknown() {
		value := m.DefaultPermissionSetARN.ValueString()
		update.PermissionSetARN = &value
	}
	if !m.ApprovalExpiryHours.IsNull() && !m.ApprovalExpiryHours.IsUnknown() {
		value := m.ApprovalExpiryHours.ValueInt64()
		update.ApprovalExpiryHours = &value
	}
	return update
}

func (r *organizationSettingsResource) refresh(ctx context.Context, state *organizationSettingsModel, diags *diag.Diagnostics) {
	settings, err := r.client.GetSettings(ctx)
	if err != nil {
		diags.AddError("Error reading organization settings", err.Error())
		return
	}
	if settings == nil {
		diags.AddError("Customer not found", "The API key's customer no longer exists.")
		return
	}

	state.ID = types.StringValue("settings")
	state.Name = nullIfEmpty(settings.Name)
	state.DefaultPermissionSetARN = nullIfEmpty(settings.PermissionSetARN)
	state.ApprovalExpiryHours = types.Int64Value(settings.ApprovalExpiryHours)
}

func (r *organizationSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var config organizationSettingsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.UpdateSettings(ctx, config.updateFromConfig()); err != nil {
		resp.Diagnostics.AddError("Error creating organization settings", err.Error())
		return
	}

	var state organizationSettingsModel
	r.refresh(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *organizationSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := r.client.GetSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization settings", err.Error())
		return
	}
	if settings == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue("settings")
	state.Name = nullIfEmpty(settings.Name)
	state.DefaultPermissionSetARN = nullIfEmpty(settings.PermissionSetARN)
	state.ApprovalExpiryHours = types.Int64Value(settings.ApprovalExpiryHours)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *organizationSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var config organizationSettingsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.UpdateSettings(ctx, config.updateFromConfig()); err != nil {
		resp.Diagnostics.AddError("Error updating organization settings", err.Error())
		return
	}

	var state organizationSettingsModel
	r.refresh(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete is intentionally a no-op. The customer settings object cannot be
// deleted; Terraform drops it from state and leaves the server unchanged.
func (r *organizationSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *organizationSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
