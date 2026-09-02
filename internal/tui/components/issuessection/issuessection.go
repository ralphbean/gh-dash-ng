package issuessection

import (
	"fmt"
	"slices"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/dlvhdr/gh-dash/v4/internal/config"
	"github.com/dlvhdr/gh-dash/v4/internal/data"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/fullsendmonitor"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/issuerow"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/section"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/table"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/tasks"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/constants"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/context"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/keys"
	"github.com/dlvhdr/gh-dash/v4/internal/utils"
)

const SectionType = "issue"

type Model struct {
	section.BaseModel
	Issues          []data.IssueData
	projectedIssues []data.IssueData
}

func NewModel(
	id int,
	ctx *context.ProgramContext,
	cfg config.IssuesSectionConfig,
	lastUpdated time.Time,
	createdAt time.Time,
) Model {
	m := Model{}
	m.BaseModel = section.NewModel(
		ctx,
		section.NewSectionOptions{
			Id:          id,
			Config:      cfg.ToSectionConfig(),
			Type:        SectionType,
			Columns:     GetSectionColumns(cfg, ctx),
			Singular:    m.GetItemSingularForm(),
			Plural:      m.GetItemPluralForm(),
			LastUpdated: lastUpdated,
			CreatedAt:   createdAt,
		},
	)
	m.Issues = []data.IssueData{}

	return m
}

func (m *Model) Update(msg tea.Msg) (section.Section, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:

		if m.IsSearchFocused() {
			switch msg.String() {
			case "ctrl+c", "esc":
				m.SearchBar.SetValue(m.SearchValue)
				blinkCmd := m.SetIsSearching(false)
				return m, blinkCmd

			case "enter":
				m.SearchValue = m.SearchBar.Value()
				m.SyncSmartFilterWithSearchValue()
				m.SetIsSearching(false)
				m.ResetRows()
				return m, tea.Batch(m.FetchNextPageSectionRows()...)
			}

			break
		}

		if m.IsPromptConfirmationFocused() {
			switch msg.String() {
			case "ctrl+c", "esc":
				m.PromptConfirmationBox.Reset()
				cmd = m.SetIsPromptConfirmationShown(false)
				return m, cmd

			case "enter":
				input := m.PromptConfirmationBox.Value()
				action := m.GetPromptConfirmationAction()
				issue := m.GetCurrRow()
				sid := tasks.SectionIdentifier{Id: m.Id, Type: SectionType}
				if action == "snooze" {
					if m.applySnooze(input, issue) {
						cmd = tasks.SnoozeFeedback(
							m.Ctx,
							sid,
							snoozeKey(issue),
							fmt.Sprintf("Issue #%d", issue.GetNumber()),
						)
					}
					m.Table.SetRows(m.BuildRows())
				} else if input == "Y" || input == "y" {
					switch action {
					case "close":
						cmd = tasks.CloseIssue(m.Ctx, sid, issue)
					case "reopen":
						cmd = tasks.ReopenIssue(m.Ctx, sid, issue)
					}
				}

				m.PromptConfirmationBox.Reset()
				blinkCmd := m.SetIsPromptConfirmationShown(false)

				return m, tea.Batch(cmd, blinkCmd)
			}
			break
		}

		switch {
		case key.Matches(msg, keys.IssueKeys.ToggleSmartFiltering):
			if m.HasCurrentRepoNameInConfiguredFilter() || !m.HasRepoNameInConfiguredFilter() {
				m.IsFilteredByCurrentRemote = !m.IsFilteredByCurrentRemote
			}
			searchValue := m.GetSearchValue()
			if m.SearchValue != searchValue {
				m.SearchValue = searchValue
				m.SearchBar.SetValue(searchValue)
				m.SetIsSearching(false)
				m.ResetRows()
				return m, tea.Batch(m.FetchNextPageSectionRows()...)
			}
		}

	case fullsendmonitor.FullsendStatusUpdatedMsg:
		m.rebuildRowsPreservingSelection()

	case tasks.UpdateIssueMsg:
		for i, currIssue := range m.Issues {
			if currIssue.Number == msg.IssueNumber {
				if msg.IsClosed != nil {
					if *msg.IsClosed {
						currIssue.State = "CLOSED"
					} else {
						currIssue.State = "OPEN"
					}
				}
				if msg.Labels != nil {
					currIssue.Labels.Nodes = msg.Labels.Nodes
				}
				if msg.NewComment != nil {
					currIssue.Comments.Nodes = append(currIssue.Comments.Nodes, *msg.NewComment)
				}
				if msg.AddedAssignees != nil {
					currIssue.Assignees.Nodes = addAssignees(
						currIssue.Assignees.Nodes, msg.AddedAssignees.Nodes)
				}
				if msg.RemovedAssignees != nil {
					currIssue.Assignees.Nodes = removeAssignees(
						currIssue.Assignees.Nodes, msg.RemovedAssignees.Nodes)
				}
				m.Issues[i] = currIssue
				m.SetIsLoading(false)
				m.Table.SetRows(m.BuildRows())
				break
			}
		}

	case SectionIssuesFetchedMsg:
		if m.LastFetchTaskId == msg.TaskId {
			if m.PageInfo != nil {
				m.Issues = append(m.Issues, msg.Issues...)
			} else {
				m.Issues = msg.Issues
			}
			m.TotalCount = msg.TotalCount
			m.SetIsLoading(false)
			m.PageInfo = &msg.PageInfo
			m.Table.SetRows(m.BuildRows())
			m.UpdateLastUpdated(time.Now())
			m.UpdateTotalItemsCount(m.TotalCount)
		}
	}

	search, searchCmd := m.SearchBar.Update(msg)
	m.SearchBar = search

	prompt, promptCmd := m.PromptConfirmationBox.Update(msg)
	m.PromptConfirmationBox = prompt

	table, tableCmd := m.Table.Update(msg)
	m.Table = table

	return m, tea.Batch(cmd, searchCmd, promptCmd, tableCmd)
}

