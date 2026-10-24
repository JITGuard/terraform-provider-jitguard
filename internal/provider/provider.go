package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const envAPIKey = "JITGUARD_API_KEY"

// Ensure provider.Provider is implemented.
var _ provider.Provider = (*jitguardProvider)(nil)

type jitguardProvider struct {
	version string
}

// New returns the JIT Guard provider implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &jitguardProvider{version: version}
	}
}

type providerModel struct {
	Endpoint   types.String `tfsdk:"endpoint"`
	APIKey     types.String `tfsdk:"api_key"`
	SkipVerify types.Bool   `tfsdk:"skip_verify"`
}

func (p *jitguardProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "jitguard"
	resp.Version = p.version
}

func (p *jitguardProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage JIT Guard approval rules as code. The provider " +
			"talks to the API-key-secured machine API; the API key is created " +
			"and revoked in the JIT Guard web app (Keys section).",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Required:    true,
				Description: "JIT Guard API base URL, e.g. https://jitguard.example.com. Set to a non-production environment to test safely.",
			},
			"api_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				Description:         "API key (bg_sk_...) minted in the JIT Guard web app. May also be set via the JITGUARD_API_KEY environment variable.",
				MarkdownDescription: "API key (`bg_sk_...`) minted in the JIT Guard web app. May also be set via the `JITGUARD_API_KEY` environment variable.",
			},
			"skip_verify": schema.BoolAttribute{
				Optional:            true,
				Description:         "Skip TLS certificate verification. Intended only for local HTTPS testing against a self-signed endpoint.",
				MarkdownDescription: "Skip TLS certificate verification. Intended only for local HTTPS testing against a self-signed endpoint.",
			},
		},
	}
}

func (p *jitguardProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := config.Endpoint.ValueString()
	if endpoint == "" {
		resp.Diagnostics.AddError("Missing endpoint", "The jitguard provider requires an `endpoint` (the API base URL, e.g. https://jitguard.example.com).")
		return
	}

	apiKey := config.APIKey.ValueString()
	if apiKey == "" {
		apiKey = os.Getenv(envAPIKey)
	}
	if apiKey == "" {
		resp.Diagnostics.AddError("Missing API key", "Set `api_key` in the provider block or export the JITGUARD_API_KEY environment variable.")
		return
	}

	skipVerify := config.SkipVerify.ValueBool()

	client := NewClient(endpoint, apiKey, skipVerify)
	client.userAgent = "jitguard-terraform-provider/" + p.version
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *jitguardProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewApprovalRuleResource,
		NewOrganizationSettingsResource,
		NewPagerDutyIntegrationResource,
	}
}

func (p *jitguardProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewApprovalRulesDataSource,
		NewAWSConnectionDataSource,
		NewIntegrationsDataSource,
	}
}
