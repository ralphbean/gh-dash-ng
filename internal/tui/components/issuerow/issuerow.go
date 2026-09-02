package issuerow

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/dlvhdr/gh-dash/v4/internal/config"
	"github.com/dlvhdr/gh-dash/v4/internal/data"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/table"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/constants"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/context"
	"github.com/dlvhdr/gh-dash/v4/internal/utils"
)

type Issue struct {
	Ctx            *context.ProgramContext
	Data           data.IssueData
	ShowAuthorIcon bool
}

func (issue *Issue) ToTableRow() table.Row {
	return table.Row{
		issue.renderFullsendStatus(),
		issue.renderNeedsAttention(),
		issue.renderStatus(),
		issue.renderRepoName(),
		issue.renderTitle(),
		issue.renderOpenedBy(),
		issue.renderAssignees(),
		issue.renderNumComments(),
		issue.renderNumReactions(),
		issue.renderUpdateAt(),
		issue.renderCreatedAt(),
	}
}

func (issue *Issue) renderNeedsAttention() string {
	user := issue.Ctx.User
	if user == "" {
		return ""
	}

	comments := issue.Data.Comments.Nodes
	if len(comments) == 0 {
		return ""
	}

	// Comments are fetched with "last: 15", so the final node is the
	// most recent comment. Show the indicator when someone other than
	// the current user authored it.
	lastComment := comments[len(comments)-1]
	if lastComment.Author.Login == user {
		return ""
	}
	return issue.getTextStyle().Render(constants.EyesIcon)
}

func (issue *Issue) getTextStyle() lipgloss.Style {
	return components.GetIssueTextStyle(issue.Ctx)
}

func (issue *Issue) renderUpdateAt() string {
	timeFormat := issue.Ctx.Config.Defaults.DateFormat

	updatedAtOutput := ""
	if timeFormat == "" || timeFormat == "relative" {
		updatedAtOutput = utils.TimeElapsed(issue.Data.UpdatedAt)
	} else {
		updatedAtOutput = issue.Data.UpdatedAt.Format(timeFormat)
	}

	return issue.getTextStyle().Render(updatedAtOutput)
}

func (issue *Issue) renderCreatedAt() string {
	timeFormat := issue.Ctx.Config.Defaults.DateFormat

	createdAtOutput := ""
	if timeFormat == "" || timeFormat == "relative" {
		createdAtOutput = utils.TimeElapsed(issue.Data.CreatedAt)
	} else {
		createdAtOutput = issue.Data.CreatedAt.Format(timeFormat)
	}

	return issue.getTextStyle().Render(createdAtOutput)
}

func (issue *Issue) renderRepoName() string {
	repoName := issue.Data.Repository.Name
	return issue.getTextStyle().Render(repoName)
}

func (issue *Issue) renderTitle() string {
	return components.RenderIssueTitle(
		issue.Ctx,
		issue.Data.State,
		issue.Data.Title,
		issue.Data.Number,
	)
}

func (issue *Issue) renderOpenedBy() string {
	return issue.getTextStyle().Render(issue.Data.GetAuthor(issue.Ctx.Theme, issue.ShowAuthorIcon))
}

func (issue *Issue) renderAssignees() string {
	assignees := make([]string, 0, len(issue.Data.Assignees.Nodes))
	for _, assignee := range issue.Data.Assignees.Nodes {
		assignees = append(assignees, assignee.Login)
	}
	return issue.getTextStyle().Render(strings.Join(assignees, ","))
}

func (issue *Issue) renderStatus() string {
	if issue.Data.State == "OPEN" {
		return lipgloss.NewStyle().Foreground(issue.Ctx.Styles.Colors.OpenIssue).Render("")
	} else {
		return issue.getTextStyle().Render("")
	}
}

func (issue *Issue) renderNumComments() string {
	return issue.getTextStyle().Render(fmt.Sprintf("%d", issue.Data.Comments.TotalCount))
}

func (issue *Issue) renderNumReactions() string {
	return issue.getTextStyle().Render(fmt.Sprintf("%d", issue.Data.Reactions.TotalCount))
}

// Unicode spinner characters for animation
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (issue *Issue) renderAgentIndicator(agent data.ActiveAgent) string {
	// Get spinner frame based on current time (time-based animation)
	// Update approximately every 100ms
	frameIndex := (time.Now().UnixMilli() / 100) % int64(len(spinnerFrames))
	spinner := spinnerFrames[frameIndex]

	// Get agent type abbreviation (first letter)
	abbreviation := ""
	if len(agent.Type) > 0 {
		abbreviation = string(agent.Type[0])
	}

	// Format: spinner + abbreviation (e.g., "⠋ C" for Code agent)
	return fmt.Sprintf("%s %s", spinner, abbreviation)
}

func (issue *Issue) renderFullsendStatus() string {
	// Check if fullsend integration is enabled
	if !config.IsFeatureEnabled(config.FF_FULLSEND_INTEGRATION) {
		return ""
	}

	// Get fullsend status from store
	owner, repoName := issue.Data.GetRepoNameAndOwner()
	fullsendStatus := data.GetFullsendStatusStore().Get(owner, repoName, issue.Data.GetNumber())

	// Check if there are active agents
	if len(fullsendStatus.ActiveAgents) == 0 {
		return ""
	}

	agents := fullsendStatus.ActiveAgents

	// Handle multiple concurrent agents
	if len(agents) > 2 {
		// Show spinner + count for >2 agents (e.g., "⠋ 9")
		frameIndex := (time.Now().UnixMilli() / 100) % int64(len(spinnerFrames))
		spinner := spinnerFrames[frameIndex]
		return issue.getTextStyle().Render(fmt.Sprintf("%s %d", spinner, len(agents)))
	}

	// Build indicator for 1-2 agents
	var indicators []string
	for _, agent := range agents {
		indicator := issue.renderAgentIndicator(agent)
		indicators = append(indicators, indicator)
	}

	return issue.getTextStyle().Render(strings.Join(indicators, " "))
}
