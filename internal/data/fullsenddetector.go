// Package data provides fullsend agent detection from GitHub workflow runs.
//
// Workflow Detection Strategy:
//
// Fullsend uses a reusable workflow architecture where all agent execution happens
// as jobs within a single "fullsend" workflow run. The workflow detection strategy:
//
// 1. Query workflow runs for a repository using the GitHub Actions API
// 2. Filter to runs with name "fullsend"
// 3. For each fullsend run, fetch its jobs via the jobs API
// 4. Extract agent type from job names matching pattern "dispatch / <AgentType>"
// 5. Agent types are dynamically extracted - no hardcoding needed
//
// This approach works for any agent type (Code, Review, Triage, Fix, Retro, etc.)
// without requiring workflow file parsing. All information is available from the
// GitHub Actions API.
//
// Example job names:
//   - "dispatch / Review" -> agent type "Review"
//   - "dispatch / Code" -> agent type "Code"
//   - "dispatch / Triage" -> agent type "Triage"
package data

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"charm.land/log/v2"
	gh "github.com/cli/go-gh/v2/pkg/api"
)

// WorkflowJob represents a job within a workflow run
type WorkflowJob struct {
	ID          int64     `json:"id"`
	RunID       int64     `json:"run_id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`     // queued, in_progress, completed
	Conclusion  string    `json:"conclusion"` // success, failure, cancelled, skipped
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// WorkflowJobsResponse represents the response from the workflow jobs API
type WorkflowJobsResponse struct {
	TotalCount int           `json:"total_count"`
	Jobs       []WorkflowJob `json:"jobs"`
}

// GetWorkflowJobs fetches jobs for a specific workflow run
// Uses GitHub REST API: GET /repos/{owner}/{repo}/actions/runs/{run_id}/jobs
func GetWorkflowJobs(owner, repo string, runID int64) ([]WorkflowJob, *RateLimitInfo, error) {
	client, err := gh.DefaultRESTClient()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	path := fmt.Sprintf("repos/%s/%s/actions/runs/%d/jobs", owner, repo, runID)

	log.Debug("GitHub REST API call", "method", "GET", "path", path)
	resp, err := client.Request(http.MethodGet, path, nil)
	if err != nil {
		log.Error("GitHub REST API call failed", "method", "GET", "path", path, "error", err)
		return nil, nil, fmt.Errorf("failed to query workflow jobs: %w", err)
	}
	defer resp.Body.Close()

	rateLimit := parseRateLimitHeaders(resp.Header)
	log.Debug("GitHub REST API response", "method", "GET", "path", path, "status", resp.StatusCode,
		"rate_remaining", rateLimit.Remaining, "rate_limit", rateLimit.Limit)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, rateLimit, fmt.Errorf(
			"GitHub API returned %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var result WorkflowJobsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, rateLimit, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Debug(
		"Fetched workflow jobs",
		"repo",
		fmt.Sprintf("%s/%s", owner, repo),
		"runID",
		runID,
		"count",
		len(result.Jobs),
	)
	return result.Jobs, rateLimit, nil
}

// ExtractAgentType extracts the agent type from a fullsend job name
// Pattern: "dispatch / <AgentType>" (e.g., "dispatch / Review", "dispatch / Code")
// Returns the agent type (e.g., "Review", "Code") and true if the pattern matches,
// or empty string and false if it doesn't match
func ExtractAgentType(jobName string) (string, bool) {
	const prefix = "dispatch / "
	if strings.HasPrefix(jobName, prefix) {
		agentType := strings.TrimPrefix(jobName, prefix)
		// Trim any additional whitespace
		agentType = strings.TrimSpace(agentType)
		if agentType == "" {
			return "", false
		}

		// Filter out infrastructure jobs (not actual agents)
		// These are orchestration/routing jobs in the fullsend workflow
		infraJobs := []string{
			"Harness dispatch", // Orchestrator
			"Route",            // Routing logic
		}
		for _, infraJob := range infraJobs {
			if agentType == infraJob {
				return "", false
			}
		}

		return agentType, true
	}
	return "", false
}

// DetectedAgent represents a fullsend agent detected from a workflow run
type DetectedAgent struct {
	Type       string    // Agent type (e.g., "Review", "Code", "Triage", "Fix")
	WorkflowID int64     // Workflow run ID
	JobID      int64     // Job ID within the workflow run
	Status     string    // Job status: "queued", "in_progress", "completed"
	Conclusion string    // Job conclusion: "success", "failure", "cancelled", "skipped" (only when status is "completed")
	StartedAt  time.Time // When the job started
}

// DetectFullsendAgents identifies active fullsend agents from workflow run jobs
// Examines jobs for the given workflow run and extracts agent information from job names
// Returns all detected agents, including those in terminal states (for caller to filter if needed)
func DetectFullsendAgents(
	owner, repo string,
	run WorkflowRun,
) ([]DetectedAgent, *RateLimitInfo, error) {
	// Check if this is a fullsend workflow by name
	if run.Name != "fullsend" {
		// Not a fullsend workflow, return empty list (not an error)
		return []DetectedAgent{}, nil, nil
	}

	// Fetch jobs for this workflow run
	jobs, rateLimit, err := GetWorkflowJobs(owner, repo, run.Id)
	if err != nil {
		return nil, rateLimit, fmt.Errorf(
			"failed to fetch jobs for workflow run %d: %w",
			run.Id,
			err,
		)
	}

	agents := []DetectedAgent{}
	for _, job := range jobs {
		agentType, matches := ExtractAgentType(job.Name)
		if matches {
			agents = append(agents, DetectedAgent{
				Type:       agentType,
				WorkflowID: run.Id,
				JobID:      job.ID,
				Status:     job.Status,
				Conclusion: job.Conclusion,
				StartedAt:  job.StartedAt,
			})
		}
		// Jobs that don't match the pattern are silently ignored (graceful fallback)
	}

	if len(agents) > 0 {
		log.Debug("Detected fullsend agents",
			"repo", fmt.Sprintf("%s/%s", owner, repo),
			"runID", run.Id,
			"agents", len(agents))
	}

	return agents, rateLimit, nil
}

// FilterActiveAgents filters a list of detected agents to only those that are currently active
// Active means status is "queued" or "in_progress"
func FilterActiveAgents(agents []DetectedAgent) []DetectedAgent {
	active := []DetectedAgent{}
	for _, agent := range agents {
		if agent.Status == "queued" || agent.Status == "in_progress" {
			active = append(active, agent)
		}
	}
	return active
}

// DetectActiveFullsendAgents is a convenience function that detects and filters to only active agents
func DetectActiveFullsendAgents(
	owner, repo string,
	run WorkflowRun,
) ([]DetectedAgent, *RateLimitInfo, error) {
	agents, rateLimit, err := DetectFullsendAgents(owner, repo, run)
	if err != nil {
		return nil, rateLimit, err
	}
	return FilterActiveAgents(agents), rateLimit, nil
}

// DetectFullsendAgentsFromMultipleRuns processes multiple workflow runs and detects fullsend agents
// Handles multiple concurrent agents across different runs for the same issue/PR
// Returns all detected agents (caller can filter by active/completed as needed)
func DetectFullsendAgentsFromMultipleRuns(
	owner, repo string,
	runs []WorkflowRun,
) ([]DetectedAgent, *RateLimitInfo, error) {
	allAgents := []DetectedAgent{}
	var lastRateLimit *RateLimitInfo

	for _, run := range runs {
		agents, rateLimit, err := DetectFullsendAgents(owner, repo, run)
		if err != nil {
			// Log error but continue with other runs (graceful degradation)
			log.Warn("Failed to detect agents from workflow run",
				"repo", fmt.Sprintf("%s/%s", owner, repo),
				"runID", run.Id,
				"error", err)
			continue
		}

		allAgents = append(allAgents, agents...)

		if rateLimit != nil {
			lastRateLimit = rateLimit
			// Check if we should back off to avoid rate limiting
			if rateLimit.ShouldBackoff() {
				log.Warn("Approaching rate limit, stopping agent detection early",
					"remaining", rateLimit.Remaining,
					"limit", rateLimit.Limit,
					"processed", len(allAgents),
					"total_runs", len(runs))
				break
			}
		}
	}

	return allAgents, lastRateLimit, nil
}
