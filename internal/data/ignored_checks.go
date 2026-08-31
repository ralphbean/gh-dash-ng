package data

import (
	"strings"

	checks "github.com/dlvhdr/x/gh-checks"

	"github.com/dlvhdr/gh-dash/v4/internal/config"
)

func checkIdentity(parts ...string) string {
	identityParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" && trimmed != "/" {
			identityParts = append(identityParts, trimmed)
		}
	}
	return strings.Join(identityParts, "/")
}

func CheckRunIdentity(checkRun CheckRun) string {
	return checkIdentity(
		string(checkRun.CheckSuite.Creator.Login),
		string(checkRun.CheckSuite.WorkflowRun.Workflow.Name),
		string(checkRun.Name),
	)
}

func StatusContextIdentity(statusContext StatusContext) string {
	return checkIdentity(string(statusContext.Creator.Login), string(statusContext.Context))
}

func CheckSuiteIdentity(suite CheckSuiteNode) string {
	workflow := strings.TrimSpace(string(suite.WorkflowRun.Workflow.Name))
	if workflow != "" {
		return workflow
	}
	return strings.TrimSpace(string(suite.App.Name))
}

func IsCheckRunIgnored(patterns []string, checkRun CheckRun) bool {
	return config.IsCheckIgnored(patterns, CheckRunIdentity(checkRun))
}

func IsStatusContextIgnored(patterns []string, statusContext StatusContext) bool {
	return config.IsCheckIgnored(patterns, StatusContextIdentity(statusContext))
}

func IsCheckSuiteIgnored(patterns []string, suite CheckSuiteNode) bool {
	return config.IsCheckIgnored(patterns, CheckSuiteIdentity(suite))
}

func FilteredStatusCheckRollup(
	commits LastCommitStatusContexts,
	patterns []string,
) checks.CommitState {
	if len(commits.Nodes) == 0 {
		return checks.CommitStateUnknown
	}

	state := checks.CommitStateUnknown
	for _, node := range commits.Nodes[0].Commit.StatusCheckRollup.Contexts.Nodes {
		var rawState string
		switch node.Typename {
		case "CheckRun":
			if IsCheckRunIgnored(patterns, node.CheckRun) {
				continue
			}
			rawState = string(node.CheckRun.Conclusion)
			if checks.IsStatusWaiting(string(node.CheckRun.Status)) {
				rawState = string(node.CheckRun.Status)
			}
		case "StatusContext":
			if IsStatusContextIgnored(patterns, node.StatusContext) {
				continue
			}
			rawState = string(node.StatusContext.State)
		default:
			continue
		}

		if checks.IsConclusionAFailure(rawState) {
			return checks.CommitStateFailure
		}
		if checks.IsStatusWaiting(rawState) {
			state = checks.CommitStatePending
		} else if state == checks.CommitStateUnknown {
			state = checks.CommitStateSuccess
		}
	}
	return state
}
