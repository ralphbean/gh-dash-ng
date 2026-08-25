package prrow

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"charm.land/log/v2"
	checks "github.com/dlvhdr/x/gh-checks"
	"github.com/dlvhdr/gh-dash/v4/internal/config"

	"github.com/dlvhdr/gh-dash/v4/internal/data"
	"github.com/dlvhdr/gh-dash/v4/internal/git"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/common"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/table"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/constants"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/context"
	"github.com/dlvhdr/gh-dash/v4/internal/utils"
)

type PullRequest struct {
	Ctx            *context.ProgramContext
	Data           *Data
	Branch         git.Branch
	Columns        []table.Column
	ShowAuthorIcon bool
}

func (pr *PullRequest) getTextStyle() lipgloss.Style {
	return components.GetIssueTextStyle(pr.Ctx)
}

func (pr *PullRequest) renderNumComments() string {
	if pr.Data.Primary == nil {
		return "-"
	}

	unresolvedThreads := pr.Data.Primary.UnresolvedThreadsCount()
	if unresolvedThreads == 0 {
		return ""
	}

	numCommentsStyle := pr.Ctx.Styles.Common.FaintTextStyle
	return numCommentsStyle.Render(fmt.Sprintf("%d", unresolvedThreads))
}

// renderReviewStatusFor renders the review-status icon for either the human
// or bot group of a PR's reviews. See data.ComputeReviewStatus for how a
// group's aggregate status is determined.
func (pr *PullRequest) renderReviewStatusFor(bot bool) string {
	if pr.Data.Primary == nil {
		return "-"
	}
	reviewCellStyle := pr.getTextStyle()

	human, botReviews := data.PartitionByBotAuthor(
		data.ReviewSummariesFromReviewsWithAuthorType(pr.Data.Primary.Reviews),
	)
	group := human
	if bot {
		group = botReviews
	}

	switch data.ComputeReviewStatus(group) {
	case "APPROVED":
		return reviewCellStyle.Foreground(pr.Ctx.Theme.SuccessText).Render(constants.ApprovedIcon)
	case "CHANGES_REQUESTED":
		return reviewCellStyle.Foreground(pr.Ctx.Theme.ErrorText).
			Render(constants.ChangesRequestedIcon)
	case "COMMENTED":
		return reviewCellStyle.Render(pr.Ctx.Styles.Common.CommentGlyph)
	default:
		return reviewCellStyle.Render(pr.Ctx.Styles.Common.WaitingGlyph)
	}
}

func (pr *PullRequest) renderReviewStatusHuman() string {
	return pr.renderReviewStatusFor(false)
}

func (pr *PullRequest) renderReviewStatusBot() string {
	return pr.renderReviewStatusFor(true)
}

func (pr *PullRequest) renderStar() string {
	if pr.Data == nil || pr.Data.Primary == nil {
		return ""
	}

	key := fmt.Sprintf("pr:%s#%d", pr.Data.GetRepoNameWithOwner(), pr.Data.GetNumber())
	if !data.GetStarStore().IsStarred(key) {
		return ""
	}

	return pr.getTextStyle().Foreground(pr.Ctx.Theme.WarningText).Render(constants.StarIcon)
}

func (pr *PullRequest) renderState() string {
	mergeCellStyle := lipgloss.NewStyle()

	if pr.Data.Primary == nil {
		return mergeCellStyle.Foreground(pr.Ctx.Theme.SuccessText).Render("󰜛")
	}

	switch pr.Data.Primary.State {
	case "OPEN":
		if pr.Data.Primary.IsInMergeQueue {
			return mergeCellStyle.Foreground(pr.Ctx.Theme.WarningText).
				Render(constants.MergeQueueIcon)
		}
		if pr.Data.Primary.IsDraft {
			return mergeCellStyle.Foreground(pr.Ctx.Theme.FaintText).Render(constants.DraftIcon)
		} else {
			return mergeCellStyle.Foreground(pr.Ctx.Styles.Colors.OpenPR).Render(constants.OpenIcon)
		}
	case "CLOSED":
		return mergeCellStyle.Foreground(pr.Ctx.Styles.Colors.ClosedPR).
			Render(constants.ClosedIcon)
	case "MERGED":
		return mergeCellStyle.Foreground(pr.Ctx.Styles.Colors.MergedPR).
			Render(constants.MergedIcon)
	default:
		return mergeCellStyle.Foreground(pr.Ctx.Theme.FaintText).Render("-")
	}
}

