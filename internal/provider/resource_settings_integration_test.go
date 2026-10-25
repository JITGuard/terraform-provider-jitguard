package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// fakeConfigAPI is an in-memory settings + integrations API. It keeps the
// mutable state so create/update/import can be exercised through the real
// Terraform plugin protocol.
type fakeConfigAPI struct {
	mu           sync.Mutex
	settings     Settings
	integrations Integrations
}

func newFakeConfigAPI() *fakeConfigAPI {
	return &fakeConfigAPI{
		settings: Settings{
			Name:                "Acme",
			ApprovalExpiryHours: 8,
			ConnectionStatus:    "connected",
			ManagementAccountID: "111111111111",
			SSORegion:           "eu-west-1",
			SSOInstanceARN:      "arn:aws:sso:::instance/ssoins-1111111111111111",
		},
		integrations: Integrations{
			Slack:     SlackIntegration{Connected: true, TeamID: "T12345", TeamName: "acme"},
			PagerDuty: PagerDutyIntegration{Connected: false, MaskedAPIKey: ""},
		},
	}
}

func (f *fakeConfigAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+testAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthenticated"})
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/customer/api/settings":
		json.NewEncoder(w).Encode(f.settings)
	case r.Method == http.MethodPut && r.URL.Path == "/api/v1/customer/api/settings":
		var update SettingsUpdate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if update.Name != nil {
			f.settings.Name = *update.Name
		}
		if update.ApprovalExpiryHours != nil {
			f.settings.ApprovalExpiryHours = *update.ApprovalExpiryHours
		}
		json.NewEncoder(w).Encode(f.settings)
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/customer/api/integrations":
		json.NewEncoder(w).Encode(f.integrations)
	case r.Method == http.MethodPut && r.URL.Path == "/api/v1/customer/api/integrations":
		var update map[string]any
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if key, ok := update["pagerDutyApiKey"].(string); ok {
			f.integrations.PagerDuty.Connected = true
			if len(key) >= 4 {
				f.integrations.PagerDuty.MaskedAPIKey = "****" + key[len(key)-4:]
			} else {
				f.integrations.PagerDuty.MaskedAPIKey = "****"
			}
		}
		if disconnect, ok := update["disconnectPagerDuty"].(bool); ok && disconnect {
			f.integrations.PagerDuty.Connected = false
			f.integrations.PagerDuty.MaskedAPIKey = ""
		}
		json.NewEncoder(w).Encode(f.integrations)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestOrganizationSettingsResource(t *testing.T) {
	fake := newFakeConfigAPI()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	cfg := providerConfig(srv.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "jitguard_organization_settings" "this" {
  name                  = "Acme Corp"
  approval_expiry_hours = 12
}

data "jitguard_aws_connection" "this" {}

data "jitguard_integrations" "this" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jitguard_organization_settings.this", "id", "settings"),
					resource.TestCheckResourceAttr("jitguard_organization_settings.this", "name", "Acme Corp"),
					resource.TestCheckResourceAttr("jitguard_organization_settings.this", "approval_expiry_hours", "12"),
					resource.TestCheckResourceAttr("data.jitguard_aws_connection.this", "connection_status", "connected"),
					resource.TestCheckResourceAttr("data.jitguard_aws_connection.this", "management_account_id", "111111111111"),
					resource.TestCheckResourceAttr("data.jitguard_aws_connection.this", "sso_region", "eu-west-1"),
					resource.TestCheckResourceAttr("data.jitguard_integrations.this", "slack.connected", "true"),
					resource.TestCheckResourceAttr("data.jitguard_integrations.this", "slack.team_name", "acme"),
					resource.TestCheckResourceAttr("data.jitguard_integrations.this", "pagerduty.connected", "false"),
				),
			},
			{
				Config: cfg + `
resource "jitguard_organization_settings" "this" {
  name                  = "Acme Corp (updated)"
  approval_expiry_hours = 24
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jitguard_organization_settings.this", "name", "Acme Corp (updated)"),
					resource.TestCheckResourceAttr("jitguard_organization_settings.this", "approval_expiry_hours", "24"),
				),
			},
			{
				ResourceName:      "jitguard_organization_settings.this",
				ImportState:       true,
				ImportStateId:     "settings",
				ImportStateVerify: true,
			},
		},
	})
}

func TestPagerDutyIntegrationResource(t *testing.T) {
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
  api_key         = "pd-key-1234"
  api_key_version = 0
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jitguard_integration_pagerduty.this", "id", "pagerduty"),
					resource.TestCheckResourceAttr("jitguard_integration_pagerduty.this", "connected", "true"),
					resource.TestCheckResourceAttr("jitguard_integration_pagerduty.this", "masked_api_key", "****1234"),
					resource.TestCheckResourceAttr("jitguard_integration_pagerduty.this", "api_key_version", "0"),
				),
			},
			{
				// Rotate the write-only key by bumping the version.
				Config: cfg + `
resource "jitguard_integration_pagerduty" "this" {
  api_key         = "pd-key-5678"
  api_key_version = 1
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("jitguard_integration_pagerduty.this", "connected", "true"),
					resource.TestCheckResourceAttr("jitguard_integration_pagerduty.this", "masked_api_key", "****5678"),
					resource.TestCheckResourceAttr("jitguard_integration_pagerduty.this", "api_key_version", "1"),
				),
			},
			{
				ResourceName:            "jitguard_integration_pagerduty.this",
				ImportState:             true,
				ImportStateId:           "pagerduty",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key", "api_key_version"},
			},
		},
	})
}

func TestPagerDutyIntegrationResource_RequiresKey(t *testing.T) {
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
  api_key_version = 0
}
`,
				ExpectError: regexp.MustCompile("api_key is required"),
			},
		},
	})
}
