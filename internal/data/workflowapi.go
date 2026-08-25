package data

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"charm.land/log/v2"
	gh "github.com/cli/go-gh/v2/pkg/api"
)

// RateLimitInfo tracks GitHub API rate limit status
type RateLimitInfo struct {
	Remaining int
	Limit     int
	ResetAt   time.Time
}

// QueryWorkflowRuns fetches workflow runs for a repository
// Uses the GitHub REST API: GET /repos/{owner}/{repo}/actions/runs
func QueryWorkflowRuns(owner, repo string, perPage int) ([]WorkflowRun, *RateLimitInfo, error) {
	client, err := gh.DefaultRESTClient()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	path := fmt.Sprintf("repos/%s/%s/actions/runs?per_page=%d", owner, repo, perPage)

	resp, err := client.Request(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query workflow runs: %w", err)
	}
	defer resp.Body.Close()

	// Parse rate limit headers
	rateLimit := parseRateLimitHeaders(resp.Header)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, rateLimit, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var result WorkflowRunsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, rateLimit, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Debug("Fetched workflow runs", "repo", fmt.Sprintf("%s/%s", owner, repo), "count", len(result.WorkflowRuns))
	return result.WorkflowRuns, rateLimit, nil
}

// QueryWorkflowRunsForPR fetches workflow runs associated with a specific PR
// Filters by the PR's head SHA
func QueryWorkflowRunsForPR(owner, repo string, prNumber int, headSHA string) ([]WorkflowRun, *RateLimitInfo, error) {
	client, err := gh.DefaultRESTClient()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Query workflow runs filtered by the PR's head SHA
	path := fmt.Sprintf("repos/%s/%s/actions/runs?head_sha=%s&per_page=50", owner, repo, headSHA)

	resp, err := client.Request(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query workflow runs for PR: %w", err)
	}
	defer resp.Body.Close()

	rateLimit := parseRateLimitHeaders(resp.Header)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, rateLimit, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var result WorkflowRunsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, rateLimit, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Debug("Fetched workflow runs for PR", "repo", fmt.Sprintf("%s/%s", owner, repo), "pr", prNumber, "count", len(result.WorkflowRuns))
	return result.WorkflowRuns, rateLimit, nil
}

// GetWorkflowRunStatus fetches the current status of a specific workflow run
func GetWorkflowRunStatus(owner, repo string, runID int64) (*WorkflowRun, *RateLimitInfo, error) {
	client, err := gh.DefaultRESTClient()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	path := fmt.Sprintf("repos/%s/%s/actions/runs/%d", owner, repo, runID)

	resp, err := client.Request(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get workflow run status: %w", err)
	}
	defer resp.Body.Close()

	rateLimit := parseRateLimitHeaders(resp.Header)

	if resp.StatusCode == http.StatusNotFound {
		return nil, rateLimit, fmt.Errorf("workflow run %d not found", runID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, rateLimit, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var run WorkflowRun
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return nil, rateLimit, fmt.Errorf("failed to decode response: %w", err)
	}

	return &run, rateLimit, nil
}

// parseRateLimitHeaders extracts rate limit information from response headers
func parseRateLimitHeaders(headers http.Header) *RateLimitInfo {
	remaining, _ := strconv.Atoi(headers.Get("X-RateLimit-Remaining"))
	limit, _ := strconv.Atoi(headers.Get("X-RateLimit-Limit"))
	resetUnix, _ := strconv.ParseInt(headers.Get("X-RateLimit-Reset"), 10, 64)

	return &RateLimitInfo{
		Remaining: remaining,
		Limit:     limit,
		ResetAt:   time.Unix(resetUnix, 0),
	}
}

// ShouldBackoff determines if we should back off due to rate limiting
// Returns true if remaining requests are below 20% of the limit
func (r *RateLimitInfo) ShouldBackoff() bool {
	if r == nil || r.Limit == 0 {
		return false
	}
	threshold := r.Limit / 5 // 20% threshold
	return r.Remaining < threshold
}

// GetBackoffDuration calculates how long to wait before retrying
// Uses exponential backoff based on attempt number
func GetBackoffDuration(attempt int) time.Duration {
	// Exponential backoff: 2^attempt seconds, capped at 5 minutes
	duration := time.Duration(1<<uint(attempt)) * time.Second
	maxDuration := 5 * time.Minute
	if duration > maxDuration {
		return maxDuration
	}
	return duration
}

// BatchWorkflowRequest represents a request for workflow runs for a specific item
type BatchWorkflowRequest struct {
	Owner   string
	Repo    string
	Number  int    // Issue or PR number
	HeadSHA string // For PRs, the head commit SHA
}

// BatchWorkflowResult contains workflow runs and any error for a specific request
type BatchWorkflowResult struct {
	Request      BatchWorkflowRequest
	WorkflowRuns []WorkflowRun
	Error        error
}

// BatchQueryWorkflowRuns queries workflow runs for multiple items in sequence
// Returns results for each request, continuing on individual errors
// Rate limit info is from the last successful request
func BatchQueryWorkflowRuns(requests []BatchWorkflowRequest) ([]BatchWorkflowResult, *RateLimitInfo, error) {
	if len(requests) == 0 {
		return []BatchWorkflowResult{}, nil, nil
	}

	results := make([]BatchWorkflowResult, 0, len(requests))
	var lastRateLimit *RateLimitInfo

	for _, req := range requests {
		var runs []WorkflowRun
		var rateLimit *RateLimitInfo
		var err error

		if req.HeadSHA != "" {
			// For PRs, query by head SHA
			runs, rateLimit, err = QueryWorkflowRunsForPR(req.Owner, req.Repo, req.Number, req.HeadSHA)
		} else {
			// For issues or general queries, get recent runs
			runs, rateLimit, err = QueryWorkflowRuns(req.Owner, req.Repo, 50)
		}

		results = append(results, BatchWorkflowResult{
			Request:      req,
			WorkflowRuns: runs,
			Error:        err,
		})

		if rateLimit != nil {
			lastRateLimit = rateLimit
			// Check if we should back off
			if rateLimit.ShouldBackoff() {
				log.Warn("Approaching rate limit, stopping batch query early",
					"remaining", rateLimit.Remaining,
					"limit", rateLimit.Limit,
					"processed", len(results),
					"total", len(requests))
				break
			}
		}
	}

	return results, lastRateLimit, nil
}
