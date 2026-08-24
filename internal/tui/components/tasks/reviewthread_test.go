package tasks

import (
	"os/exec"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/dlvhdr/gh-dash/v4/internal/tui/context"
)

func TestReplyToReviewThread_TaskConfiguration(t *testing.T) {
	var capturedTask context.Task
	ctx := &context.ProgramContext{
		User: "octocat",
		StartTask: func(task context.Task) tea.Cmd {
			capturedTask = task
			return nil
		},
	}
	section := SectionIdentifier{Id: 2, Type: "pr"}
	pr := mockIssue{number: 42, repoName: "owner/repo"}

	cmd := ReplyToReviewThread(ctx, section, pr, "THREAD_ID", "sounds good")

	require.NotNil(t, cmd)
	require.Equal(t, "thread_reply_42", capturedTask.Id)
	require.Equal(t, "Replying to review thread", capturedTask.StartText)
	require.Equal(t, "Replied to review thread", capturedTask.FinishedText)
	require.Equal(t, context.TaskStart, capturedTask.State)
}

func TestReplyToReviewThread_MsgCallbackReturnsNewThreadReply(t *testing.T) {
	prNumber := 42
	threadId := "THREAD_ID"
	body := "sounds good"

	// Mirrors ReplyToReviewThread's GitHubTask, exercising the Msg callback
	// directly (matching TestCloseIssue_MsgCallbackReturnsCorrectUpdateIssueMsg's
	// pattern in issue_test.go).
	task := GitHubTask{
		Args: []string{
			"api",
			"graphql",
			"-f",
			"query=mutation($threadId:ID!,$body:String!){addPullRequestReviewThreadReply(input:{pullRequestReviewThreadId:$threadId,body:$body}){comment{id}}}",
			"-f",
			"threadId=" + threadId,
			"-f",
			"body=" + body,
		},
		Msg: func(c *exec.Cmd, err error) tea.Msg {
			return UpdatePRMsg{
				PrNumber: prNumber,
				NewThreadReply: &ThreadReply{
					ThreadId: threadId,
				},
			}
		},
	}

	require.Contains(t, task.Args, "threadId="+threadId)
	require.Contains(t, task.Args, "body="+body)

	msg := task.Msg(nil, nil)
	updateMsg, ok := msg.(UpdatePRMsg)

	require.True(t, ok, "Msg should return UpdatePRMsg")
	require.Equal(t, prNumber, updateMsg.PrNumber)
	require.NotNil(t, updateMsg.NewThreadReply)
	require.Equal(t, threadId, updateMsg.NewThreadReply.ThreadId)
	require.Nil(t, updateMsg.ResolvedThreadId, "a reply must not also resolve the thread")
}

func TestResolveReviewThread_TaskConfiguration(t *testing.T) {
	var capturedTask context.Task
	ctx := &context.ProgramContext{
		StartTask: func(task context.Task) tea.Cmd {
			capturedTask = task
			return nil
		},
	}
	section := SectionIdentifier{Id: 2, Type: "pr"}
	pr := mockIssue{number: 42, repoName: "owner/repo"}

	cmd := ResolveReviewThread(ctx, section, pr, "THREAD_ID")

	require.NotNil(t, cmd)
	require.Equal(t, "thread_resolve_42", capturedTask.Id)
	require.Equal(t, "Resolving review thread", capturedTask.StartText)
	require.Equal(t, "Review thread resolved", capturedTask.FinishedText)
	require.Equal(t, context.TaskStart, capturedTask.State)
}

func TestResolveReviewThread_MsgCallbackReturnsResolvedThreadId(t *testing.T) {
	prNumber := 42
	threadId := "THREAD_ID"

	task := GitHubTask{
		Args: []string{
			"api",
			"graphql",
			"-f",
			"query=mutation($threadId:ID!){resolveReviewThread(input:{threadId:$threadId}){thread{id}}}",
			"-f",
			"threadId=" + threadId,
		},
		Msg: func(c *exec.Cmd, err error) tea.Msg {
			return UpdatePRMsg{
				PrNumber:         prNumber,
				ResolvedThreadId: &threadId,
			}
		},
	}

	msg := task.Msg(nil, nil)
	updateMsg, ok := msg.(UpdatePRMsg)

	require.True(t, ok, "Msg should return UpdatePRMsg")
	require.Equal(t, prNumber, updateMsg.PrNumber)
	require.NotNil(t, updateMsg.ResolvedThreadId)
	require.Equal(t, threadId, *updateMsg.ResolvedThreadId)
	require.Nil(t, updateMsg.NewThreadReply, "resolving must not also post a reply")
}
