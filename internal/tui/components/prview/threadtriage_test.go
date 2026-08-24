package prview

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/dlvhdr/gh-dash/v4/internal/config"
	"github.com/dlvhdr/gh-dash/v4/internal/data"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/prrow"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/context"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/theme"
)

func newTestModelForTriage(t *testing.T) Model {
	t.Helper()
	cfg, err := config.ParseConfig(config.Location{
		ConfigFlag:       "../../../config/testdata/test-config.yml",
		SkipGlobalConfig: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	thm := theme.ParseTheme(&cfg)
	ctx := &context.ProgramContext{
		Config:    &cfg,
		Theme:     thm,
		Styles:    context.InitStyles(thm),
		StartTask: func(context.Task) tea.Cmd { return nil },
	}

	m := NewModel(ctx)
	m.ctx = ctx
	m.pr = &prrow.PullRequest{
		Ctx: ctx,
		Data: &prrow.Data{
			Primary:    &data.PullRequestData{Url: "https://github.com/o/r/pull/1", Number: 1},
			IsEnriched: true,
		},
	}
	return m
}

// withTriageThreads puts the model directly into an active triage session
// with the given queue, bypassing the fetch in EnterThreadTriage.
func withTriageThreads(m *Model, threads []data.ReviewThreadWithComments) {
	m.threadTriage = threadTriageState{
		active:       true,
		threads:      threads,
		currentIndex: 0,
		prevTab:      m.carousel.SelectedItem(),
	}
}

func TestEnterThreadTriage_ReturnsCommandRegardlessOfEnrichment(t *testing.T) {
	for _, enriched := range []bool{true, false} {
		m := newTestModelForTriage(t)
		m.pr.Data.IsEnriched = enriched

		cmd := m.EnterThreadTriage()

		require.NotNil(
			t,
			cmd,
			"EnterThreadTriage should always refetch, regardless of IsEnriched=%v",
			enriched,
		)
	}
}

func TestEnterThreadTriage_NilWithNoPR(t *testing.T) {
	m := newTestModelForTriage(t)
	m.pr = nil

	require.Nil(t, m.EnterThreadTriage())
}

func TestSetReviewThreads_FiltersSortsAndActivatesTriage(t *testing.T) {
	m := newTestModelForTriage(t)

	threads := []data.ReviewThreadWithComments{
		{Id: "resolved", Path: "a.go", Line: 1, IsResolved: true},
		{Id: "z-file", Path: "z.go", Line: 5, IsResolved: false},
		{Id: "a-file-line10", Path: "a.go", Line: 10, IsResolved: false},
		{Id: "a-file-line2", Path: "a.go", Line: 2, IsResolved: false},
	}

	m.SetReviewThreads(ReviewThreadsFetchedMsg{Threads: threads})

	require.True(t, m.IsTriagingThreads())
	require.Len(t, m.threadTriage.threads, 3, "resolved threads should be excluded from the queue")
	require.Equal(t, "a-file-line2", m.threadTriage.threads[0].Id)
	require.Equal(t, "a-file-line10", m.threadTriage.threads[1].Id)
	require.Equal(t, "z-file", m.threadTriage.threads[2].Id)
	require.Equal(t, 0, m.threadTriage.currentIndex)
}

func TestSetReviewThreads_NoUnresolvedThreadsStaysInTriageWithEmptyQueue(t *testing.T) {
	m := newTestModelForTriage(t)

	m.SetReviewThreads(ReviewThreadsFetchedMsg{Threads: []data.ReviewThreadWithComments{
		{Id: "resolved", IsResolved: true},
	}})

	require.True(t, m.IsTriagingThreads(), "triage should remain active with an empty queue")
	require.Empty(t, m.threadTriage.threads)
	_, ok := m.currentThread()
	require.False(t, ok)
}

func TestSetReviewThreads_ErrorDoesNotEnterTriage(t *testing.T) {
	m := newTestModelForTriage(t)

	m.SetReviewThreads(ReviewThreadsFetchedMsg{Err: errors.New("fetch failed")})

	require.False(t, m.IsTriagingThreads())
}

func TestNextPrevThread_WrapAndLeaveStateUnchanged(t *testing.T) {
	m := newTestModelForTriage(t)
	withTriageThreads(&m, []data.ReviewThreadWithComments{
		{Id: "t1"}, {Id: "t2"}, {Id: "t3"},
	})

	m.NextThread()
	require.Equal(t, 1, m.threadTriage.currentIndex)

	m.NextThread()
	m.NextThread()
	require.Equal(
		t,
		0,
		m.threadTriage.currentIndex,
		"next from the last thread should wrap to the first",
	)

	m.PrevThread()
	require.Equal(
		t,
		2,
		m.threadTriage.currentIndex,
		"prev from the first thread should wrap to the last",
	)

	for _, thread := range m.threadTriage.threads {
		require.False(t, thread.IsResolved)
		require.Empty(t, thread.Comments.Nodes)
	}
}

func TestResolveCurrentThread_NotAllowedIsNoOp(t *testing.T) {
	m := newTestModelForTriage(t)
	withTriageThreads(&m, []data.ReviewThreadWithComments{
		{Id: "t1", ViewerCanResolve: false},
	})

	cmd := m.ResolveCurrentThread()

	require.Nil(t, cmd)
	require.Len(
		t,
		m.threadTriage.threads,
		1,
		"thread should remain in the queue when resolve isn't allowed",
	)
}

func TestResolveCurrentThread_AllowedRemovesFromQueueAndAdvances(t *testing.T) {
	m := newTestModelForTriage(t)
	withTriageThreads(&m, []data.ReviewThreadWithComments{
		{Id: "t1", ViewerCanResolve: true},
		{Id: "t2", ViewerCanResolve: true},
	})

	cmd := m.ResolveCurrentThread()

	require.NotNil(t, cmd)
	require.True(
		t,
		m.IsTriagingThreads(),
		"triage should remain active with threads left in the queue",
	)
	require.Len(t, m.threadTriage.threads, 1)
	require.Equal(t, "t2", m.threadTriage.threads[0].Id)
	require.Equal(t, 0, m.threadTriage.currentIndex)
}

func TestResolveCurrentThread_LastThreadExitsTriage(t *testing.T) {
	m := newTestModelForTriage(t)
	withTriageThreads(&m, []data.ReviewThreadWithComments{
		{Id: "t1", ViewerCanResolve: true},
	})

	cmd := m.ResolveCurrentThread()

	require.NotNil(t, cmd)
	require.False(t, m.IsTriagingThreads(), "resolving the last thread should exit triage")
}

func TestStartThreadReply_NotAllowedIsNoOp(t *testing.T) {
	m := newTestModelForTriage(t)
	withTriageThreads(&m, []data.ReviewThreadWithComments{
		{Id: "t1", ViewerCanReply: false},
	})

	cmd := m.StartThreadReply()

	require.Nil(t, cmd)
	require.False(t, m.IsTextInputBoxFocused())
}

func TestStartThreadReply_AllowedOpensEditor(t *testing.T) {
	m := newTestModelForTriage(t)
	withTriageThreads(&m, []data.ReviewThreadWithComments{
		{Id: "t1", ViewerCanReply: true},
	})

	cmd := m.StartThreadReply()

	require.NotNil(t, cmd)
	require.True(t, m.IsTextInputBoxFocused())
}

func TestSubmitThreadReply_PostsToCurrentThreadWithoutAdvancing(t *testing.T) {
	m := newTestModelForTriage(t)
	withTriageThreads(&m, []data.ReviewThreadWithComments{
		{Id: "t1", ViewerCanReply: true},
		{Id: "t2", ViewerCanReply: true},
	})

	cmd := m.StartThreadReply()
	require.NotNil(t, cmd)
	m.editor.SetValue("sounds good")

	newM, submitCmd := m.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	m = newM

	require.NotNil(t, submitCmd, "submitting a non-empty reply should return a command")
	require.Equal(
		t,
		0,
		m.threadTriage.currentIndex,
		"replying should not advance to the next thread",
	)
	require.Len(
		t,
		m.threadTriage.threads,
		2,
		"replying should not remove the thread from the queue",
	)
}

func TestExitThreadTriage_RestoresPriorTab(t *testing.T) {
	m := newTestModelForTriage(t)
	m.carousel.MoveRight()
	prevTab := m.carousel.SelectedItem()
	withTriageThreads(&m, []data.ReviewThreadWithComments{{Id: "t1"}})
	m.carousel.MoveRight()
	require.NotEqual(t, prevTab, m.carousel.SelectedItem())

	m.ExitThreadTriage()

	require.False(t, m.IsTriagingThreads())
	require.Equal(t, prevTab, m.carousel.SelectedItem())
}

func TestViewThreadTriage_EmptyStateWhenQueueIsEmpty(t *testing.T) {
	m := newTestModelForTriage(t)
	m.SetReviewThreads(ReviewThreadsFetchedMsg{Threads: nil})

	view := m.View()

	require.Contains(t, view, "No unresolved review threads.")
}

func TestViewThreadTriage_OutdatedIndicator(t *testing.T) {
	m := newTestModelForTriage(t)
	withTriageThreads(&m, []data.ReviewThreadWithComments{
		{Id: "t1", Path: "a.go", Line: 1, IsOutdated: true},
	})

	view := m.View()

	require.Contains(t, view, "OUTDATED")
}

func TestAppendThreadReply_AppendsToMatchingTriageThread(t *testing.T) {
	m := newTestModelForTriage(t)
	withTriageThreads(&m, []data.ReviewThreadWithComments{
		{Id: "t1", Comments: data.ReviewComments{Nodes: []data.ReviewComment{{Body: "original"}}}},
		{Id: "t2"},
	})

	reply := data.ReviewComment{Body: "sounds good", Author: struct{ Login string }{Login: "me"}}
	m.AppendThreadReply("t1", reply)

	require.Len(t, m.threadTriage.threads[0].Comments.Nodes, 2)
	require.Equal(t, "sounds good", m.threadTriage.threads[0].Comments.Nodes[1].Body)
	require.Empty(
		t,
		m.threadTriage.threads[1].Comments.Nodes,
		"unrelated thread should be unchanged",
	)
}

func TestAppendThreadReply_NoOpWhenNotTriaging(t *testing.T) {
	m := newTestModelForTriage(t)

	reply := data.ReviewComment{Body: "sounds good"}
	m.AppendThreadReply("t1", reply) // should not panic
}

func TestAppendThreadReply_NoOpWhenThreadNotInQueue(t *testing.T) {
	m := newTestModelForTriage(t)
	withTriageThreads(&m, []data.ReviewThreadWithComments{
		{Id: "t1"},
	})

	reply := data.ReviewComment{Body: "sounds good"}
	m.AppendThreadReply("nonexistent", reply)

	require.Empty(t, m.threadTriage.threads[0].Comments.Nodes)
}

func TestViewThreadTriage_NoOutdatedIndicatorWhenNotOutdated(t *testing.T) {
	m := newTestModelForTriage(t)
	withTriageThreads(&m, []data.ReviewThreadWithComments{
		{Id: "t1", Path: "a.go", Line: 1, IsOutdated: false},
	})

	view := m.View()

	require.NotContains(t, view, "OUTDATED")
}
