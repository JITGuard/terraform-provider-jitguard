package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// machineAPIPrefix is the versioned public API prefix (#1544). The endpoint
	// configured on the provider is the API base URL, e.g.
	// https://jitguard.example.com — the client appends this path. CloudFront
	// rewrites /api/v1/customer/<rest> to the legacy gateway route
	// /api/customer/<rest>, so both aliases reach the same handler. The legacy
	// alias remains available until 2027-10-25.
	machineAPIPrefix = "/api/v1/customer/api"

	// rulesPath is the machine API rule CRUDL path (ADR-0015 / #523).
	rulesPath = machineAPIPrefix + "/rules"

	defaultRequestTimeout = 30 * time.Second
)

// Client is a small HTTP client for the JIT Guard machine API. It is not
// an AWS SDK surface — just the rules CRUDL endpoints authenticated with a
// bearer API key.
type Client struct {
	baseURL   string
	apiKey    string
	http      *http.Client
	userAgent string
}

// NewClient builds a Client. baseURL is the API origin (no path). skipVerify
// disables TLS certificate verification for local HTTPS testing.
func NewClient(baseURL, apiKey string, skipVerify bool) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if skipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402 — explicit opt-in for local testing
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http: &http.Client{
			Timeout:   defaultRequestTimeout,
			Transport: transport,
		},
	}
}

// Rule is the machine API rule object (POST/PUT body and GET response).
// Field names are the camelCase contract from the API; the Terraform layer
// maps these to snake_case attributes.
type Rule struct {
	RuleID              string                   `json:"ruleId,omitempty"`
	Name                string                   `json:"name"`
	Description         string                   `json:"description"`
	Enabled             bool                     `json:"enabled"`
	Outcome             string                   `json:"outcome"`
	ApproverGroupIds    []string                 `json:"approverGroupIds"`
	AutoApproveIfOnCall bool                     `json:"autoApproveIfOnCall"`
	SSOGroupIds         []string                 `json:"ssoGroupIds"`
	Accounts            []AccountCondition       `json:"accounts"`
	PermissionSets      []PermissionSetCondition `json:"permissionSets"`
	CreatedAt           string                   `json:"createdAt,omitempty"`
	UpdatedAt           string                   `json:"updatedAt,omitempty"`
	// ETag is the optimistic-concurrency tag from the response header; it is
	// not part of the JSON body.
	ETag string `json:"-"`
}

// AccountCondition is one entry of the rule's `accounts` array.
type AccountCondition struct {
	Type      string `json:"type"`
	AccountID string `json:"accountId,omitempty"`
	TagKey    string `json:"tagKey,omitempty"`
	TagValue  string `json:"tagValue,omitempty"`
	OUId      string `json:"ouId,omitempty"`
}

// PermissionSetCondition is one entry of the rule's `permissionSets` array.
type PermissionSetCondition struct {
	Type     string `json:"type"`
	ARN      string `json:"arn,omitempty"`
	TagKey   string `json:"tagKey,omitempty"`
	TagValue string `json:"tagValue,omitempty"`
}

