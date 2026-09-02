package prssection

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/dlvhdr/gh-dash/v4/internal/config"
	"github.com/dlvhdr/gh-dash/v4/internal/data"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/fullsendmonitor"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/prompt"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/prrow"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/search"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/section"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/tasks"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/constants"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/context"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/keys"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/theme"
)

// newTestModel creates a minimal Model with the prompt confirmation box
// focused and a single PR row so that GetCurrRow returns non-nil.
func newTestModel(action string) Model {
	cfg, err := config.ParseConfig(config.Location{
		ConfigFlag:       "../../../config/testdata/test-config.yml",
		SkipGlobalConfig: true,
	})
	if err != nil {
		panic(err)
	}
	thm := theme.ParseTheme(&cfg)
	ctx := &context.ProgramContext{
		Config: &cfg,
		Theme:  thm,
		Styles: context.InitStyles(thm),
		StartTask: func(task context.Task) tea.Cmd {
			return func() tea.Msg { return nil }
		},
	}
	m := Model{
		BaseModel: section.BaseModel{
			Ctx:                       ctx,
			IsPromptConfirmationShown: true,
			PromptConfirmationAction:  action,
			PromptConfirmationBox:     prompt.NewModel(ctx),
			SearchBar:                 search.NewModel(ctx, search.SearchOptions{}),
		},
		Prs: []prrow.Data{
			{Primary: &data.PullRequestData{Number: 42}},
		},
	}
	m.PromptConfirmationBox.Focus()
	m.Table.UpdateProgramContext(ctx)
	return m
}

func testPR(number int) prrow.Data {
	return prrow.Data{Primary: &data.PullRequestData{
		Number: number,
		Repository: data.Repository{
			Name:          "repo",
			NameWithOwner: "owner/repo",
			Owner:         data.Owner{Login: "owner"},
		},
	}}
}

func enableFullsend(t *testing.T) {
	t.Helper()
	t.Setenv(config.FF_FULLSEND_INTEGRATION, "1")
	data.GetFullsendStatusStore().Clear()
	t.Cleanup(data.GetFullsendStatusStore().Clear)
}

func TestOrderedPRsGroupsActiveAgentsStably(t *testing.T) {
	enableFullsend(t)
	m := newTestModel("")
	m.Prs = []prrow.Data{testPR(1), testPR(2), testPR(3), testPR(4)}
	data.GetFullsendStatusStore().Set("owner", "repo", 2, data.FullsendStatus{
		ActiveAgents: []data.ActiveAgent{{Status: "in_progress"}},
	})
	data.GetFullsendStatusStore().Set("owner", "repo", 4, data.FullsendStatus{
		ActiveAgents: []data.ActiveAgent{{Status: "queued"}},
	})

	ordered, headers := m.orderedPRs()
	require.Equal(t, []int{1, 3, 2, 4}, []int{
		ordered[0].GetNumber(),
		ordered[1].GetNumber(),
		ordered[2].GetNumber(),
		ordered[3].GetNumber(),
	})
	require.Equal(t, "○ No active agent", headers[0])
	require.Equal(t, "● Active agent", headers[2])
}

func TestOrderedPRsDisabledKeepsOriginalOrder(t *testing.T) {
	data.GetFullsendStatusStore().Clear()
	m := newTestModel("")
	m.Prs = []prrow.Data{testPR(2), testPR(1)}
	data.GetFullsendStatusStore().Set("owner", "repo", 2, data.FullsendStatus{
		ActiveAgents: []data.ActiveAgent{{Status: "in_progress"}},
	})

	ordered, headers := m.orderedPRs()
	require.Equal(t, []int{2, 1}, []int{ordered[0].GetNumber(), ordered[1].GetNumber()})
	require.Nil(t, headers)
}

func TestFullsendUpdatePreservesSelectedPRIdentity(t *testing.T) {
	enableFullsend(t)
	m := newTestModel("")
	m.IsPromptConfirmationShown = false
	m.Prs = []prrow.Data{testPR(1), testPR(2), testPR(3)}
	m.Table.SetRows(m.BuildRows())
	m.Table.SetCurrItem(1)
	require.Equal(t, 2, m.GetCurrRow().GetNumber())
	data.GetFullsendStatusStore().Set("owner", "repo", 2, data.FullsendStatus{
		ActiveAgents: []data.ActiveAgent{{Status: "in_progress"}},
	})

	_, _ = m.Update(fullsendmonitor.FullsendStatusUpdatedMsg{Repo: "owner/repo", Number: 2})

	require.Equal(t, 2, m.GetCurrRow().GetNumber())
	require.Equal(t, 2, m.Table.GetCurrItem())
}

