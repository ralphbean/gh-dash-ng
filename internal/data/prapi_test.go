package data

import (
	"io"
	"net/http"
	"strings"
	"testing"

	gh "github.com/cli/go-gh/v2/pkg/api"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestFetchPullRequestsSelectsStatusContextQueryOnlyWhenConfigured(t *testing.T) {
	originalClient := client
	defer func() { client = originalClient }()

	var requestBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		requestBody = string(body)
		response := `{"data":{"search":{"nodes":[],"issueCount":0,` +
			`"pageInfo":{"hasNextPage":false,"startCursor":"","endCursor":""}}}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(response)),
			Header:     make(http.Header),
		}, nil
	})
	testClient, err := gh.NewGraphQLClient(gh.ClientOptions{
		Host:      "example.test",
		AuthToken: "test",
		Transport: transport,
	})
	require.NoError(t, err)
	client = testClient

	_, err = FetchPullRequests("author:@me", 20, nil, nil)
	require.NoError(t, err)
	require.NotContains(t, requestBody, "filteredCommits")
	require.NotContains(t, requestBody, "contexts(last: 100)")

	_, err = FetchPullRequests("author:@me", 20, nil, []string{"*fullsend/dispatch*"})
	require.NoError(t, err)
	require.Contains(t, requestBody, "filteredCommits: commits(last: 1)")
	require.Contains(t, requestBody, "contexts(last: 100)")
}

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
