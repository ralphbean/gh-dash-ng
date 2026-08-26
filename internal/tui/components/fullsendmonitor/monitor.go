package fullsendmonitor

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"github.com/dlvhdr/gh-dash/v4/internal/data"
)

// FullsendStatusUpdatedMsg is sent when fullsend agent status is updated
type FullsendStatusUpdatedMsg struct {
	Repo         string // "owner/repo"
	Number       int    // Issue or PR number
	ActiveAgents []data.ActiveAgent
	Error        error
}

// pollTickMsg is an internal message to trigger periodic polling
type pollTickMsg struct {
	timestamp time.Time
}

// PollStartedMsg is sent when a poll cycle starts
type PollStartedMsg struct {
	NumRepos int
}

// PollCompletedMsg is sent when a poll cycle completes
type PollCompletedMsg struct {
	NumRepos        int
	NumPRs          int
	NumActiveAgents int
	Error           error
}

// TriggerDelayedPollMsg triggers a delayed poll for a specific PR
type TriggerDelayedPollMsg struct {
	Owner  string
	Repo   string
	Number int
	Delay  time.Duration
}

// DelayedPollTickMsg is sent after the delay to trigger the actual poll
type DelayedPollTickMsg struct {
	Owner  string
	Repo   string
	Number int
}

// PRInfo represents a PR to monitor for fullsend workflows
type PRInfo struct {
	Owner  string
	Repo   string
	Number int
	Title  string
}

// RepoInfo aggregates all PRs and issues for a repository
type RepoInfo struct {
	Owner  string
	Repo   string
	PRs    map[int]string // number -> title
	Issues map[int]string // number -> title
}

// Monitor manages background polling of fullsend workflow status
type Monitor struct {
	cache              *data.FullsendCache
	visibleRepos       map[string]RepoInfo // Set of repos with their PRs (key: "owner/repo")
	activeWorkflows    map[string]int      // Count of active workflows per repo
	lastPollTime       time.Time
	retryAttempts      map[string]int      // Track retry attempts per repo
	enabled            bool
	lazyLoadComplete   bool          // Track if initial lazy load is done
	pollInterval       time.Duration // Polling interval
}

// NewMonitor creates a new fullsend status monitor
func NewMonitor(enabled bool, pollInterval time.Duration) *Monitor {
	return &Monitor{
		cache:            data.NewFullsendCache(),
		visibleRepos:     make(map[string]RepoInfo),
		activeWorkflows:  make(map[string]int),
		retryAttempts:    make(map[string]int),
		enabled:          enabled,
		lazyLoadComplete: false,
		pollInterval:     pollInterval,
	}
}

// Init initializes the monitor (lazy loading - no initial fetch)
func (m *Monitor) Init() tea.Cmd {
	if !m.enabled {
		return nil
	}
	// Lazy loading: skip initial fetch, populate on first tick
	return m.scheduleNextTick()
}

// Update handles messages for the monitor
func (m *Monitor) Update(msg tea.Msg) tea.Cmd {
	if !m.enabled {
		return nil
	}

	switch msg := msg.(type) {
	case pollTickMsg:
		// Automatic tick - poll and schedule next tick
		return m.pollVisibleRepos(true)
	case TriggerDelayedPollMsg:
		return m.scheduleDelayedPoll(msg)
	case DelayedPollTickMsg:
		return m.pollSinglePR(msg.Owner, msg.Repo, msg.Number)
	}

	return nil
}

// scheduleNextTick returns a command that sends a tick message after the appropriate interval
func (m *Monitor) scheduleNextTick() tea.Cmd {
	interval := m.calculatePollInterval()
	log.Debug("Scheduling next poll tick", "interval", interval, "interval_seconds", interval.Seconds())
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		log.Debug("Poll tick fired", "timestamp", t)
		return pollTickMsg{timestamp: t}
	})
}

// calculatePollInterval returns the configured polling interval
func (m *Monitor) calculatePollInterval() time.Duration {
	if m.pollInterval == 0 {
		// Default to 5 minutes if not configured
		log.Debug("Poll interval not configured, using default", "interval", "5m")
		return 5 * time.Minute
	}
	log.Debug("Using configured poll interval", "interval", m.pollInterval)
	return m.pollInterval
}