func GetSectionColumns(
	cfg config.IssuesSectionConfig,
	ctx *context.ProgramContext,
) []table.Column {
	dLayout := ctx.Config.Defaults.Layout.Issues
	sLayout := cfg.Layout

	updatedAtLayout := config.MergeColumnConfigs(
		dLayout.UpdatedAt,
		sLayout.UpdatedAt,
	)
	createdAtLayout := config.MergeColumnConfigs(
		dLayout.CreatedAt,
		sLayout.CreatedAt,
	)
	stateLayout := config.MergeColumnConfigs(dLayout.State, sLayout.State)
	repoLayout := config.MergeColumnConfigs(dLayout.Repo, sLayout.Repo)
	titleLayout := config.MergeColumnConfigs(dLayout.Title, sLayout.Title)
	creatorLayout := config.MergeColumnConfigs(dLayout.Creator, sLayout.Creator)
	assigneesLayout := config.MergeColumnConfigs(
		dLayout.Assignees,
		sLayout.Assignees,
	)
	commentsLayout := config.MergeColumnConfigs(
		dLayout.Comments,
		sLayout.Comments,
	)
	reactionsLayout := config.MergeColumnConfigs(
		dLayout.Reactions,
		sLayout.Reactions,
	)
	needsAttentionLayout := config.MergeColumnConfigs(
		dLayout.NeedsAttention,
		sLayout.NeedsAttention,
	)

	return []table.Column{
		{
			Title:  "🤖",
			Width:  utils.IntPtr(6),
			Hidden: utils.BoolPtr(!config.IsFeatureEnabled(config.FF_FULLSEND_INTEGRATION)),
		},
		{
			Title:  "",
			Width:  utils.IntPtr(4),
			Hidden: needsAttentionLayout.Hidden,
		},
		{
			Title:  "",
			Width:  stateLayout.Width,
			Hidden: stateLayout.Hidden,
		},
		{
			Title:  "",
			Width:  repoLayout.Width,
			Hidden: repoLayout.Hidden,
		},
		{
			Title:  "Title",
			Grow:   utils.BoolPtr(true),
			Hidden: titleLayout.Hidden,
		},
		{
			Title:  "Creator",
			Width:  creatorLayout.Width,
			Hidden: creatorLayout.Hidden,
		},
		{
			Title:  "Assignees",
			Width:  assigneesLayout.Width,
			Hidden: assigneesLayout.Hidden,
		},
		{
			Title:  constants.CommentsIcon,
			Width:  &issueNumCommentsCellWidth,
			Hidden: commentsLayout.Hidden,
		},
		{
			Title:  "",
			Width:  &issueNumCommentsCellWidth,
			Hidden: reactionsLayout.Hidden,
		},
		{
			Title:  "󱦻",
			Width:  updatedAtLayout.Width,
			Hidden: updatedAtLayout.Hidden,
		},
		{
			Title:  "󱡢",
			Width:  createdAtLayout.Width,
			Hidden: createdAtLayout.Hidden,
		},
	}
}

