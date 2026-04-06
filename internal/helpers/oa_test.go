package helpers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/openapi"
	"github.com/spf13/cobra"
)

func TestOAApprovalDetailOpenAPICommandCallsOfficialAPI(t *testing.T) {
	var gotHeader string
	var gotInstanceID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("x-acs-dingtalk-access-token")
		gotInstanceID = r.URL.Query().Get("processInstanceId")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":{"processInstanceId":"proc-123","formComponentValues":[{"name":"请假类型","value":"年假"}]}}`))
	}))
	defer server.Close()
	t.Setenv(openapi.OAOpenAPIBaseURLEnv, server.URL)

	root := buildOATestRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"oa", "approval", "detail", "--instance-id", "proc-123", "--token", "test-token"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\noutput:\n%s", err, out.String())
	}

	if gotHeader != "test-token" {
		t.Fatalf("token header = %q, want test-token", gotHeader)
	}
	if gotInstanceID != "proc-123" {
		t.Fatalf("processInstanceId = %q, want proc-123", gotInstanceID)
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode output: %v\n%s", err, out.String())
	}
	if payload["success"] != true {
		t.Fatalf("success = %v, want true", payload["success"])
	}
	result, ok := payload["result"].(map[string]any)
	if !ok {
		t.Fatalf("result missing or wrong type: %#v", payload["result"])
	}
	if result["processInstanceId"] != "proc-123" {
		t.Fatalf("result.processInstanceId = %v, want proc-123", result["processInstanceId"])
	}
}

func TestOAApprovalDetailOpenAPICommandSupportsDryRun(t *testing.T) {
	root := buildOATestRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--dry-run", "oa", "approval", "detail", "--instance-id", "proc-123"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\noutput:\n%s", err, out.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode dry-run output: %v\n%s", err, out.String())
	}
	if payload["dry_run"] != true {
		t.Fatalf("dry_run = %v, want true", payload["dry_run"])
	}
}

func TestOAApprovalDetailOpenAPICommandRequiresInstanceID(t *testing.T) {
	root := buildOATestRoot()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"oa", "approval", "detail", "--token", "test-token"})

	if err := root.Execute(); err == nil {
		t.Fatal("expected validation error when --instance-id is missing")
	}
}

func TestOAApprovalDetailOpenAPICommandMapsUnauthorizedToAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"token invalid"}`))
	}))
	defer server.Close()
	t.Setenv(openapi.OAOpenAPIBaseURLEnv, server.URL)

	root := buildOATestRoot()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"oa", "approval", "detail", "--instance-id", "proc-123", "--token", "bad-token"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected auth error")
	}
	if !strings.Contains(err.Error(), "token invalid") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func buildOATestRoot() *cobra.Command {
	root := &cobra.Command{Use: "dws"}
	root.PersistentFlags().String("token", "", "")
	root.PersistentFlags().Int("timeout", 30, "")
	root.PersistentFlags().StringP("format", "f", "json", "")
	root.PersistentFlags().Bool("dry-run", false, "")
	root.AddCommand(oaHandler{}.Command(nil))
	return root
}
