package data

import (
	"testing"
	"time"

	gh "github.com/cli/go-gh/v2/pkg/api"
	"github.com/stretchr/testify/require"
)

func TestClearEnrichmentCache(t *testing.T) {
	// Save original state
	originalCachedClient := cachedClient
	defer func() {
		cachedClient = originalCachedClient
	}()

	t.Run("clears nil cache without panic", func(t *testing.T) {
		cachedClient = nil
		require.True(t, IsEnrichmentCacheCleared(), "cache should be cleared initially")

		ClearEnrichmentCache()
		require.True(t, IsEnrichmentCacheCleared(), "cache should remain cleared")
	})

	t.Run("clears non-nil cache", func(t *testing.T) {
		// Simulate having a cached client (we use an empty struct pointer
		// since we can't create a real GraphQL client without credentials)
		cachedClient = &gh.GraphQLClient{}
		require.False(
			t,
			IsEnrichmentCacheCleared(),
			"cache should not be cleared when client is set",
		)

		ClearEnrichmentCache()
		require.True(
			t,
			IsEnrichmentCacheCleared(),
			"cache should be cleared after ClearEnrichmentCache",
		)
	})
}

func TestIsEnrichmentCacheCleared(t *testing.T) {
	// Save original state
	originalCachedClient := cachedClient
	defer func() {
		cachedClient = originalCachedClient
	}()

	t.Run("returns true when cache is nil", func(t *testing.T) {
		cachedClient = nil
		require.True(t, IsEnrichmentCacheCleared())
	})

	t.Run("returns false when cache is set", func(t *testing.T) {
		cachedClient = &gh.GraphQLClient{}
		require.False(t, IsEnrichmentCacheCleared())
	})
}

func TestUnresolvedThreadsCount(t *testing.T) {
	newNodes := func(resolved ...bool) []struct{ IsResolved bool } {
		nodes := make([]struct{ IsResolved bool }, len(resolved))
		for i, r := range resolved {
			nodes[i] = struct{ IsResolved bool }{IsResolved: r}
		}
		return nodes
	}

	tests := []struct {
		name  string
		nodes []struct{ IsResolved bool }
		want  int
	}{
		{
			name:  "no threads",
			nodes: newNodes(),
			want:  0,
		},
		{
			name:  "all resolved",
			nodes: newNodes(true, true, true),
			want:  0,
		},
		{
			name:  "some unresolved",
			nodes: newNodes(false, true, false),
			want:  2,
		},
		{
			name:  "all unresolved",
			nodes: newNodes(false, false),
			want:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := PullRequestData{
				ReviewThreads: ReviewThreadsWithResolution{Nodes: tt.nodes},
			}
			require.Equal(t, tt.want, pr.UnresolvedThreadsCount())
		})
	}
}

func TestReviewThreadWithComments_Fields(t *testing.T) {
	// FetchReviewThreads itself talks to GitHub's GraphQL API and isn't
	// unit-tested here (matching FetchPullRequest/FetchPullRequests, which
	// have no direct unit tests either); the queue-building logic that
	// excludes IsResolved: true threads is exercised where it lives, in the
	// prview package. This just pins down the struct's new fields.
	thread := ReviewThreadWithComments{
		Id:               "thread-1",
		IsResolved:       true,
		ViewerCanReply:   true,
		ViewerCanResolve: false,
		Path:             "main.go",
		Line:             42,
		Comments: ReviewComments{
			Nodes: []ReviewComment{
				{Body: "looks good", DiffHunk: "@@ -1,2 +1,2 @@\n-old\n+new"},
			},
		},
	}

	require.True(t, thread.IsResolved)
	require.True(t, thread.ViewerCanReply)
	require.False(t, thread.ViewerCanResolve)
	require.Equal(t, "main.go", thread.Path)
	require.Equal(t, "@@ -1,2 +1,2 @@\n-old\n+new", thread.Comments.Nodes[0].DiffHunk)
}

func TestComputeReviewStatus(t *testing.T) {
	tests := []struct {
		name    string
		reviews []ReviewSummary
		want    string
	}{
		{
			name:    "empty input",
			reviews: nil,
			want:    "",
		},
		{
			name: "single approval",
			reviews: []ReviewSummary{
				{Login: "alice", State: "APPROVED"},
			},
			want: "APPROVED",
		},
		{
			name: "single changes requested",
			reviews: []ReviewSummary{
				{Login: "alice", State: "CHANGES_REQUESTED"},
			},
			want: "CHANGES_REQUESTED",
		},
		{
			name: "single comment",
			reviews: []ReviewSummary{
				{Login: "alice", State: "COMMENTED"},
			},
			want: "COMMENTED",
		},
		{
			name: "approval then comment stays approved",
			reviews: []ReviewSummary{
				{Login: "alice", State: "APPROVED"},
				{Login: "alice", State: "COMMENTED"},
			},
			want: "APPROVED",
		},
		{
			name: "changes requested then comment stays changes requested",
			reviews: []ReviewSummary{
				{Login: "alice", State: "CHANGES_REQUESTED"},
				{Login: "alice", State: "COMMENTED"},
			},
			want: "CHANGES_REQUESTED",
		},
		{
			name: "multiple authors with mixed states",
			reviews: []ReviewSummary{
				{Login: "alice", State: "APPROVED"},
				{Login: "bob", State: "CHANGES_REQUESTED"},
				{Login: "carol", State: "COMMENTED"},
			},
			want: "CHANGES_REQUESTED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ComputeReviewStatus(tt.reviews))
		})
	}
}