// visibleIssues returns m.Issues with snoozed issues filtered out. It's used
// by both BuildRows and GetCurrRow so that the table's row index (which only
// ever spans visible rows) maps consistently to the underlying data.
func (m Model) visibleIssues() []data.IssueData {
	snoozeStore := data.GetSnoozeStore()
	visible := make([]data.IssueData, 0, len(m.Issues))
	for _, issue := range m.Issues {
		if snoozeStore.IsSnoozed(snoozeKey(&issue)) {
			continue
		}
		visible = append(visible, issue)
	}
	return visible
}

func issueHasActiveAgent(issue *data.IssueData) bool {
	owner, repo := issue.GetRepoNameAndOwner()
	status := data.GetFullsendStatusStore().Get(owner, repo, issue.GetNumber())
	for _, agent := range status.ActiveAgents {
		if agent.Status == "queued" || agent.Status == "in_progress" {
			return true
		}
	}
	return false
}

func (m Model) orderedIssues() ([]data.IssueData, map[int]string) {
	visible := m.visibleIssues()
	if !config.IsFeatureEnabled(config.FF_FULLSEND_INTEGRATION) || len(visible) == 0 {
		return visible, nil
	}
	idle := make([]data.IssueData, 0, len(visible))
	active := make([]data.IssueData, 0, len(visible))
	for _, issue := range visible {
		if issueHasActiveAgent(&issue) {
			active = append(active, issue)
		} else {
			idle = append(idle, issue)
		}
	}
	ordered := append(idle, active...)
	headers := make(map[int]string, 2)
	if len(idle) > 0 {
		headers[0] = "○ No active agent"
	}
	if len(active) > 0 {
		headers[len(idle)] = "● Active agent"
	}
	return ordered, headers
}

func (m *Model) BuildRows() []table.Row {
	var rows []table.Row
	ordered, headers := m.orderedIssues()
	m.projectedIssues = ordered
	m.Table.GroupHeaders = headers
	for _, currIssue := range ordered {
		issueModel := issuerow.Issue{Ctx: m.Ctx, Data: currIssue, ShowAuthorIcon: m.ShowAuthorIcon}
		rows = append(rows, issueModel.ToTableRow())
	}

	if rows == nil {
		rows = []table.Row{}
	}

	return rows
}

func (m *Model) NumRows() int {
	ordered, _ := m.orderedIssues()
	return len(ordered)
}

func (m *Model) GetCurrRow() data.RowData {
	idx := m.Table.GetCurrItem()
	visible := m.projectedIssues
	if len(m.Table.Rows) == 0 || len(visible) != len(m.Table.Rows) {
		visible, _ = m.orderedIssues()
	}
	if idx < 0 || idx >= len(visible) {
		return nil
	}
	issue := visible[idx]
	return &issue
}

func issueIdentity(issue *data.IssueData) string {
	if issue == nil {
		return ""
	}
	owner, repo := issue.GetRepoNameAndOwner()
	return fmt.Sprintf("issue:%s/%s#%d", owner, repo, issue.GetNumber())
}

func (m *Model) rebuildRowsPreservingSelection() {
	selected, _ := m.GetCurrRow().(*data.IssueData)
	identity := issueIdentity(selected)
	m.Table.SetRows(m.BuildRows())
	if identity == "" {
		return
	}
	ordered, _ := m.orderedIssues()
	for i := range ordered {
		if issueIdentity(&ordered[i]) == identity {
			m.Table.SetCurrItem(i)
			m.Table.SyncViewPortContent()
			return
		}
	}
}

func (m *Model) FetchNextPageSectionRows() []tea.Cmd {
	if m == nil {
		return nil
	}

	if m.PageInfo != nil && !m.PageInfo.HasNextPage {
		return nil
	}

	var cmds []tea.Cmd

	startCursor := time.Now().String()
	if m.PageInfo != nil {
		startCursor = m.PageInfo.StartCursor
	}
	taskId := fmt.Sprintf("fetching_issues_%d_%s", m.Id, startCursor)
	m.LastFetchTaskId = taskId
	task := context.Task{
		Id:        taskId,
		StartText: fmt.Sprintf(`Fetching issues for "%s"`, m.Config.Title),
		FinishedText: fmt.Sprintf(
			`Issues for "%s" have been fetched`,
			m.Config.Title,
		),
		State: context.TaskStart,
		Error: nil,
	}
	startCmd := m.Ctx.StartTask(task)
	cmds = append(cmds, startCmd)

	fetchCmd := func() tea.Msg {
		limit := m.Config.Limit
		if limit == nil {
			limit = &m.Ctx.Config.Defaults.IssuesLimit
		}
		res, err := data.FetchIssues(m.GetFilters(), *limit, m.PageInfo)
		if err != nil {
			return constants.TaskFinishedMsg{
				SectionId:   m.Id,
				SectionType: m.Type,
				TaskId:      taskId,
				Err:         err,
			}
		}

		return constants.TaskFinishedMsg{
			SectionId:   m.Id,
			SectionType: m.Type,
			TaskId:      taskId,
			Msg: SectionIssuesFetchedMsg{
				Issues:     res.Issues,
				TotalCount: res.TotalCount,
				PageInfo:   res.PageInfo,
				TaskId:     taskId,
			},
		}
	}
	cmds = append(cmds, fetchCmd)

	return cmds
}

