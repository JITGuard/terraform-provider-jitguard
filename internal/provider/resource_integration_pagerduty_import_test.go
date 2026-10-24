package provider

import (
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// A freshly imported PagerDuty integration has no client-side rotation
// version. Read must default it to 0 so the imported state matches the schema
// default; otherwise Terraform reports "Provider produced inconsistent result
// after apply" when the import and update are combined.
func TestPagerDutyIntegrationResource_ImportDefaultsVersion(t *testing.T) {
	fake := newFakeConfigAPI()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	cfg := providerConfig(srv.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "jitguard_integration_pagerduty" "this" {
  api_key = "pd-key-1234"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jitguard_integration_pagerduty.this", "connected", "true"),
					resource.TestCheckResourceAttr("jitguard_integration_pagerduty.this", "api_key_version", "0"),
				),
			},
			{
				ResourceName:            "jitguard_integration_pagerduty.this",
				ImportState:             true,
				ImportStateId:           "pagerduty",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key"},
			},
		},
	})
}
