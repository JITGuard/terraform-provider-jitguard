package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testAPIKey = "bg_sk_live_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA_TgAAAA"

func newTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, testAPIKey, false), srv
}

func testRule() Rule {
	return Rule{
		RuleID:              "rule-1",
		Name:                "SRE Emergency",
		Description:         "test",
		Enabled:             true,
		Outcome:             "auto_approve",
		ApproverGroupIds:    []string{},
		AutoApproveIfOnCall: true,
		SSOGroupIds:         []string{"grp-sre"},
		Accounts: []AccountCondition{
			{Type: "account_id", AccountID: "111111111111"},
		},
		PermissionSets: []PermissionSetCondition{
			{Type: "permission_set_arn", ARN: "arn:aws:sso:::permissionSet/ssoins-1111111111111111/ps-22222222222222222222222222222222"},
		},
		CreatedAt: "2026-08-01T00:00:00Z",
		UpdatedAt: "2026-08-01T00:00:00Z",
	}
}

func TestClient_GetRule(t *testing.T) {
	rule := testRule()
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/api/customer/api/rules/rule-1" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("ETag", `"2026-08-01T00:00:00Z"`)
		json.NewEncoder(w).Encode(rule)
	})

	got, err := client.GetRule(context.Background(), "rule-1")
	if err != nil {
		t.Fatalf("GetRule: %v", err)
	}
	if got == nil {
		t.Fatal("GetRule returned nil")
	}
	if got.RuleID != "rule-1" || got.Name != "SRE Emergency" {
		t.Fatalf("unexpected rule: %+v", got)
	}
}

func TestClient_GetRule_NotFound(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	got, err := client.GetRule(context.Background(), "missing")
	if err != nil {
		t.Fatalf("GetRule: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for 404, got %+v", got)
	}
}

func TestClient_GetRule_Unauthorized(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthenticated", "message": "Unknown API key"})
	})

	_, err := client.GetRule(context.Background(), "rule-1")
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "Unknown API key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_ListRules_Pagination(t *testing.T) {
	page1 := ListRulesResponse{
		Items:      []Rule{testRule()},
		NextCursor: "cursor-2",
	}
	page2 := ListRulesResponse{Items: []Rule{}}

	calls := 0
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("cursor") == "cursor-2" {
			json.NewEncoder(w).Encode(page2)
			return
		}
		json.NewEncoder(w).Encode(page1)
	})

	rules, err := client.ListRules(context.Background())
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if calls != 2 {
		t.Fatalf("expected 2 list calls (pagination), got %d", calls)
	}
}

func TestClient_CreateRule(t *testing.T) {
	rule := testRule()
	created := rule
	created.RuleID = "new-uuid"

	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/api/customer/api/rules" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body Rule
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Name != "SRE Emergency" {
			t.Errorf("expected name in body, got %+v", body)
		}
		w.Header().Set("ETag", `"2026-08-01T00:00:00Z"`)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)
	})

	got, err := client.CreateRule(context.Background(), &rule)
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	if got.RuleID != "new-uuid" {
		t.Fatalf("expected ruleId new-uuid, got %q", got.RuleID)
	}
	if got.ETag != `"2026-08-01T00:00:00Z"` {
		t.Fatalf("expected ETag header captured, got %q", got.ETag)
	}
}

func TestClient_UpdateRule_IfMatchSent(t *testing.T) {
	updated := testRule()
	updated.UpdatedAt = "2026-08-02T00:00:00Z"

	var gotIfMatch string
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		gotIfMatch = r.Header.Get("If-Match")
		w.Header().Set("ETag", `"2026-08-02T00:00:00Z"`)
		json.NewEncoder(w).Encode(updated)
	})

	got, err := client.UpdateRule(context.Background(), "rule-1", `"2026-08-01T00:00:00Z"`, &updated)
	if err != nil {
		t.Fatalf("UpdateRule: %v", err)
	}
	if gotIfMatch != `"2026-08-01T00:00:00Z"` {
		t.Fatalf("expected If-Match header sent, got %q", gotIfMatch)
	}
	if got.UpdatedAt != "2026-08-02T00:00:00Z" {
		t.Fatalf("unexpected updated rule: %+v", got)
	}
}

func TestClient_UpdateRule_PreconditionFailed(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPreconditionFailed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Precondition Failed", "message": "Rule changed since it was read"})
	})

	_, err := client.UpdateRule(context.Background(), "rule-1", `"stale"`, &Rule{})
	if err == nil {
		t.Fatal("expected 412 to surface as an error")
	}
	if !strings.Contains(err.Error(), "precondition failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_DeleteRule(t *testing.T) {
	var gotIfMatch string
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		gotIfMatch = r.Header.Get("If-Match")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})

	deleted, err := client.DeleteRule(context.Background(), "rule-1", `"2026-08-01T00:00:00Z"`)
	if err != nil {
		t.Fatalf("DeleteRule: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted=true")
	}
	if gotIfMatch != `"2026-08-01T00:00:00Z"` {
		t.Fatalf("expected If-Match header sent, got %q", gotIfMatch)
	}
}