func (pr *PullRequest) GetStatusChecksRollup() checks.CommitState {
	if pr.Data == nil || pr.Data.Primary == nil {
		return checks.CommitStateUnknown
	}
	commits := pr.Data.Primary.Commits.Nodes
	if len(commits) == 0 {
		return checks.CommitStateUnknown
	}

	return checks.CommitState(commits[0].Commit.StatusCheckRollup.State)
}

func (pr *PullRequest) renderCiStatus() string {
	if pr.Data.Primary == nil {
		return "-"
	}

	accStatus := pr.GetStatusChecksRollup()
	ciCellStyle := pr.getTextStyle()

	switch accStatus {
	case checks.CommitStateSuccess:
		ciCellStyle = ciCellStyle.Foreground(pr.Ctx.Theme.SuccessText)
		return ciCellStyle.Render(constants.SuccessIcon)
	case checks.CommitStateExpected, checks.CommitStatePending:
		return ciCellStyle.Render(pr.Ctx.Styles.Common.WaitingGlyph)
	case checks.CommitStateError, checks.CommitStateFailure:
		ciCellStyle = ciCellStyle.Foreground(pr.Ctx.Theme.ErrorText)
		return ciCellStyle.Render(constants.FailureIcon)
	default:
		ciCellStyle = ciCellStyle.Foreground(pr.Ctx.Theme.FaintText)
		return ciCellStyle.Render(constants.EmptyIcon)
	}
}

func (pr *PullRequest) RenderLines(isSelected bool) string {
	if pr.Data.Primary == nil {
		return "-"
	}
	deletions := max(pr.Data.Primary.Deletions, 0)

	var additionsFg, deletionsFg compat.AdaptiveColor
	additionsFg = pr.Ctx.Theme.SuccessText
	deletionsFg = pr.Ctx.Theme.ErrorText

	baseStyle := lipgloss.NewStyle()
	if isSelected {
		baseStyle = baseStyle.Background(pr.Ctx.Theme.SelectedBackground)
	}

	additionsText := baseStyle.
		Foreground(additionsFg).
		Render(fmt.Sprintf("+%s", components.FormatNumber(pr.Data.Primary.Additions)))
	deletionsText := baseStyle.
		Foreground(deletionsFg).
		Render(fmt.Sprintf("-%s", components.FormatNumber(deletions)))

	return pr.getTextStyle().Render(
		keepSameSpacesOnAddDeletions(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				additionsText,
				baseStyle.Render(" "),
				deletionsText,
			)),
	)
}

func keepSameSpacesOnAddDeletions(str string) string {
	strAsList := strings.Split(str, " ")
	return fmt.Sprintf(
		"%7s",
		strAsList[0],
	) + " " + fmt.Sprintf(
		"%7s",
		strAsList[1],
	)
}

func (pr *PullRequest) renderTitle() string {
	return components.RenderIssueTitle(
		pr.Ctx,
		pr.Data.Primary.State,
		pr.Data.Primary.Title,
		pr.Data.Primary.Number,
	)
}

func (pr *PullRequest) renderExtendedTitle(isSelected bool) string {
	baseStyle := lipgloss.NewStyle()
	if isSelected {
		baseStyle = baseStyle.Foreground(pr.Ctx.Theme.SecondaryText).
			Background(pr.Ctx.Theme.SelectedBackground)
	}

	author := baseStyle.Bold(true).Render(fmt.Sprintf("@%s",
		pr.Data.Primary.GetAuthor(pr.Ctx.Theme, pr.ShowAuthorIcon)))
	top := lipgloss.JoinHorizontal(lipgloss.Top, pr.Data.Primary.Repository.NameWithOwner,
		fmt.Sprintf(" #%d by %s", pr.Data.Primary.Number, author))
	branchHidden := pr.Ctx.Config.Defaults.Layout.Prs.Base.Hidden
	if branchHidden == nil || !*branchHidden {
		branch := baseStyle.Render(pr.Data.Primary.HeadRefName)
		top = lipgloss.JoinHorizontal(lipgloss.Top, top, baseStyle.Render(" · "), branch)
	}
	title := pr.Data.Primary.Title
	var titleColumn table.Column
	for _, column := range pr.Columns {
		if column.Grow != nil && *column.Grow {
			titleColumn = column
		}
	}
	width := titleColumn.ComputedWidth - 2
	top = baseStyle.Foreground(pr.Ctx.Theme.SecondaryText).
		Width(width).
		MaxWidth(width).
		Height(1).
		MaxHeight(1).
		Render(top)
	title = baseStyle.Foreground(pr.Ctx.Theme.PrimaryText).Bold(true).Width(width).MaxWidth(
		width).Height(1).MaxHeight(1).Render(title)

	return baseStyle.Render(lipgloss.JoinVertical(lipgloss.Left, top, title))
}

