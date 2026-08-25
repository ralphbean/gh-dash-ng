package data

import (
	"fmt"
	"sync"
	"time"
)

// FullsendCache provides two-layer caching for fullsend workflow monitoring:
// 1. Workflow run cache: completed runs (never expire within session)
// 2. Issue/PR check cache: last check time + active runs (5-minute TTL)
type FullsendCache struct {
	mu sync.RWMutex

	// Layer 1: Workflow run cache (run ID → terminal status)
	// Completed runs are cached indefinitely (no TTL)
	workflowRuns map[int64]WorkflowRunStatus

	// Layer 2: Issue/PR check cache (repo+number → check metadata)
	// Has 5-minute TTL for re-checking new dispatches
	itemChecks map[string]ItemCheckCache
}

// WorkflowRunStatus represents a workflow run's terminal state
type WorkflowRunStatus struct {
	Status      string    // "completed", "failed", "cancelled"
	CompletedAt time.Time // When it reached terminal state
}

// ItemCheckCache tracks when we last checked an issue/PR and what runs are active
type ItemCheckCache struct {
	LastChecked  time.Time   // Last time we queried this item
	ActiveRunIDs []int64     // Workflow run IDs currently active
	CachedAt     time.Time   // When this cache entry was created
}

// Cache TTL Configuration:
//
// itemCheckTTL (5 minutes): How long to cache the "last checked" timestamp for an
// issue/PR before querying GitHub again for new workflow runs. This balances freshness
// (detecting new fullsend agent dispatches) against API rate limit conservation.
// 5 minutes ensures new dispatches are discovered reasonably quickly while avoiding
// excessive API calls.
//
// Workflow run cache (no TTL): Completed workflow runs never change state, so they are
// cached indefinitely (for the session lifetime). This eliminates redundant API calls
// for runs we've already seen complete.
const itemCheckTTL = 5 * time.Minute

// NewFullsendCache creates a new cache instance
func NewFullsendCache() *FullsendCache {
	return &FullsendCache{
		workflowRuns: make(map[int64]WorkflowRunStatus),
		itemChecks:   make(map[string]ItemCheckCache),
	}
}

// GetWorkflowRunStatus retrieves cached workflow run status
// Returns the status and true if found, empty string and false otherwise
func (c *FullsendCache) GetWorkflowRunStatus(runID int64) (WorkflowRunStatus, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	status, ok := c.workflowRuns[runID]
	return status, ok
}

// SetWorkflowRunStatus caches a workflow run's terminal status
// Only terminal states (completed, failed, cancelled) should be cached
func (c *FullsendCache) SetWorkflowRunStatus(runID int64, status string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.workflowRuns[runID] = WorkflowRunStatus{
		Status:      status,
		CompletedAt: time.Now(),
	}
}

// GetItemCheck retrieves cached check metadata for an issue/PR
// Returns the cache entry and true if found and not expired, empty entry and false otherwise
func (c *FullsendCache) GetItemCheck(repo string, number int) (ItemCheckCache, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := makeItemKey(repo, number)
	entry, ok := c.itemChecks[key]
	if !ok {
		return ItemCheckCache{}, false
	}

	// Check if TTL has expired
	if time.Since(entry.CachedAt) > itemCheckTTL {
		return ItemCheckCache{}, false
	}

	return entry, true
}

// SetItemCheck caches check metadata for an issue/PR
func (c *FullsendCache) SetItemCheck(repo string, number int, activeRunIDs []int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := makeItemKey(repo, number)
	c.itemChecks[key] = ItemCheckCache{
		LastChecked:  time.Now(),
		ActiveRunIDs: activeRunIDs,
		CachedAt:     time.Now(),
	}
}

// EvictExpiredItemChecks removes expired entries from the item check cache
// This should be called periodically to prevent unbounded memory growth
func (c *FullsendCache) EvictExpiredItemChecks() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	evicted := 0
	for key, entry := range c.itemChecks {
		if time.Since(entry.CachedAt) > itemCheckTTL {
			delete(c.itemChecks, key)
			evicted++
		}
	}
	return evicted
}

// makeItemKey creates a cache key for an issue/PR
func makeItemKey(repo string, number int) string {
	return fmt.Sprintf("%s#%d", repo, number)
}