func TestFullsendUpdatePreservesSelectionAcrossMultipleGroupChanges(t *testing.T) {
	enableFullsend(t)
	m := newTestModel("")
	m.IsPromptConfirmationShown = false
	m.Prs = []prrow.Data{testPR(1), testPR(2), testPR(3)}
	data.GetFullsendStatusStore().Set("owner", "repo", 1, data.FullsendStatus{
		ActiveAgents: []data.ActiveAgent{{Status: "in_progress"}},
	})
	m.Table.SetRows(m.BuildRows())
	m.Table.SetCurrItem(2)
	require.Equal(t, 1, m.GetCurrRow().GetNumber())

	data.GetFullsendStatusStore().Set("owner", "repo", 1, data.FullsendStatus{})
	data.GetFullsendStatusStore().Set("owner", "repo", 2, data.FullsendStatus{
		ActiveAgents: []data.ActiveAgent{{Status: "queued"}},
	})
	_, _ = m.Update(fullsendmonitor.FullsendStatusUpdatedMsg{Repo: "owner/repo", Number: 2})

	require.Equal(t, 1, m.GetCurrRow().GetNumber())
	require.Equal(t, 0, m.Table.GetCurrItem())
}

func TestPRFullsendColumnIsLeftmost(t *testing.T) {
	enableFullsend(t)
	m := newTestModel("")
	cols := GetSectionColumns(config.PrsSectionConfig{}, m.Ctx)
	require.Equal(t, "🤖", cols[0].Title)
}

func TestActiveGroupRowActionsTargetSelectedPR(t *testing.T) {
	enableFullsend(t)
	withTestStarStore(t)
	m := newTestModel("")
	m.IsPromptConfirmationShown = false
	m.Prs = []prrow.Data{testPR(1), testPR(2)}
	data.GetFullsendStatusStore().Set("owner", "repo", 2, data.FullsendStatus{
		ActiveAgents: []data.ActiveAgent{{Status: "in_progress"}},
	})
	m.Table.SetRows(m.BuildRows())
	m.Table.SetCurrItem(1)

	_, _ = m.Update(starKeyPressMsg())

	require.True(t, data.GetStarStore().IsStarred("pr:owner/repo#2"))
	require.False(t, data.GetStarStore().IsStarred("pr:owner/repo#1"))
}

func TestConfirmation_EmptyInputDoesNotConfirm(t *testing.T) {
	// Pressing Enter without typing anything should NOT confirm, since the
	// prompt says (y/N) indicating N is the default.
	m := newTestModel("close")

	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	_, _ = m.Update(msg)

	require.False(t, m.IsPromptConfirmationShown,
		"confirmation prompt should be dismissed")
}

func TestConfirmation_AcceptWithLowercaseY(t *testing.T) {
	m := newTestModel("merge")
	m.PromptConfirmationBox.SetValue("y")

	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "lowercase y should execute the action")
}

func TestConfirmation_AcceptWithUppercaseY(t *testing.T) {
	m := newTestModel("reopen")
	m.PromptConfirmationBox.SetValue("Y")

	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "uppercase Y should execute the action")
}

func TestConfirmation_RejectWithN(t *testing.T) {
	m := newTestModel("close")
	m.PromptConfirmationBox.SetValue("n")

	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	_, cmd := m.Update(msg)

	// cmd is a batch of (nil, blinkCmd) -- the nil means no action was taken.
	// We verify the prompt is dismissed regardless.
	require.False(t, m.IsPromptConfirmationShown,
		"confirmation prompt should be dismissed on rejection")
	_ = cmd
}

func TestConfirmation_CancelWithEsc(t *testing.T) {
	m := newTestModel("merge")

	msg := tea.KeyPressMsg{Code: tea.KeyEsc}
	_, cmd := m.Update(msg)

	require.False(t, m.IsPromptConfirmationShown,
		"Esc should dismiss the confirmation prompt")
	_ = cmd
}

func TestConfirmation_CancelWithCtrlC(t *testing.T) {
	m := newTestModel("update")

	msg := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	_, cmd := m.Update(msg)

	require.False(t, m.IsPromptConfirmationShown,
		"Ctrl+C should dismiss the confirmation prompt")
	_ = cmd
}

func TestConfirmation_AllActions(t *testing.T) {
	actions := []string{"close", "reopen", "ready", "merge", "update", "approveWorkflows"}

	for _, action := range actions {
		t.Run(action+"_empty_input_does_not_confirm", func(t *testing.T) {
			m := newTestModel(action)

			msg := tea.KeyPressMsg{Code: tea.KeyEnter}
			_, _ = m.Update(msg)

			require.False(t, m.IsPromptConfirmationShown,
				"empty input should dismiss prompt for action %q", action)
		})

		t.Run(action+"_explicit_y", func(t *testing.T) {
			m := newTestModel(action)
			m.PromptConfirmationBox.SetValue("y")

			msg := tea.KeyPressMsg{Code: tea.KeyEnter}
			_, cmd := m.Update(msg)

			require.NotNil(t, cmd,
				"explicit y should confirm for action %q", action)
		})
	}
}