func (pr *PullRequest) renderAuthor() string {
	return pr.getTextStyle().Render(pr.Data.Primary.GetAuthor(pr.Ctx.Theme, pr.ShowAuthorIcon))
}

func (pr *PullRequest) renderAssignees() string {
	if pr.Data.Primary == nil {
		return ""
	}
	assignees := make([]string, 0, len(pr.Data.Primary.Assignees.Nodes))
	for _, assignee := range pr.Data.Primary.Assignees.Nodes {
		assignees = append(assignees, assignee.Login)
	}
	return pr.getTextStyle().Render(strings.Join(assignees, ","))
}

func (pr *PullRequest) renderLabels(isSelected bool) string {
	if pr.Data == nil || pr.Data.Primary == nil || len(pr.Data.Primary.Labels.Nodes) == 0 {
		return ""
	}

	labelsWidth := 0
	for _, column := range pr.Columns {
		if column.Title != constants.LabelsIcon {
			continue
		}

		labelsWidth = column.ComputedWidth
		if labelsWidth == 0 && column.Width != nil {
			labelsWidth = *column.Width
		}
		break
	}

	if labelsWidth <= 2 {
		return ""
	}

	maxRows := 2
	if pr.Ctx != nil &&
		pr.Ctx.Config != nil &&
		pr.Ctx.Config.Theme != nil &&
		pr.Ctx.Config.Theme.Ui.Table.Compact {
		maxRows = 1
	}

	pillStyle := pr.getTextStyle()
	if pr.Ctx != nil {
		pillStyle = pr.Ctx.Styles.PrView.PillStyle
	}

	rowStyle := lipgloss.NewStyle()
	if isSelected && pr.Ctx != nil {
		rowStyle = rowStyle.Background(pr.Ctx.Theme.SelectedBackground)
		pillStyle = pillStyle.
			BorderLeftBackground(pr.Ctx.Theme.SelectedBackground).
			BorderRightBackground(pr.Ctx.Theme.SelectedBackground)
	}

	return common.RenderLabels(
		pr.Data.Primary.Labels.Nodes,
		common.LabelOpts{
			Width:     labelsWidth - 2,
			MaxRows:   maxRows,
			PillStyle: pillStyle,
			RowStyle:  rowStyle,
		},
	)
}

func (pr *PullRequest) renderRepoName() string {
	repoName := ""
	if !pr.Ctx.Config.Theme.Ui.Table.Compact {
		repoName = pr.Data.Primary.Repository.NameWithOwner
	} else {
		repoName = pr.Data.Primary.HeadRepository.Name
	}
	return pr.getTextStyle().Foreground(pr.Ctx.Theme.FaintText).Render(repoName)
}

func (pr *PullRequest) renderUpdateAt() string {
	timeFormat := pr.Ctx.Config.Defaults.DateFormat

	updatedAtOutput := ""
	t := pr.Branch.LastUpdatedAt
	if pr.Data.Primary != nil {
		t = &pr.Data.Primary.UpdatedAt
	}

	if t == nil {
		return ""
	}

	if timeFormat == "" || timeFormat == "relative" {
		updatedAtOutput = utils.TimeElapsed(*t)
	} else {
		updatedAtOutput = t.Format(timeFormat)
	}

	return pr.getTextStyle().Foreground(pr.Ctx.Theme.FaintText).Render(updatedAtOutput)
}

func (pr *PullRequest) renderCreatedAt() string {
	timeFormat := pr.Ctx.Config.Defaults.DateFormat

	createdAtOutput := ""
	t := pr.Branch.CreatedAt
	if pr.Data.Primary != nil {
		t = &pr.Data.Primary.CreatedAt
	}

	if t == nil {
		return ""
	}

	if timeFormat == "" || timeFormat == "relative" {
		createdAtOutput = utils.TimeElapsed(*t)
	} else {
		createdAtOutput = t.Format(timeFormat)
	}

	return pr.getTextStyle().Foreground(pr.Ctx.Theme.FaintText).Render(createdAtOutput)
}

func (pr *PullRequest) renderBaseName() string {
	if pr.Data.Primary == nil {
		return ""
	}
	return pr.getTextStyle().Render(pr.Data.Primary.BaseRefName)
}

