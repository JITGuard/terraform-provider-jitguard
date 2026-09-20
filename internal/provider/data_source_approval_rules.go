package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// Ensure datasource.DataSource is implemented.
var _ datasource.DataSource = (*approvalRulesDataSource)(nil)

type approvalRulesDataSource struct {
	client *Client
}

// NewApprovalRulesDataSource returns the jitguard_approval_rules data source.
func NewApprovalRulesDataSource() datasource.DataSource {
	return &approvalRulesDataSource{}
}

type approvalRulesModel struct {
	Rules []approvalRuleModel `tfsdk:"rules"`
}

func (d *approvalRulesDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "jitguard_approval_rules"
}

func (d *approvalRulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists every approval rule for the customer. Useful for " +
			"discovering existing rules before adopting them into Terraform.",
		Attributes: map[string]schema.Attribute{
			"rules": schema.ListAttribute{
				Computed:    true,
				Description: "List of approval rules.",
				ElementType: ruleObjectType(),
			},
		},
	}
}

func (d *approvalRulesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.client = client
}

func (d *approvalRulesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	rules, err := d.client.ListRules(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading rules", err.Error())
		return
	}

	models := make([]approvalRuleModel, 0, len(rules))
	for i := range rules {
		var m approvalRuleModel
		if err := m.fromRule(ctx, &rules[i]); err != nil {
			resp.Diagnostics.AddError("Error storing rule state", err.Error())
			return
		}
		models = append(models, m)
	}

	state := approvalRulesModel{Rules: models}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
