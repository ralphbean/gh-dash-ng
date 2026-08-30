package fullsendmonitor

import (
	"errors"
	"testing"
	"time"

	"github.com/dlvhdr/gh-dash/v4/internal/data"
)

func TestPollRepoSyncClearsCompletedActivity(t *testing.T) {
	store := data.GetFullsendStatusStore()
	store.Clear()
	store.Set("owner", "repo", 42, activeStatus())

	monitor := newTestMonitor()
	monitor.queryWorkflowRuns = func(string, string, int) ([]data.WorkflowRun, *data.RateLimitInfo, error) {
		return nil, nil, nil
	}

	monitor.pollRepoSync("owner/repo", testRepo())

	if agents := store.Get("owner", "repo", 42).ActiveAgents; len(agents) != 0 {
		t.Fatalf("expected completed activity to be cleared, got %d active agents", len(agents))
	}
}

func TestPollRepoSyncPreservesStatusOnRunQueryError(t *testing.T) {
	store := data.GetFullsendStatusStore()
	store.Clear()
	store.Set("owner", "repo", 42, activeStatus())

	monitor := newTestMonitor()
	monitor.queryWorkflowRuns = func(string, string, int) ([]data.WorkflowRun, *data.RateLimitInfo, error) {
		return nil, nil, errors.New("run query failed")
	}

	monitor.pollRepoSync("owner/repo", testRepo())

	if agents := store.Get("owner", "repo", 42).ActiveAgents; len(agents) != 1 {
		t.Fatalf("expected prior status to be preserved, got %d active agents", len(agents))
	}
}

func TestPollRepoSyncPreservesStatusOnJobQueryError(t *testing.T) {
	store := data.GetFullsendStatusStore()
	store.Clear()
	store.Set("owner", "repo", 42, activeStatus())

	monitor := newTestMonitor()
	monitor.queryWorkflowRuns = func(string, string, int) ([]data.WorkflowRun, *data.RateLimitInfo, error) {
		return []data.WorkflowRun{{Id: 100, Name: "fullsend", DisplayTitle: "target"}}, nil, nil
	}
	monitor.detectAgents = func(string, string, data.WorkflowRun) ([]data.DetectedAgent, *data.RateLimitInfo, error) {
		return nil, nil, errors.New("job query failed")
	}

	monitor.pollRepoSync("owner/repo", testRepo())

	if agents := store.Get("owner", "repo", 42).ActiveAgents; len(agents) != 1 {
		t.Fatalf("expected prior status to be preserved, got %d active agents", len(agents))
	}
}

func TestPollSinglePRRejectsUnrelatedRun(t *testing.T) {
	store := data.GetFullsendStatusStore()
	store.Clear()
	store.Set("owner", "repo", 42, activeStatus())

	monitor := newTestMonitor()
	monitor.queryWorkflowRuns = func(string, string, int) ([]data.WorkflowRun, *data.RateLimitInfo, error) {
		return []data.WorkflowRun{{Id: 100, Name: "fullsend", DisplayTitle: "some other PR"}}, nil, nil
	}
	monitor.detectAgents = func(string, string, data.WorkflowRun) ([]data.DetectedAgent, *data.RateLimitInfo, error) {
		t.Fatal("detector called for an unrelated workflow run")
		return nil, nil, nil
	}

	monitor.pollSinglePR("owner", "repo", 42, "target")()

	if agents := store.Get("owner", "repo", 42).ActiveAgents; len(agents) != 0 {
		t.Fatalf("expected unrelated run to leave the item inactive, got %d active agents", len(agents))
	}
}

func newTestMonitor() *Monitor {
	return NewMonitor(true, time.Minute)
}

func testRepo() RepoInfo {
	return RepoInfo{
		Owner:  "owner",
		Repo:   "repo",
		PRs:    map[int]string{42: "target"},
		Issues: map[int]string{},
	}
}

func activeStatus() data.FullsendStatus {
	return data.FullsendStatus{
		ActiveAgents: []data.ActiveAgent{{Type: "Code", Status: "in_progress"}},
		LastChecked:  time.Now(),
	}
}
