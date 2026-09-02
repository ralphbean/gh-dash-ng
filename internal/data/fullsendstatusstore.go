package data

import (
	"fmt"
	"sync"

	"charm.land/log/v2"
)

// FullsendStatusStore tracks fullsend agent status for issues and PRs separately from GraphQL data
type FullsendStatusStore struct {
	mu     sync.RWMutex
	status map[string]FullsendStatus // key: "owner/repo#number"
}

var (
	globalFullsendStatusStore *FullsendStatusStore
	storeOnce                 sync.Once
)

// GetFullsendStatusStore returns the global singleton store
func GetFullsendStatusStore() *FullsendStatusStore {
	storeOnce.Do(func() {
		globalFullsendStatusStore = &FullsendStatusStore{
			status: make(map[string]FullsendStatus),
		}
	})
	return globalFullsendStatusStore
}

// Set stores the fullsend status for a given repo and number
func (s *FullsendStatusStore) Set(owner, repo string, number int, status FullsendStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := fmt.Sprintf("%s/%s#%d", owner, repo, number)
	log.Debug("FullsendStatusStore.Set",
		"key", key,
		"owner", owner,
		"repo", repo,
		"number", number,
		"active_agents", len(status.ActiveAgents))
	s.status[key] = status
}

// Get retrieves the fullsend status for a given repo and number
func (s *FullsendStatusStore) Get(owner, repo string, number int) FullsendStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := fmt.Sprintf("%s/%s#%d", owner, repo, number)
	status := s.status[key]
	log.Debug("FullsendStatusStore.Get",
		"key", key,
		"owner", owner,
		"repo", repo,
		"number", number,
		"active_agents", len(status.ActiveAgents))
	return status
}

// Clear removes all stored status
func (s *FullsendStatusStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = make(map[string]FullsendStatus)
}