func TestPartitionByBotAuthor(t *testing.T) {
	tests := []struct {
		name      string
		reviews   []ReviewSummary
		wantHuman []ReviewSummary
		wantBot   []ReviewSummary
	}{
		{
			name:      "empty",
			reviews:   nil,
			wantHuman: nil,
			wantBot:   nil,
		},
		{
			name: "all human",
			reviews: []ReviewSummary{
				{Login: "alice", Typename: "User", State: "APPROVED"},
				{Login: "bob", Typename: "User", State: "COMMENTED"},
			},
			wantHuman: []ReviewSummary{
				{Login: "alice", Typename: "User", State: "APPROVED"},
				{Login: "bob", Typename: "User", State: "COMMENTED"},
			},
			wantBot: nil,
		},
		{
			name: "all bot",
			reviews: []ReviewSummary{
				{Login: "dependabot", Typename: "Bot", State: "APPROVED"},
			},
			wantHuman: nil,
			wantBot: []ReviewSummary{
				{Login: "dependabot", Typename: "Bot", State: "APPROVED"},
			},
		},
		{
			name: "mixed",
			reviews: []ReviewSummary{
				{Login: "alice", Typename: "User", State: "APPROVED"},
				{Login: "dependabot", Typename: "Bot", State: "COMMENTED"},
			},
			wantHuman: []ReviewSummary{
				{Login: "alice", Typename: "User", State: "APPROVED"},
			},
			wantBot: []ReviewSummary{
				{Login: "dependabot", Typename: "Bot", State: "COMMENTED"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			human, bot := PartitionByBotAuthor(tt.reviews)
			require.Equal(t, tt.wantHuman, human)
			require.Equal(t, tt.wantBot, bot)
		})
	}
}

func TestSetClient(t *testing.T) {
	// Save original state
	originalClient := client
	originalCachedClient := cachedClient
	defer func() {
		client = originalClient
		cachedClient = originalCachedClient
	}()

	t.Run("sets both client and cachedClient", func(t *testing.T) {
		client = nil
		cachedClient = nil

		// SetClient with nil should set both to nil
		SetClient(nil)
		require.Nil(t, client)
		require.True(t, IsEnrichmentCacheCleared())
	})
}

func TestMakeMergedPRQuery(t *testing.T) {
	cutoff := time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		filters string
		want    string
	}{
		{
			name:    "replaces is:open with is:merged and adds date",
			filters: "is:open author:@me",
			want:    "author:@me is:merged merged:>=2026-08-19",
		},
		{
			name:    "strips existing is:closed",
			filters: "is:closed author:@me",
			want:    "author:@me is:merged merged:>=2026-08-19",
		},
		{
			name:    "strips existing merged: qualifier",
			filters: "is:open merged:>=2020-01-01 author:@me",
			want:    "author:@me is:merged merged:>=2026-08-19",
		},
		{
			name:    "handles no is: qualifier",
			filters: "author:@me review-requested:@me",
			want:    "author:@me review-requested:@me is:merged merged:>=2026-08-19",
		},
		{
			name:    "case insensitive is:Open",
			filters: "Is:Open author:@me",
			want:    "author:@me is:merged merged:>=2026-08-19",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeMergedPRQuery(tt.filters, cutoff)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMergePullRequestResults(t *testing.T) {
	t.Run("no duplicates", func(t *testing.T) {
		primary := []PullRequestData{
			{Number: 1, Title: "PR 1"},
			{Number: 2, Title: "PR 2"},
		}
		merged := []PullRequestData{
			{Number: 3, Title: "PR 3"},
		}
		result := MergePullRequestResults(primary, merged)
		require.Len(t, result, 3)
	})

	t.Run("duplicates kept from primary", func(t *testing.T) {
		primary := []PullRequestData{
			{Number: 1, Title: "Primary PR 1"},
		}
		merged := []PullRequestData{
			{Number: 1, Title: "Merged PR 1"},
			{Number: 2, Title: "Merged PR 2"},
		}
		result := MergePullRequestResults(primary, merged)
		require.Len(t, result, 2)
		require.Equal(t, "Primary PR 1", result[0].Title)
		require.Equal(t, "Merged PR 2", result[1].Title)
	})

	t.Run("empty primary", func(t *testing.T) {
		merged := []PullRequestData{
			{Number: 1, Title: "PR 1"},
		}
		result := MergePullRequestResults(nil, merged)
		require.Len(t, result, 1)
		require.Equal(t, "PR 1", result[0].Title)
	})

	t.Run("empty merged", func(t *testing.T) {
		primary := []PullRequestData{
			{Number: 1, Title: "PR 1"},
		}
		result := MergePullRequestResults(primary, nil)
		require.Len(t, result, 1)
	})

	t.Run("both empty", func(t *testing.T) {
		result := MergePullRequestResults(nil, nil)
		require.Empty(t, result)
	})
}
