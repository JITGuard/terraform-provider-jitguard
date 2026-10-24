package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*pagerDutyIntegrationResource)(nil)
var _ resource.ResourceWithImportState = (*pagerDutyIntegrationResource)(nil)

type pagerDutyIntegrationResource struct {
	client *Client
}

// NewPagerDutyIntegrationResource returns the singleton PagerDuty integration
// resource. The API key is write-only: Terraform sends it on apply but never
// stores it in state or plan.
func NewPagerDutyIntegrationResource() resource.Resource {
	return &pagerDutyIntegrationResource{}
}

type pagerDutyIntegrationModel struct {
	ID            types.String `tfsdk:"id"`
	APIKey        types.String `tfsdk:"api_key"`
	APIKeyVersion types.Int64  `tfsdk:"api_key_version"`
	Connected     types.Bool   `tfsdk:"connected"`
	MaskedAPIKey  types.String `tfsdk:"masked_api_key"`
}

func (r *pagerDutyIntegrationResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "jitguard_integration_pagerduty"
}

func (r *pagerDutyIntegrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The customer's PagerDuty integration. PagerDuty is the only integration the machine API " +
			"can manage; Slack is OAuth-installed in the web app. Deleting this resource disconnects PagerDuty.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Singleton identifier, always \"pagerduty\".",
			},
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				Description: "PagerDuty REST API key. Write-only: it is sent on apply but never stored in state or plan. Bump api_key_version to rotate it.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"api_key_version": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "Rotation trigger. Bump this after changing api_key so Terraform re-sends the write-only value.",
			},
			"connected": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether a PagerDuty API key is currently stored.",
			},
			"masked_api_key": schema.StringAttribute{
				Computed:    true,
				Description: "The server-side masked key (e.g. ****1234). Never the full key.",
			},
		},
	}
}

func (r *pagerDutyIntegrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *pagerDutyIntegrationResource) refresh(ctx context.Context, state *pagerDutyIntegrationModel, diags *diag.Diagnostics) {
	integrations, err := r.client.GetIntegrations(ctx)
	if err != nil {
		diags.AddError("Error reading PagerDuty integration", err.Error())
		return
	}
	state.ID = types.StringValue("pagerduty")
	state.Connected = types.BoolValue(integrations.PagerDuty.Connected)
	state.MaskedAPIKey = types.StringValue(integrations.PagerDuty.MaskedAPIKey)
	state.APIKey = types.StringNull()
}

func apiKeyVersionOrZero(version types.Int64) types.Int64 {
	if version.IsNull() || version.IsUnknown() {
		return types.Int64Value(0)
	}
	return version
}

func (r *pagerDutyIntegrationResource) requireAPIKey(config pagerDutyIntegrationModel, diags *diag.Diagnostics) bool {
	if config.APIKey.IsNull() || config.APIKey.IsUnknown() || config.APIKey.ValueString() == "" {
		diags.AddAttributeError(
			path.Root("api_key"),
			"api_key is required",
			"Set api_key to the PagerDuty REST API key. It is write-only and is not stored in Terraform state.",
		)
		return false
	}
	return true
}

func (r *pagerDutyIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var config, plan pagerDutyIntegrationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireAPIKey(config, &resp.Diagnostics) {
		return
	}

	if _, err := r.client.SetPagerDutyIntegration(ctx, config.APIKey.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error connecting PagerDuty", err.Error())
		return
	}

	var state pagerDutyIntegrationModel
	r.refresh(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	state.APIKeyVersion = apiKeyVersionOrZero(plan.APIKeyVersion)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pagerDutyIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pagerDutyIntegrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// api_key_version is a client-side rotation trigger. An imported resource
	// has no stored value, so default it to 0. This also keeps the post-import
	// state consistent with the schema default instead of returning null after
	// an import/update.
	state.APIKeyVersion = apiKeyVersionOrZero(state.APIKeyVersion)

	integrations, err := r.client.GetIntegrations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading PagerDuty integration", err.Error())
		return
	}
	if !integrations.PagerDuty.Connected {
		// Disconnected outside Terraform: drop it so the next plan recreates it.
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue("pagerduty")
	state.Connected = types.BoolValue(true)
	state.MaskedAPIKey = types.StringValue(integrations.PagerDuty.MaskedAPIKey)
	state.APIKey = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pagerDutyIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var config, plan pagerDutyIntegrationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireAPIKey(config, &resp.Diagnostics) {
		return
	}

	if _, err := r.client.SetPagerDutyIntegration(ctx, config.APIKey.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error updating PagerDuty integration", err.Error())
		return
	}

	var state pagerDutyIntegrationModel
	r.refresh(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	state.APIKeyVersion = apiKeyVersionOrZero(plan.APIKeyVersion)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pagerDutyIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state pagerDutyIntegrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.DisconnectPagerDutyIntegration(ctx); err != nil {
		resp.Diagnostics.AddError("Error disconnecting PagerDuty", err.Error())
	}
}

func (r *pagerDutyIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