// ListRulesResponse is the GET list payload.
type ListRulesResponse struct {
	Items      []Rule `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// APIError carries the machine API error shape: { error, message?, field? }.
type APIError struct {
	StatusCode int    `json:"-"`
	ErrorCode  string `json:"error"`
	Message    string `json:"message,omitempty"`
	Field      string `json:"field,omitempty"`
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = e.ErrorCode
	}
	if e.Field != "" {
		return fmt.Sprintf("jitguard API %d (%s): %s (field=%s)", e.StatusCode, e.ErrorCode, msg, e.Field)
	}
	return fmt.Sprintf("jitguard API %d (%s): %s", e.StatusCode, e.ErrorCode, msg)
}

// request performs an authenticated JSON request and decodes the response.
// okStatuses are the acceptable status codes for this call.
func (c *Client) request(ctx context.Context, method, path string, body any, okStatuses []int) (*http.Response, []byte, error) {
	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read response body: %w", err)
	}

	statusOK := false
	for _, s := range okStatuses {
		if resp.StatusCode == s {
			statusOK = true
			break
		}
	}
	if !statusOK {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		// Try to decode the structured error; fall back to raw body.
		if jsonErr := json.Unmarshal(respBody, apiErr); jsonErr != nil || apiErr.ErrorCode == "" {
			apiErr.ErrorCode = http.StatusText(resp.StatusCode)
			apiErr.Message = strings.TrimSpace(string(respBody))
		}
		return resp, respBody, apiErr
	}
	return resp, respBody, nil
}

// GetRule fetches a single rule. Returns (nil, nil) when not found.
func (c *Client) GetRule(ctx context.Context, ruleID string) (*Rule, error) {
	path := rulesPath + "/" + url.PathEscape(ruleID)
	resp, body, err := c.request(ctx, http.MethodGet, path, nil, []int{http.StatusOK, http.StatusNotFound})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	var rule Rule
	if err := json.Unmarshal(body, &rule); err != nil {
		return nil, fmt.Errorf("decode rule: %w", err)
	}
	return &rule, nil
}

// ListRules fetches every rule for the customer, following cursor pages.
func (c *Client) ListRules(ctx context.Context) ([]Rule, error) {
	var all []Rule
	cursor := ""
	for {
		path := rulesPath
		if cursor != "" {
			path += "?cursor=" + url.QueryEscape(cursor)
		}
		_, body, err := c.request(ctx, http.MethodGet, path, nil, []int{http.StatusOK})
		if err != nil {
			return nil, err
		}
		var page ListRulesResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("decode rules list: %w", err)
		}
		all = append(all, page.Items...)
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	return all, nil
}

// CreateRule POSTs a full rule and returns the stored rule with its ruleId.
func (c *Client) CreateRule(ctx context.Context, rule *Rule) (*Rule, error) {
	resp, body, err := c.request(ctx, http.MethodPost, rulesPath, rule, []int{http.StatusCreated, http.StatusOK})
	if err != nil {
		return nil, err
	}
	var created Rule
	if err := json.Unmarshal(body, &created); err != nil {
		return nil, fmt.Errorf("decode created rule: %w", err)
	}
	if created.RuleID == "" {
		created.RuleID = rule.RuleID
	}
	created.ETag = resp.Header.Get("ETag")
	return &created, nil
}

// UpdateRule PUTs a full replacement with optimistic concurrency. ifMatch is
// the current quoted ETag (e.g. `"2026-08-01T00:00:00Z"`).
func (c *Client) UpdateRule(ctx context.Context, ruleID, ifMatch string, rule *Rule) (*Rule, error) {
	path := rulesPath + "/" + url.PathEscape(ruleID)
	reqBody, err := json.Marshal(rule)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+path, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", ifMatch)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("PUT %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var updated Rule
		if err := json.Unmarshal(body, &updated); err != nil {
			return nil, fmt.Errorf("decode updated rule: %w", err)
		}
		updated.ETag = resp.Header.Get("ETag")
		return &updated, nil
	case http.StatusNotFound:
		return nil, nil
	case http.StatusPreconditionFailed:
		return nil, fmt.Errorf("rule %s changed on the server since it was read (If-Match precondition failed) — re-run terraform plan to reconcile", ruleID)
	case http.StatusPreconditionRequired:
		return nil, fmt.Errorf("rule %s requires an If-Match header (internal provider bug)", ruleID)
	default:
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if jsonErr := json.Unmarshal(body, apiErr); jsonErr != nil || apiErr.ErrorCode == "" {
			apiErr.ErrorCode = http.StatusText(resp.StatusCode)
			apiErr.Message = strings.TrimSpace(string(body))
		}
		return nil, apiErr
	}
}

// DeleteRule removes a rule with optimistic concurrency. Returns (true, nil)
// when deleted, (false, nil) when it was already gone.
func (c *Client) DeleteRule(ctx context.Context, ruleID, ifMatch string) (bool, error) {
	path := rulesPath + "/" + url.PathEscape(ruleID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return false, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("If-Match", ifMatch)

	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("DELETE %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("read response body: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	case http.StatusPreconditionFailed:
		return false, fmt.Errorf("rule %s changed on the server since it was read (If-Match precondition failed) — re-run terraform plan to reconcile", ruleID)
	default:
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if jsonErr := json.Unmarshal(body, apiErr); jsonErr != nil || apiErr.ErrorCode == "" {
			apiErr.ErrorCode = http.StatusText(resp.StatusCode)
			apiErr.Message = strings.TrimSpace(string(body))
		}
		return false, apiErr
	}
}

const (
	settingsPath     = machineAPIPrefix + "/settings"
	integrationsPath = machineAPIPrefix + "/integrations"
)

// Settings is the machine API settings response. It includes the AWS
// connection facts because the same route serves the aws_connection data
// source; the settings resource only manages the mutable fields.
type Settings struct {
	Name                string `json:"name"`
	ApprovalExpiryHours int64  `json:"approvalExpiryHours"`
	ConnectionStatus    string `json:"connectionStatus"`
	ManagementAccountID string `json:"managementAccountId"`
	SSORegion           string `json:"ssoRegion"`
	SSOInstanceARN      string `json:"ssoInstanceArn"`
}

// SettingsUpdate is a partial update. Nil fields are omitted, so a resource
// that only configures one attribute cannot clobber the others.
type SettingsUpdate struct {
	Name                *string `json:"name,omitempty"`
	ApprovalExpiryHours *int64  `json:"approvalExpiryHours,omitempty"`
}

// GetSettings fetches the customer settings. Returns (nil, nil) when the API
// reports the customer does not exist (404).
func (c *Client) GetSettings(ctx context.Context) (*Settings, error) {
	resp, body, err := c.request(ctx, http.MethodGet, settingsPath, nil, []int{http.StatusOK, http.StatusNotFound})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	var settings Settings
	if err := json.Unmarshal(body, &settings); err != nil {
		return nil, fmt.Errorf("decode settings: %w", err)
	}
	return &settings, nil
}

// UpdateSettings applies a partial settings update and returns the new state.
func (c *Client) UpdateSettings(ctx context.Context, update SettingsUpdate) (*Settings, error) {
	_, body, err := c.request(ctx, http.MethodPut, settingsPath, update, []int{http.StatusOK})
	if err != nil {
		return nil, err
	}
	var settings Settings
	if err := json.Unmarshal(body, &settings); err != nil {
		return nil, fmt.Errorf("decode settings: %w", err)
	}
	return &settings, nil
}

// SlackIntegration is the read-only Slack status. Slack is OAuth-installed and
// cannot be managed by the machine API.
type SlackIntegration struct {
	Connected bool   `json:"connected"`
	TeamID    string `json:"teamId"`
	TeamName  string `json:"teamName"`
}

// PagerDutyIntegration is the PagerDuty status. The key itself is never
// returned; masked_api_key is the only server-side view.
type PagerDutyIntegration struct {
	Connected    bool   `json:"connected"`
	MaskedAPIKey string `json:"maskedApiKey"`
}

// Integrations is the machine API integrations response.
type Integrations struct {
	Slack     SlackIntegration     `json:"slack"`
	PagerDuty PagerDutyIntegration `json:"pagerDuty"`
}

// GetIntegrations fetches the Slack and PagerDuty status.
func (c *Client) GetIntegrations(ctx context.Context) (*Integrations, error) {
	_, body, err := c.request(ctx, http.MethodGet, integrationsPath, nil, []int{http.StatusOK})
	if err != nil {
		return nil, err
	}
	var integrations Integrations
	if err := json.Unmarshal(body, &integrations); err != nil {
		return nil, fmt.Errorf("decode integrations: %w", err)
	}
	return &integrations, nil
}

// SetPagerDutyIntegration validates and stores the PagerDuty API key. The API
// validates the key against PagerDuty before accepting it.
func (c *Client) SetPagerDutyIntegration(ctx context.Context, apiKey string) (*Integrations, error) {
	return c.updateIntegrations(ctx, map[string]any{"pagerDutyApiKey": apiKey})
}

// DisconnectPagerDutyIntegration removes the stored PagerDuty key.
func (c *Client) DisconnectPagerDutyIntegration(ctx context.Context) (*Integrations, error) {
	return c.updateIntegrations(ctx, map[string]any{"disconnectPagerDuty": true})
}

func (c *Client) updateIntegrations(ctx context.Context, update map[string]any) (*Integrations, error) {
	_, body, err := c.request(ctx, http.MethodPut, integrationsPath, update, []int{http.StatusOK})
	if err != nil {
		return nil, err
	}
	var integrations Integrations
	if err := json.Unmarshal(body, &integrations); err != nil {
		return nil, fmt.Errorf("decode integrations: %w", err)
	}
	return &integrations, nil
}
