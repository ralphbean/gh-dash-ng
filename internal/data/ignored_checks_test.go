package data

import (
	"testing"

	graphql "github.com/cli/shurcooL-graphql"
	checks "github.com/dlvhdr/x/gh-checks"
	"github.com/stretchr/testify/require"
)

func TestCheckIdentities(t *testing.T) {
	t.Parallel()

	checkRun := CheckRun{Name: "dispatch-pr"}
	checkRun.CheckSuite.Creator.Login = "fullsend-ai-bot"
	checkRun.CheckSuite.WorkflowRun.Workflow.Name = "fullsend"
	require.Equal(t, "fullsend-ai-bot/fullsend/dispatch-pr", CheckRunIdentity(checkRun))

	status := StatusContext{Context: "ci/test"}
	status.Creator.Login = "octocat"
	require.Equal(t, "octocat/ci/test", StatusContextIdentity(status))

	suite := CheckSuiteNode{}
	suite.WorkflowRun.Workflow.Name = "build"
	require.Equal(t, "build", CheckSuiteIdentity(suite))
	suite.WorkflowRun.Workflow.Name = ""
	suite.App.Name = "GitHub Actions"
	require.Equal(t, "GitHub Actions", CheckSuiteIdentity(suite))
}

func TestFilteredStatusCheckRollup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		nodes    []StatusCheckContextNode
		patterns []string
		want     checks.CommitState
	}{
		{
			name: "ignored failure leaves success",
			nodes: []StatusCheckContextNode{
				checkRunNode(
					"fullsend-ai-bot", "fullsend", "dispatch-pr", "COMPLETED",
					checks.CheckRunStateFailure,
				),
				checkRunNode(
					"github-actions", "ci", "test", "COMPLETED",
					checks.CheckRunStateSuccess,
				),
			},
			patterns: []string{"fullsend-ai-*/fullsend/dispatch*"},
			want:     checks.CommitStateSuccess,
		},
		{
			name: "non ignored failure wins",
			nodes: []StatusCheckContextNode{
				checkRunNode(
					"fullsend-ai-bot", "fullsend", "dispatch-pr", "COMPLETED",
					checks.CheckRunStateFailure,
				),
				checkRunNode(
					"github-actions", "ci", "test", "COMPLETED",
					checks.CheckRunStateFailure,
				),
			},
			patterns: []string{"fullsend-ai-*/fullsend/dispatch*"},
			want:     checks.CommitStateFailure,
		},
		{
			name: "all ignored is unknown",
			nodes: []StatusCheckContextNode{checkRunNode(
				"fullsend-ai-bot", "fullsend", "dispatch-pr", "COMPLETED",
				checks.CheckRunStateFailure,
			)},
			patterns: []string{"fullsend-ai-*/fullsend/dispatch*"},
			want:     checks.CommitStateUnknown,
		},
		{
			name: "pending beats success",
			nodes: []StatusCheckContextNode{checkRunNode(
				"github-actions", "ci", "test", "IN_PROGRESS", "",
			)},
			want: checks.CommitStatePending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			commits := LastCommitStatusContexts{}
			commits.Nodes = append(commits.Nodes, struct {
				Commit struct {
					StatusCheckRollup struct {
						Contexts struct {
							Nodes []StatusCheckContextNode
						} `graphql:"contexts(last: 100)"`
					}
				}
			}{})
			commits.Nodes[0].Commit.StatusCheckRollup.Contexts.Nodes = tt.nodes
			require.Equal(t, tt.want, FilteredStatusCheckRollup(commits, tt.patterns))
		})
	}
}

func checkRunNode(
	creator string,
	workflow string,
	name string,
	status string,
	conclusion checks.CheckRunState,
) StatusCheckContextNode {
	node := StatusCheckContextNode{Typename: "CheckRun"}
	node.CheckRun.Name = graphql.String(name)
	node.CheckRun.Status = graphql.String(status)
	node.CheckRun.Conclusion = conclusion
	node.CheckRun.CheckSuite.Creator.Login = graphql.String(creator)
	node.CheckRun.CheckSuite.WorkflowRun.Workflow.Name = graphql.String(workflow)
	return node
}