// SetVisiblePRs updates the set of PRs that are currently visible
// Aggregates PRs by repository to minimize API calls
func (m *Monitor) SetVisiblePRs(prs []PRInfo) {
	// Preserve existing issues when updating PRs
	existingRepos := m.visibleRepos
	m.visibleRepos = make(map[string]RepoInfo)

	// Aggregate PRs by repository
	for _, pr := range prs {
		repoKey := fmt.Sprintf("%s/%s", pr.Owner, pr.Repo)

		repo, exists := m.visibleRepos[repoKey]
		if !exists {
			// Check if we have existing issues for this repo
			if existingRepo, hasExisting := existingRepos[repoKey]; hasExisting {
				repo = RepoInfo{
					Owner:  pr.Owner,
					Repo:   pr.Repo,
					PRs:    make(map[int]string),
					Issues: existingRepo.Issues,
				}
			} else {
				repo = RepoInfo{
					Owner:  pr.Owner,
					Repo:   pr.Repo,
					PRs:    make(map[int]string),
					Issues: make(map[int]string),
				}
			}
		}
		repo.PRs[pr.Number] = pr.Title
		m.visibleRepos[repoKey] = repo
	}
}

// SetVisibleIssues updates the set of issues that are currently visible
// Aggregates issues by repository to minimize API calls
func (m *Monitor) SetVisibleIssues(issues []PRInfo) { // Reuse PRInfo structure
	// Preserve existing PRs when updating issues
	existingRepos := m.visibleRepos
	if m.visibleRepos == nil {
		m.visibleRepos = make(map[string]RepoInfo)
	}

	// Aggregate issues by repository
	for _, issue := range issues {
		repoKey := fmt.Sprintf("%s/%s", issue.Owner, issue.Repo)

		repo, exists := m.visibleRepos[repoKey]
		if !exists {
			// Check if we have existing PRs for this repo
			if existingRepo, hasExisting := existingRepos[repoKey]; hasExisting {
				repo = RepoInfo{
					Owner:  issue.Owner,
					Repo:   issue.Repo,
					PRs:    existingRepo.PRs,
					Issues: make(map[int]string),
				}
			} else {
				repo = RepoInfo{
					Owner:  issue.Owner,
					Repo:   issue.Repo,
					PRs:    make(map[int]string),
					Issues: make(map[int]string),
				}
			}
		}
		if repo.Issues == nil {
			repo.Issues = make(map[int]string)
		}
		repo.Issues[issue.Number] = issue.Title
		m.visibleRepos[repoKey] = repo
	}
}

// pollVisibleRepos queries workflow status for all visible repositories
// Returns a batch command that will fetch status for each repo
// scheduleNext indicates whether to schedule the next automatic tick (true for periodic ticks, false for manual triggers)
func (m *Monitor) pollVisibleRepos(scheduleNext bool) tea.Cmd {
	if !m.enabled {
		return nil
	}

	m.lastPollTime = time.Now()

	// Lazy loading on first tick
	if !m.lazyLoadComplete {
		m.lazyLoadComplete = true
		log.Debug("Fullsend monitor: first tick (lazy load complete)")
	}

	log.Debug("pollVisibleRepos called", "num_repos", len(m.visibleRepos), "schedule_next", scheduleNext)

	// Create a snapshot of visible repos to poll
	reposToList := make([]struct {
		key  string
		info RepoInfo
	}, 0, len(m.visibleRepos))
	for key, info := range m.visibleRepos {
		reposToList = append(reposToList, struct {
			key  string
			info RepoInfo
		}{key, info})
	}

	numRepos := len(reposToList)
	if numRepos == 0 {
		// Schedule next tick even if no repos (but only if requested)
		if scheduleNext {
			return m.scheduleNextTick()
		}
		return nil
	}

	// Send poll started message, then execute poll, then send completed message
	startCmd := func() tea.Msg {
		return PollStartedMsg{NumRepos: numRepos}
	}

	pollCmd := func() tea.Msg {
		log.Debug("Poll started", "num_repos", numRepos)

		// Poll all repos synchronously (they make parallel API calls internally)
		totalPRs := 0
		totalActiveAgents := 0
		for _, repoEntry := range reposToList {
			totalPRs += len(repoEntry.info.PRs) + len(repoEntry.info.Issues)
			// Execute the poll synchronously
			m.pollRepoSync(repoEntry.key, repoEntry.info)
		}

		// Count total active agents across all repos
		for _, count := range m.activeWorkflows {
			totalActiveAgents += count
		}

		log.Debug("Poll completed", "num_repos", numRepos, "num_prs", totalPRs, "num_active_agents", totalActiveAgents)

		return PollCompletedMsg{
			NumRepos:        numRepos,
			NumPRs:          totalPRs,
			NumActiveAgents: totalActiveAgents,
			Error:           nil,
		}
	}

	// Batch poll commands, and optionally schedule next tick
	cmds := []tea.Cmd{startCmd, pollCmd}
	if scheduleNext {
		cmds = append(cmds, m.scheduleNextTick())
	}
	return tea.Batch(cmds...)
}