func TestClient_DeleteRule_NotFound(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	deleted, err := client.DeleteRule(context.Background(), "missing", `"*"`)
	if err != nil {
		t.Fatalf("DeleteRule: %v", err)
	}
	if deleted {
		t.Fatal("expected deleted=false for 404")
	}
}

func TestClient_GetSettings(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/customer/api/settings" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(Settings{
			Name:                "Acme",
			NameSet:             true,
			ApprovalExpiryHours: 12,
			PermissionSetARN:    "arn:aws:sso:::permissionSet/ps-1",
			ConnectionStatus:    "connected",
			ManagementAccountID: "111111111111",
			SSORegion:           "eu-west-1",
			SSOInstanceARN:      "arn:aws:sso:::instance/ssoins-1",
		})
	})

	settings, err := client.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings == nil {
		t.Fatal("GetSettings returned nil")
	}
	if settings.Name != "Acme" || settings.ApprovalExpiryHours != 12 || settings.ConnectionStatus != "connected" {
		t.Fatalf("unexpected settings: %+v", settings)
	}
}

func TestClient_GetSettings_NotFound(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	settings, err := client.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings != nil {
		t.Fatalf("expected nil for 404, got %+v", settings)
	}
}

func TestClient_UpdateSettings_PartialBody(t *testing.T) {
	var body map[string]any
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/customer/api/settings" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		json.NewEncoder(w).Encode(Settings{Name: "Acme", ApprovalExpiryHours: 12})
	})

	expiry := int64(12)
	if _, err := client.UpdateSettings(context.Background(), SettingsUpdate{ApprovalExpiryHours: &expiry}); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if len(body) != 1 || body["approvalExpiryHours"] != float64(12) {
		t.Fatalf("expected only approvalExpiryHours, got %#v", body)
	}
	if _, ok := body["name"]; ok {
		t.Fatalf("nil name must be omitted, got %#v", body)
	}
}

func TestClient_UpdateSettings_ClearsPermissionSet(t *testing.T) {
	var body map[string]any
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		json.NewEncoder(w).Encode(Settings{})
	})

	empty := ""
	if _, err := client.UpdateSettings(context.Background(), SettingsUpdate{PermissionSetARN: &empty}); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if body["permissionSetArn"] != "" {
		t.Fatalf("expected an explicit empty permissionSetArn, got %#v", body)
	}
}

func TestClient_GetIntegrations(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/customer/api/integrations" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(Integrations{
			Slack:     SlackIntegration{Connected: true, TeamID: "T1", TeamName: "acme"},
			PagerDuty: PagerDutyIntegration{Connected: true, MaskedAPIKey: "****-123"},
		})
	})

	integrations, err := client.GetIntegrations(context.Background())
	if err != nil {
		t.Fatalf("GetIntegrations: %v", err)
	}
	if !integrations.Slack.Connected || integrations.Slack.TeamName != "acme" {
		t.Fatalf("unexpected slack: %+v", integrations.Slack)
	}
	if !integrations.PagerDuty.Connected || integrations.PagerDuty.MaskedAPIKey != "****-123" {
		t.Fatalf("unexpected pagerduty: %+v", integrations.PagerDuty)
	}
}

func TestClient_SetPagerDutyIntegration(t *testing.T) {
	var body map[string]any
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/customer/api/integrations" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		json.NewDecoder(r.Body).Decode(&body)
		json.NewEncoder(w).Encode(Integrations{PagerDuty: PagerDutyIntegration{Connected: true}})
	})

	if _, err := client.SetPagerDutyIntegration(context.Background(), "pd-key"); err != nil {
		t.Fatalf("SetPagerDutyIntegration: %v", err)
	}
	if body["pagerDutyApiKey"] != "pd-key" {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestClient_DisconnectPagerDutyIntegration(t *testing.T) {
	var body map[string]any
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		json.NewEncoder(w).Encode(Integrations{})
	})

	if _, err := client.DisconnectPagerDutyIntegration(context.Background()); err != nil {
		t.Fatalf("DisconnectPagerDutyIntegration: %v", err)
	}
	if body["disconnectPagerDuty"] != true {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestClient_SetPagerDutyIntegration_InvalidKey(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Bad Request", "message": "Invalid PagerDuty API key"})
	})

	_, err := client.SetPagerDutyIntegration(context.Background(), "bad")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "Invalid PagerDuty API key") {
		t.Fatalf("unexpected error: %v", err)
	}
}
