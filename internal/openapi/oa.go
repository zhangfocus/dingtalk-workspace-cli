// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	authpkg "github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/auth"
	apperrors "github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/errors"
)

const (
	OAOpenAPIBaseURL        = "https://api.dingtalk.com"
	OAOpenAPIBaseURLEnv     = "DINGTALK_OPENAPI_BASE_URL"
	OAApprovalDetailAPIPath = "/v1.0/workflow/processInstances"
	OAAppAccessTokenPath    = "/v1.0/oauth2/accessToken"
)

type OAClient struct {
	BaseURL     string
	AccessToken string
	HTTPClient  *http.Client
}

func (c *OAClient) GetApprovalDetail(ctx context.Context, instanceID string) (map[string]any, error) {
	baseURL := strings.TrimSpace(c.BaseURL)
	if baseURL == "" {
		baseURL = OAOpenAPIBaseURL
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, apperrors.NewInternal(fmt.Sprintf("invalid OA OpenAPI base URL: %v", err))
	}
	parsedURL.Path = strings.TrimRight(parsedURL.Path, "/") + OAApprovalDetailAPIPath

	query := parsedURL.Query()
	query.Set("processInstanceId", instanceID)
	parsedURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, apperrors.NewInternal(fmt.Sprintf("failed to create OA approval detail request: %v", err))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-acs-dingtalk-access-token", c.AccessToken)

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, apperrors.NewAPI(
			fmt.Sprintf("failed to call OA approval detail OpenAPI: %v", err),
			apperrors.WithOperation("GET "+OAApprovalDetailAPIPath),
			apperrors.WithReason("openapi_request_failed"),
			apperrors.WithServerKey("oa"),
			apperrors.WithRetryable(true),
		)
	}
	defer resp.Body.Close()

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, apperrors.NewAPI(
			fmt.Sprintf("failed to decode OA approval detail response: %v", err),
			apperrors.WithOperation("GET "+OAApprovalDetailAPIPath),
			apperrors.WithReason("openapi_decode_failed"),
			apperrors.WithServerKey("oa"),
		)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return payload, nil
	}

	message := ExtractOAOpenAPIErrorMessage(resp.StatusCode, payload)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, apperrors.NewAuth(
			message,
			apperrors.WithOperation("GET "+OAApprovalDetailAPIPath),
			apperrors.WithReason("openapi_auth_failed"),
			apperrors.WithHint("确认 dws 已登录，或用 --token 指定有效的 access token"),
		)
	}

	return nil, apperrors.NewAPI(
		message,
		apperrors.WithOperation("GET "+OAApprovalDetailAPIPath),
		apperrors.WithReason("openapi_http_error"),
		apperrors.WithServerKey("oa"),
	)
}

func ResolveAccessToken(ctx context.Context, explicitToken string) (string, error) {
	if strings.TrimSpace(explicitToken) != "" {
		return strings.TrimSpace(explicitToken), nil
	}

	clientID, clientSecret := resolveAppCredentials()
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" {
		return "", apperrors.NewAuth(
			"未找到企业应用凭证，请先配置应用 AppKey/AppSecret",
			apperrors.WithReason("missing_app_credentials"),
			apperrors.WithHint("确认已通过 --client-id/--client-secret、环境变量或 ~/.dws/app.json 配置当前企业应用凭证"),
		)
	}

	token, err := fetchAppAccessToken(ctx, ResolveBaseURL(), clientID, clientSecret)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(token), nil
}

func resolveAppCredentials() (clientID, clientSecret string) {
	if id, secret := authpkg.ResolveAppCredentials(configDir()); strings.TrimSpace(id) != "" && strings.TrimSpace(secret) != "" {
		return strings.TrimSpace(id), strings.TrimSpace(secret)
	}

	// Recover credentials from the logged-in token snapshot when app.json is absent.
	if tokenData, err := authpkg.LoadTokenData(configDir()); err == nil && tokenData != nil {
		id := strings.TrimSpace(tokenData.ClientID)
		secret := strings.TrimSpace(authpkg.LoadClientSecret(id))
		if id != "" && secret != "" {
			return id, secret
		}
	}

	// Fall back to the same resolution chain used by the main auth flow:
	// runtime flags -> app config -> env -> compiled defaults.
	id := strings.TrimSpace(authpkg.ClientID())
	secret := strings.TrimSpace(authpkg.ClientSecret())
	if id == "" || secret == "" || strings.HasPrefix(secret, "<") {
		return "", ""
	}
	return id, secret
}

func fetchAppAccessToken(ctx context.Context, baseURL, appKey, appSecret string) (string, error) {
	endpoint, err := url.Parse(strings.TrimRight(baseURL, "/") + OAAppAccessTokenPath)
	if err != nil {
		return "", apperrors.NewInternal(fmt.Sprintf("invalid OA OpenAPI base URL: %v", err))
	}
	body, err := json.Marshal(map[string]any{
		"appKey":    appKey,
		"appSecret": appSecret,
	})
	if err != nil {
		return "", apperrors.NewInternal(fmt.Sprintf("failed to encode app access token request: %v", err))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return "", apperrors.NewInternal(fmt.Sprintf("failed to create app access token request: %v", err))
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", apperrors.NewAuth(
			fmt.Sprintf("failed to fetch app access token: %v", err),
			apperrors.WithReason("app_access_token_request_failed"),
			apperrors.WithOperation("POST "+OAAppAccessTokenPath),
		)
	}
	defer resp.Body.Close()

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", apperrors.NewAuth(
			fmt.Sprintf("failed to decode app access token response: %v", err),
			apperrors.WithReason("app_access_token_decode_failed"),
			apperrors.WithOperation("POST "+OAAppAccessTokenPath),
		)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", apperrors.NewAuth(
			ExtractOAOpenAPIErrorMessage(resp.StatusCode, payload),
			apperrors.WithReason("app_access_token_http_error"),
			apperrors.WithOperation("POST "+OAAppAccessTokenPath),
		)
	}

	for _, key := range []string{"accessToken", "access_token"} {
		if token, ok := payload[key].(string); ok && strings.TrimSpace(token) != "" {
			return token, nil
		}
	}

	return "", apperrors.NewAuth(
		"app access token response missing accessToken",
		apperrors.WithReason("missing_app_access_token"),
		apperrors.WithOperation("POST "+OAAppAccessTokenPath),
	)
}

func ResolveBaseURL() string {
	if value := strings.TrimSpace(os.Getenv(OAOpenAPIBaseURLEnv)); value != "" {
		return strings.TrimRight(value, "/")
	}
	return OAOpenAPIBaseURL
}

func ExtractOAOpenAPIErrorMessage(statusCode int, payload map[string]any) string {
	for _, key := range []string{"message", "errorMsg", "errmsg", "msg"} {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	if code, ok := payload["errcode"]; ok {
		return fmt.Sprintf("OA OpenAPI request failed with status %d (errcode=%v)", statusCode, code)
	}
	return fmt.Sprintf("OA OpenAPI request failed with status %d", statusCode)
}

func configDir() string {
	if envDir := os.Getenv("DWS_CONFIG_DIR"); envDir != "" {
		return envDir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".dws"
	}
	return filepath.Join(homeDir, ".dws")
}
