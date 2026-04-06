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

package helpers

import (
	"net/http"
	"strings"
	"time"

	apperrors "github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/errors"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/executor"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/openapi"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	RegisterPublic(func() Handler {
		return oaHandler{}
	})
}

type oaHandler struct{}

func (oaHandler) Name() string {
	return "oa"
}

func (oaHandler) Command(_ executor.Runner) *cobra.Command {
	root := &cobra.Command{
		Use:               "oa",
		Short:             "OA 审批辅助命令",
		Args:              cobra.NoArgs,
		TraverseChildren:  true,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	approval := &cobra.Command{
		Use:               "approval",
		Short:             "审批辅助命令",
		Args:              cobra.NoArgs,
		TraverseChildren:  true,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	approval.AddCommand(newOAApprovalDetailOpenAPICommand())

	root.AddCommand(approval)
	return root
}

func newOAApprovalDetailOpenAPICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "detail",
		Aliases:           []string{"detail-openapi"},
		Short:             "通过官方 OpenAPI 获取审批实例详情",
		Long:              "调用钉钉官方 OpenAPI 获取单个审批实例详情，返回完整审批数据，包括 formComponentValues 等表单字段。",
		Example:           "  dws oa approval detail --instance-id PROC-123 --format json",
		Args:              cobra.NoArgs,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceID, _ := cmd.Flags().GetString("instance-id")
			instanceID = strings.TrimSpace(instanceID)
			if instanceID == "" {
				return apperrors.NewValidation("--instance-id is required")
			}

			requestPreview := map[string]any{
				"method": "GET",
				"url":    openapi.ResolveBaseURL() + openapi.OAApprovalDetailAPIPath,
				"query": map[string]any{
					"processInstanceId": instanceID,
				},
				"headers": map[string]any{
					"x-acs-dingtalk-access-token": "[REDACTED]",
					"Content-Type":                "application/json",
				},
			}

			if commandDryRun(cmd) {
				return writeCommandPayload(cmd, map[string]any{
					"dry_run": true,
					"request": requestPreview,
					"note":    "execution skipped by --dry-run",
				})
			}

			explicitToken, err := lookupStringFlag(cmd, "token")
			if err != nil {
				return apperrors.NewInternal("failed to read --token")
			}
			token, err := openapi.ResolveAccessToken(cmd.Context(), explicitToken)
			if err != nil {
				return err
			}

			timeoutSeconds, err := lookupIntFlag(cmd, "timeout", 30)
			if err != nil {
				return apperrors.NewInternal("failed to read --timeout")
			}

			client := &openapi.OAClient{
				BaseURL:     openapi.ResolveBaseURL(),
				AccessToken: token,
				HTTPClient: &http.Client{
					Timeout: time.Duration(timeoutSeconds) * time.Second,
				},
			}

			payload, err := client.GetApprovalDetail(cmd.Context(), instanceID)
			if err != nil {
				return err
			}
			return writeCommandPayload(cmd, payload)
		},
	}
	cmd.Flags().String("instance-id", "", "审批实例 ID (必填)")
	preferLegacyLeaf(cmd)
	return cmd
}

func lookupStringFlag(cmd *cobra.Command, name string) (string, error) {
	for _, flags := range []*pflag.FlagSet{cmd.Flags(), cmd.InheritedFlags(), rootPersistentFlags(cmd)} {
		if flags == nil || flags.Lookup(name) == nil {
			continue
		}
		return flags.GetString(name)
	}
	return "", nil
}

func lookupIntFlag(cmd *cobra.Command, name string, fallback int) (int, error) {
	for _, flags := range []*pflag.FlagSet{cmd.Flags(), cmd.InheritedFlags(), rootPersistentFlags(cmd)} {
		if flags == nil || flags.Lookup(name) == nil {
			continue
		}
		return flags.GetInt(name)
	}
	return fallback, nil
}

func rootPersistentFlags(cmd *cobra.Command) *pflag.FlagSet {
	if cmd == nil || cmd.Root() == nil {
		return nil
	}
	return cmd.Root().PersistentFlags()
}
