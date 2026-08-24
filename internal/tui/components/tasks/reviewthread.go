package tasks

import (
	"os/exec"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/dlvhdr/gh-dash/v4/internal/data"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/context"
)

func ReplyToReviewThread(
	ctx *context.ProgramContext,
	section SectionIdentifier,
	pr data.RowData,
	threadId string,
	body string,
) tea.Cmd {
	prNumber := pr.GetNumber()
	return fireTask(ctx, GitHubTask{
		Id: buildTaskId("thread_reply", prNumber),
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
		Section:      section,
		StartText:    "Replying to review thread",
		FinishedText: "Replied to review thread",
		Msg: func(c *exec.Cmd, err error) tea.Msg {
			return UpdatePRMsg{
				PrNumber: prNumber,
				NewThreadReply: &ThreadReply{
					ThreadId: threadId,
					Comment: data.ReviewComment{
						Author:    struct{ Login string }{Login: ctx.User},
						Body:      body,
						UpdatedAt: time.Now(),
					},
				},
			}
		},
	})
}

func ResolveReviewThread(
	ctx *context.ProgramContext,
	section SectionIdentifier,
	pr data.RowData,
	threadId string,
) tea.Cmd {
	prNumber := pr.GetNumber()
	return fireTask(ctx, GitHubTask{
		Id: buildTaskId("thread_resolve", prNumber),
		Args: []string{
			"api",
			"graphql",
			"-f",
			"query=mutation($threadId:ID!){resolveReviewThread(input:{threadId:$threadId}){thread{id}}}",
			"-f",
			"threadId=" + threadId,
		},
		Section:      section,
		StartText:    "Resolving review thread",
		FinishedText: "Review thread resolved",
		Msg: func(c *exec.Cmd, err error) tea.Msg {
			return UpdatePRMsg{
				PrNumber:         prNumber,
				ResolvedThreadId: &threadId,
			}
		},
	})
}
