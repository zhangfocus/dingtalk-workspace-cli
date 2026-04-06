package cli_compat_test

import "testing"

func TestOAApprovalDetailOpenAPI_should_support_dry_run_without_mcp_call(t *testing.T) {
	cap := setupTestDepsWithDryRun(t, "oa")
	root := buildRoot()
	err := execCmd(t, root, []string{"oa", "approval", "detail"}, map[string]string{
		"instance-id": "PROC-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertCallCount(t, cap, 0)
}