// GetCache returns the monitor's cache (for integration with other components)
func (m *Monitor) GetCache() *data.FullsendCache {
	return m.cache
}

// IsEnabled returns whether the monitor is enabled
func (m *Monitor) IsEnabled() bool {
	return m.enabled
}

// SetPollInterval updates the polling interval
func (m *Monitor) SetPollInterval(interval time.Duration) {
	m.pollInterval = interval
}

// TriggerPoll triggers an immediate poll of all visible repos
// Does not schedule the next automatic tick (that's handled by the periodic timer)
func (m *Monitor) TriggerPoll() tea.Cmd {
	if !m.enabled {
		log.Debug("TriggerPoll: monitor not enabled")
		return nil
	}
	log.Debug("TriggerPoll: calling pollVisibleRepos", "num_repos", len(m.visibleRepos))
	return m.pollVisibleRepos(false) // Don't schedule next tick for manual polls
}

// pollRepoSync queries workflow status for a single repository and matches to PRs by head_sha
// This function is called synchronously from pollVisibleRepos
func (m *Monitor) pollRepoSync(repoKey string, repo RepoInfo) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("panic in pollRepoSync", "repoKey", repoKey, "panic", r)
		}
	}()

	log.Debug("pollRepoSync invoked",
		"repoKey", repoKey,
		"owner", repo.Owner,
		"repo", repo.Repo,
		"numPRs", len(repo.PRs),
		"numIssues", len(repo.Issues))

	// Query workflow runs for this repo (one API call for all PRs and issues in this repo)
	log.Debug("About to call QueryWorkflowRuns",
		"owner", repo.Owner,
		"repo", repo.Repo)

	runs, rateLimit, err := data.QueryWorkflowRuns(repo.Owner, repo.Repo, 50)

	log.Debug("QueryWorkflowRuns returned",
		"owner", repo.Owner,
		"repo", repo.Repo,
		"num_runs", len(runs),
		"err", err)

	if err != nil {
		// Log error but continue (graceful degradation)
		log.Error("Failed to query workflow runs for repo",
			"repo", repoKey,
			"error", err,
			"attempt", m.retryAttempts[repoKey])

		// Track retry attempt
		m.retryAttempts[repoKey]++

		// If we're hitting rate limits, back off
		if rateLimit != nil && rateLimit.ShouldBackoff() {
			log.Warn("Rate limit approaching, backing off",
				"repo", repoKey,
				"remaining", rateLimit.Remaining,
				"limit", rateLimit.Limit)
			return
		}

		return
	}

	// Reset retry counter on success
	m.retryAttempts[repoKey] = 0

	// Build reverse maps: title -> number for both PRs and issues
	prTitleToNumber := make(map[string]int)
	for number, title := range repo.PRs {
		prTitleToNumber[title] = number
	}
	issueTitleToNumber := make(map[string]int)
	for number, title := range repo.Issues {
		issueTitleToNumber[title] = number
	}

	log.Debug("Querying workflows for repo",
		"repo", repoKey,
		"num_prs", len(repo.PRs),
		"num_issues", len(repo.Issues),
		"num_runs", len(runs))

	// Process fullsend workflows and match to PRs/issues by display_title
	activeCount := 0
	store := data.GetFullsendStatusStore()

	for _, run := range runs {
		if run.Name != "fullsend" {
			continue
		}

		// Try to match workflow run to PR or issue by display_title
		prNumber, isPR := prTitleToNumber[run.DisplayTitle]
		issueNumber, isIssue := issueTitleToNumber[run.DisplayTitle]

		if !isPR && !isIssue {
			// This workflow doesn't match any visible PR or issue
			log.Debug("Workflow run doesn't match any visible PR or issue",
				"repo", repoKey,
				"runID", run.Id,
				"displayTitle", run.DisplayTitle)
			continue
		}

		var number int
		var itemType string
		if isPR {
			number = prNumber
			itemType = "PR"
		} else {
			number = issueNumber
			itemType = "issue"
		}

		log.Debug("Matched workflow run to item",
			"repo", repoKey,
			"type", itemType,
			"number", number,
			"runID", run.Id,
			"displayTitle", run.DisplayTitle)

		// Detect agents from this workflow run
		agents, _, err := data.DetectFullsendAgents(repo.Owner, repo.Repo, run)
		if err != nil {
			log.Warn("Failed to detect agents from workflow run",
				"repo", repoKey,
				"type", itemType,
				"number", number,
				"runID", run.Id,
				"error", err)
			continue
		}

		if len(agents) == 0 {
			continue
		}

		// Filter to only active agents (in_progress or queued)
		activeAgents := data.FilterActiveAgents(agents)

		if len(activeAgents) == 0 {
			continue
		}

		// Count active workflows for this item
		activeCount++

		// Convert DetectedAgent to ActiveAgent and populate store
		var storeAgents []data.ActiveAgent
		for _, agent := range activeAgents {
			storeAgents = append(storeAgents, data.ActiveAgent{
				Type:       agent.Type,
				WorkflowID: agent.WorkflowID,
				Status:     agent.Status,
				StartedAt:  agent.StartedAt,
			})
		}

		log.Debug("Updating fullsend status",
			"repo", repoKey,
			"type", itemType,
			"number", number,
			"displayTitle", run.DisplayTitle,
			"agents", len(storeAgents))

		store.Set(repo.Owner, repo.Repo, number, data.FullsendStatus{
			ActiveAgents: storeAgents,
			LastChecked:  time.Now(),
		})
	}

	// Update active workflow count for this repo
	m.activeWorkflows[repoKey] = activeCount
}

