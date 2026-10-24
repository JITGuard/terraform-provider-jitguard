package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*awsConnectionDataSource)(nil)

type awsConnectionDataSource struct {
	client *Client
}

// NewAWSConnectionDataSource returns the read-only view of JIT Guard's AWS
// connection. It deliberately does not expose AWS accounts, OUs, permission
// sets or groups; those belong to the AWS provider.
func NewAWSConnectionDataSource() datasource.DataSource {
	return &awsConnectionDataSource{}
}

type awsConnectionModel struct {
	ConnectionStatus    types.String `tfsdk:"connection_status"`
	ManagementAccountID types.String `tfsdk:"management_account_id"`
	SSORegion           types.String `tfsdk:"sso_region"`
	SSOInstanceARN      types.String `tfsdk:"sso_instance_arn"`
}

func (d *awsConnectionDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "jitguard_aws_connection"
}

func (d *awsConnectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The JIT Guard customer's AWS organization connection: status, management account, " +
			"SSO region and SSO instance ARN. Use the AWS provider for account, OU, permission-set and group lookups.",
		Attributes: map[string]schema.Attribute{
			"connection_status": schema.StringAttribute{
				Computed:    true,
				Description: "connected or not_connected.",
			},
			"management_account_id": schema.StringAttribute{
				Computed:    true,
				Description: "AWS Organizations management account ID discovered by JIT Guard.",
			},
			"sso_region": schema.StringAttribute{
				Computed:    true,
				Description: "AWS IAM Identity Center region.",
			},
			"sso_instance_arn": schema.StringAttribute{
				Computed:    true,
				Description: "AWS IAM Identity Center instance ARN.",
			},
		},
	}
}

func (d *awsConnectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *awsConnectionDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	settings, err := d.client.GetSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading AWS connection", err.Error())
		return
	}
	if settings == nil {
		resp.Diagnostics.AddError("Customer not found", "The API key's customer no longer exists.")
		return
	}

	state := awsConnectionModel{
		ConnectionStatus:    types.StringValue(settings.ConnectionStatus),
		ManagementAccountID: types.StringValue(settings.ManagementAccountID),
		SSORegion:           types.StringValue(settings.SSORegion),
		SSOInstanceARN:      types.StringValue(settings.SSOInstanceARN),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
