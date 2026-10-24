package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*integrationsDataSource)(nil)

type integrationsDataSource struct {
	client *Client
}

// NewIntegrationsDataSource returns the read-only view of the customer's
// Slack and PagerDuty integration status. Slack is OAuth-installed and
// read-only here; the PagerDuty resource manages its own key.
func NewIntegrationsDataSource() datasource.DataSource {
	return &integrationsDataSource{}
}

var slackIntegrationAttrs = map[string]attr.Type{
	"connected": types.BoolType,
	"team_id":   types.StringType,
	"team_name": types.StringType,
}

var pagerDutyIntegrationAttrs = map[string]attr.Type{
	"connected":      types.BoolType,
	"masked_api_key": types.StringType,
}

type slackIntegrationModel struct {
	Connected types.Bool   `tfsdk:"connected"`
	TeamID    types.String `tfsdk:"team_id"`
	TeamName  types.String `tfsdk:"team_name"`
}

type pagerDutyIntegrationDataModel struct {
	Connected    types.Bool   `tfsdk:"connected"`
	MaskedAPIKey types.String `tfsdk:"masked_api_key"`
}

type integrationsDataSourceModel struct {
	Slack     types.Object `tfsdk:"slack"`
	PagerDuty types.Object `tfsdk:"pagerduty"`
}

func (d *integrationsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "jitguard_integrations"
}

func (d *integrationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Read-only status of the customer's Slack and PagerDuty integrations.",
		Attributes: map[string]schema.Attribute{
			"slack": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Slack integration status. Slack is installed through OAuth in the web app.",
				Attributes: map[string]schema.Attribute{
					"connected": schema.BoolAttribute{Computed: true, Description: "Whether Slack is installed."},
					"team_id":   schema.StringAttribute{Computed: true, Description: "Slack workspace/team ID."},
					"team_name": schema.StringAttribute{Computed: true, Description: "Slack workspace/team name."},
				},
			},
			"pagerduty": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "PagerDuty integration status. The full key is never returned.",
				Attributes: map[string]schema.Attribute{
					"connected":      schema.BoolAttribute{Computed: true, Description: "Whether a PagerDuty key is stored."},
					"masked_api_key": schema.StringAttribute{Computed: true, Description: "Masked PagerDuty key, e.g. ****1234."},
				},
			},
		},
	}
}

func (d *integrationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *Client, got: %T", req.ProviderData))
		return
	}
	d.client = client
}

func (d *integrationsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	integrations, err := d.client.GetIntegrations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading integrations", err.Error())
		return
	}

	slack, diags := types.ObjectValueFrom(ctx, slackIntegrationAttrs, slackIntegrationModel{
		Connected: types.BoolValue(integrations.Slack.Connected),
		TeamID:    types.StringValue(integrations.Slack.TeamID),
		TeamName:  types.StringValue(integrations.Slack.TeamName),
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	pagerDuty, diags := types.ObjectValueFrom(ctx, pagerDutyIntegrationAttrs, pagerDutyIntegrationDataModel{
		Connected:    types.BoolValue(integrations.PagerDuty.Connected),
		MaskedAPIKey: types.StringValue(integrations.PagerDuty.MaskedAPIKey),
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := integrationsDataSourceModel{Slack: slack, PagerDuty: pagerDuty}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