// scheduleDelayedPoll schedules a delayed poll for a specific PR
func (m *Monitor) scheduleDelayedPoll(msg TriggerDelayedPollMsg) tea.Cmd {
	return tea.Tick(msg.Delay, func(t time.Time) tea.Msg {
		return DelayedPollTickMsg{
			Owner:  msg.Owner,
			Repo:   msg.Repo,
			Number: msg.Number,
		}
	})
}

// pollSinglePR polls workflow status for a single PR or issue
func (m *Monitor) pollSinglePR(owner, repo string, number int) tea.Cmd {
	return func() tea.Msg {
		log.Debug("Polling single item",
			"owner", owner,
			"repo", repo,
			"number", number)

		// Query workflow runs for this repo
		runs, _, err := data.QueryWorkflowRuns(owner, repo, 50)
		if err != nil {
			log.Error("Failed to query workflow runs for single item",
				"owner", owner,
				"repo", repo,
				"number", number,
				"error", err)
			return nil
		}

		store := data.GetFullsendStatusStore()

		// Process fullsend workflows
		for _, run := range runs {
			if run.Name != "fullsend" {
				continue
			}

			// Detect agents from this workflow run
			agents, _, err := data.DetectFullsendAgents(owner, repo, run)
			if err != nil {
				log.Warn("Failed to detect agents from workflow run",
					"owner", owner,
					"repo", repo,
					"number", number,
					"runID", run.Id,
					"error", err)
				continue
			}

			if len(agents) == 0 {
				continue
			}

			// Filter to only active agents (in_progress or queued)
			activeAgents := data.FilterActiveAgents(agents)

			if len(activeAgents) == 0 {
				continue
			}

			// Convert DetectedAgent to ActiveAgent and populate store
			var storeAgents []data.ActiveAgent
			for _, agent := range activeAgents {
				storeAgents = append(storeAgents, data.ActiveAgent{
					Type:       agent.Type,
					WorkflowID: agent.WorkflowID,
					Status:     agent.Status,
					StartedAt:  agent.StartedAt,
				})
			}

			log.Debug("Updating fullsend status for single item",
				"owner", owner,
				"repo", repo,
				"number", number,
				"agents", len(storeAgents))

			store.Set(owner, repo, number, data.FullsendStatus{
				ActiveAgents: storeAgents,
				LastChecked:  time.Now(),
			})

			// Only process the first matching run
			break
		}

		return nil
	}
}