func (pr *PullRequest) RenderState() string {
	switch pr.Data.Primary.State {
	case "OPEN":
		if pr.Data.Primary.IsInMergeQueue {
			return constants.MergeQueueIcon + " Queued"
		}
		if pr.Data.Primary.IsDraft {
			return constants.DraftIcon + " Draft"
		} else {
			return constants.OpenIcon + " Open"
		}
	case "CLOSED":
		return constants.ClosedIcon + " Closed"
	case "MERGED":
		return constants.MergedIcon + " Merged"
	default:
		return ""
	}
}

func (pr *PullRequest) RenderMergeStateStatus() string {
	if pr.Data.Primary.Mergeable == "CONFLICTING" {
		return constants.FailureIcon + " Conflicting"
	}
	switch pr.Data.Primary.MergeStateStatus {
	case "CLEAN":
		return constants.SuccessIcon + " Up-to-date"
	case "BLOCKED":
		return constants.BlockedIcon + " Blocked"
	case "BEHIND":
		return constants.BehindIcon + " Behind"
	default:
		return ""
	}
}

func (pr *PullRequest) renderMergeStatus() string {
	if pr.Data.Primary == nil {
		return ""
	}
	mergeStyle := pr.getTextStyle()

	if pr.Data.Primary.Mergeable == "CONFLICTING" {
		return mergeStyle.Foreground(pr.Ctx.Theme.ErrorText).
			Render(constants.FailureIcon)
	}

	switch pr.Data.Primary.MergeStateStatus {
	case "CLEAN":
		return mergeStyle.Foreground(pr.Ctx.Theme.SuccessText).
			Render(constants.SuccessIcon)
	case "BLOCKED":
		return mergeStyle.Foreground(pr.Ctx.Theme.ErrorText).
			Render(constants.BlockedIcon)
	case "BEHIND":
		return mergeStyle.Foreground(pr.Ctx.Theme.WarningText).
			Render(constants.BehindIcon)
	default:
		return ""
	}
}

func (pr *PullRequest) ToTableRow(isSelected bool) table.Row {
	if !pr.Ctx.Config.Theme.Ui.Table.Compact {
		return table.Row{
			pr.renderStar(),
			pr.renderState(),
			pr.renderExtendedTitle(isSelected),
			pr.renderLabels(isSelected),
			pr.renderAssignees(),
			pr.renderBaseName(),
			pr.renderNumComments(),
			pr.renderReviewStatusHuman(),
			pr.renderReviewStatusBot(),
			pr.renderCiStatus(),
			pr.renderMergeStatus(),
			pr.RenderLines(isSelected),
			pr.renderUpdateAt(),
			pr.renderCreatedAt(),
			pr.renderFullsendStatus(),
		}
	}

	return table.Row{
		pr.renderState(),
		pr.renderRepoName(),
		pr.renderTitle(),
		pr.renderAuthor(),
		pr.renderLabels(isSelected),
		pr.renderAssignees(),
		pr.renderBaseName(),
		pr.renderNumComments(),
		pr.renderReviewStatusHuman(),
		pr.renderReviewStatusBot(),
		pr.renderCiStatus(),
		pr.renderMergeStatus(),
		pr.RenderLines(isSelected),
		pr.renderUpdateAt(),
		pr.renderCreatedAt(),
		pr.renderFullsendStatus(),
	}
}

// Unicode spinner characters for animation (same as issuerow)
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (pr *PullRequest) renderAgentIndicator(agent data.ActiveAgent) string {
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

func (pr *PullRequest) renderFullsendStatus() string {
	// Check if fullsend integration is enabled
	if !config.IsFeatureEnabled(config.FF_FULLSEND_INTEGRATION) {
		return ""
	}

	// Check if we have PR data
	if pr.Data == nil || pr.Data.Primary == nil {
		return ""
	}

	// Get fullsend status from store
	owner, repoName := pr.Data.Primary.GetRepoNameAndOwner()
	fullsendStatus := data.GetFullsendStatusStore().Get(owner, repoName, pr.Data.Primary.GetNumber())

	log.Debug("renderFullsendStatus",
		"owner", owner,
		"repo", repoName,
		"number", pr.Data.Primary.GetNumber(),
		"active_agents", len(fullsendStatus.ActiveAgents))

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
		return pr.getTextStyle().Render(fmt.Sprintf("%s %d", spinner, len(agents)))
	}

	// Build indicator for 1-2 agents
	var indicators []string
	for _, agent := range agents {
		indicator := pr.renderAgentIndicator(agent)
		indicators = append(indicators, indicator)
	}

	return pr.getTextStyle().Render(strings.Join(indicators, " "))
}