func TestUpdatePRMsg_ResolvedThreadIdMarksMatchingThreadResolved(t *testing.T) {
	m := newTestModel("")
	m.Prs[0].Enriched.ReviewThreads.Nodes = []data.ReviewThreadWithComments{
		{Id: "thread-1", IsResolved: false},
		{Id: "thread-2", IsResolved: false},
	}
	threadId := "thread-2"

	_, _ = m.Update(tasks.UpdatePRMsg{PrNumber: 42, ResolvedThreadId: &threadId})

	require.False(t, m.Prs[0].Enriched.ReviewThreads.Nodes[0].IsResolved)
	require.True(t, m.Prs[0].Enriched.ReviewThreads.Nodes[1].IsResolved)
}

func TestUpdatePRMsg_ResolvedThreadIdDecrementsPrimaryUnresolvedCount(t *testing.T) {
	m := newTestModel("")
	m.Prs[0].Enriched.ReviewThreads.Nodes = []data.ReviewThreadWithComments{
		{Id: "thread-1", IsResolved: false},
	}
	m.Prs[0].Primary.ReviewThreads = data.ReviewThreadsWithResolution{
		Nodes: []struct{ IsResolved bool }{
			{IsResolved: false},
			{IsResolved: false},
		},
	}
	before := m.Prs[0].Primary.UnresolvedThreadsCount()
	threadId := "thread-1"

	_, _ = m.Update(tasks.UpdatePRMsg{PrNumber: 42, ResolvedThreadId: &threadId})

	require.Equal(t, before-1, m.Prs[0].Primary.UnresolvedThreadsCount(),
		"resolving a thread should decrement the list view's unresolved-thread count")
}

func TestUpdatePRMsg_NewThreadReplyAppendsCommentToMatchingThread(t *testing.T) {
	m := newTestModel("")
	m.Prs[0].Enriched.ReviewThreads.Nodes = []data.ReviewThreadWithComments{
		{Id: "thread-1"},
		{Id: "thread-2"},
	}
	reply := tasks.ThreadReply{
		ThreadId: "thread-2",
		Comment:  data.ReviewComment{Body: "sounds good"},
	}

	_, _ = m.Update(tasks.UpdatePRMsg{PrNumber: 42, NewThreadReply: &reply})

	require.Empty(t, m.Prs[0].Enriched.ReviewThreads.Nodes[0].Comments.Nodes)
	require.Len(t, m.Prs[0].Enriched.ReviewThreads.Nodes[1].Comments.Nodes, 1)
	require.Equal(t, "sounds good", m.Prs[0].Enriched.ReviewThreads.Nodes[1].Comments.Nodes[0].Body)
}

func withTestStarStore(t *testing.T) *data.StarStore {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "gh-dash-star-test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	store := data.NewStarStoreForTesting(filepath.Join(tempDir, "starred.json"))
	restore := data.OverrideStarStoreForTesting(store)
	t.Cleanup(restore)
	return store
}

func starKeyPressMsg() tea.KeyPressMsg {
	k := keys.PRKeys.Star.Keys()[0]
	return tea.KeyPressMsg{Code: rune(k[0]), Text: k}
}

func TestStar_TogglesSelectedRow(t *testing.T) {
	withTestStarStore(t)

	m := newTestModel("")
	m.IsPromptConfirmationShown = false

	msg := starKeyPressMsg()
	_, _ = m.Update(msg)

	require.True(t, data.GetStarStore().IsStarred("pr:#42"),
		"pressing the star key should star the selected PR")

	_, _ = m.Update(msg)

	require.False(t, data.GetStarStore().IsStarred("pr:#42"),
		"pressing the star key again should unstar the selected PR")
}

func TestStar_NoOpWithoutCurrRow(t *testing.T) {
	withTestStarStore(t)

	m := newTestModel("")
	m.IsPromptConfirmationShown = false
	m.Prs = nil

	msg := starKeyPressMsg()
	_, cmd := m.Update(msg)

	require.False(t, data.GetStarStore().IsStarred("pr:#42"))
	for _, msg := range collectMsgs(t, cmd) {
		if _, ok := msg.(constants.TaskFinishedMsg); ok {
			t.Fatal("no current row should not surface star feedback")
		}
	}
}