func (m *Model) UpdateLastUpdated(t time.Time) {
	m.Table.UpdateLastUpdated(t)
}

func (m *Model) ResetRows() {
	m.Issues = nil
	m.BaseModel.ResetRows()
}

func FetchAllSections(
	ctx *context.ProgramContext,
) (sections []section.Section, fetchAllCmd tea.Cmd) {
	sectionConfigs := ctx.Config.IssuesSections
	fetchIssuesCmds := make([]tea.Cmd, 0, len(sectionConfigs))
	sections = make([]section.Section, 0, len(sectionConfigs))
	for i, sectionConfig := range sectionConfigs {
		sectionModel := NewModel(
			i+1,
			ctx,
			sectionConfig,
			time.Now(),
			time.Now(),
		) // 0 is the search section
		if sectionConfig.Layout.CreatorIcon.Hidden != nil {
			sectionModel.ShowAuthorIcon = !*sectionConfig.Layout.CreatorIcon.Hidden
		}
		sections = append(sections, &sectionModel)
		fetchIssuesCmds = append(
			fetchIssuesCmds,
			sectionModel.FetchNextPageSectionRows()...)
	}
	return sections, tea.Batch(fetchIssuesCmds...)
}

type SectionIssuesFetchedMsg struct {
	Issues     []data.IssueData
	TotalCount int
	PageInfo   data.PageInfo
	TaskId     string
}

func addAssignees(assignees, addedAssignees []data.Assignee) []data.Assignee {
	newAssignees := assignees
	for _, assignee := range addedAssignees {
		if !assigneesContains(newAssignees, assignee) {
			newAssignees = append(newAssignees, assignee)
		}
	}

	return newAssignees
}

func removeAssignees(
	assignees, removedAssignees []data.Assignee,
) []data.Assignee {
	newAssignees := []data.Assignee{}
	for _, assignee := range assignees {
		if !assigneesContains(removedAssignees, assignee) {
			newAssignees = append(newAssignees, assignee)
		}
	}

	return newAssignees
}

func assigneesContains(assignees []data.Assignee, assignee data.Assignee) bool {
	return slices.Contains(assignees, assignee)
}

func (m Model) GetItemSingularForm() string {
	return "Issue"
}

func (m Model) GetItemPluralForm() string {
	return "Issues"
}

// visibleTotalCount adjusts TotalCount by the number of currently-snoozed
// issues among the fetched rows, so the tab badge and pager reflect what's
// actually visible. It self-corrects as snoozes expire since
// visibleIssues() re-evaluates time.Now() on every call.
func (m Model) visibleTotalCount() int {
	numSnoozed := len(m.Issues) - len(m.visibleIssues())
	return utils.Max(0, m.TotalCount-numSnoozed)
}

func (m Model) GetTotalCount() int {
	return m.visibleTotalCount()
}

func (m *Model) GetIsLoading() bool {
	return m.IsLoading
}

func (m *Model) SetIsLoading(val bool) {
	m.IsLoading = val
	m.Table.SetIsLoading(val)
}

func (m Model) GetPagerContent() string {
	pagerContent := ""
	totalCount := m.visibleTotalCount()
	if totalCount > 0 {
		pagerContent = fmt.Sprintf(
			"%v %v • %v %v/%v • Fetched %v",
			constants.WaitingIcon,
			m.LastUpdated().Format("01/02 15:04:05"),
			m.SingularForm,
			m.Table.GetCurrItem()+1,
			totalCount,
			len(m.Table.Rows),
		)
	}
	pager := m.Ctx.Styles.ListViewPort.PagerStyle.Render(pagerContent)
	return pager
}
