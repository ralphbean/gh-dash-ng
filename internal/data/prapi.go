package data

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"charm.land/log/v2"
	gh "github.com/cli/go-gh/v2/pkg/api"
	graphql "github.com/cli/shurcooL-graphql"
	checks "github.com/dlvhdr/x/gh-checks"
	"github.com/shurcooL/githubv4"

	"github.com/dlvhdr/gh-dash/v4/internal/config"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/theme"
)

type SuggestedReviewer struct {
	IsAuthor    bool
	IsCommenter bool
	Reviewer    struct {
		Login string
	}
}

type EnrichedPullRequestData struct {
	Url     string
	Number  int
	Title   string
	Body    string
	State   string
	IsDraft bool
	Author  struct {
		Login string
	}
	AuthorAssociation string
	UpdatedAt         time.Time
	CreatedAt         time.Time
	Mergeable         string
	ReviewDecision    string
	Additions         int
	Deletions         int
	HeadRefName       string
	BaseRefName       string
	HeadRepository    struct {
		Name string
	}
	HeadRef struct {
		Name string
	}
	Labels             PRLabels  `graphql:"labels(first: 6)"`
	Assignees          Assignees `graphql:"assignees(first: 3)"`
	Repository         Repository
	Commits            LastCommitWithStatusChecks `graphql:"commits(last: 1)"`
	AllCommits         AllCommits                 `graphql:"allCommits: commits(last: 100)"`
	Comments           CommentsWithBody           `graphql:"comments(last: 50, orderBy: { field: UPDATED_AT, direction: DESC })"`
	ReviewThreads      ReviewThreadsWithComments  `graphql:"reviewThreads(last: 50)"`
	ReviewRequests     ReviewRequests             `graphql:"reviewRequests(last: 100)"`
	Reviews            Reviews                    `graphql:"reviews(last: 100)"`
	SuggestedReviewers []SuggestedReviewer
	Files              ChangedFiles `graphql:"files(first: 20)"`
}

type PullRequestData struct {
	Number int
	Title  string
	Author struct {
		Login string
	}
	AuthorAssociation string
	UpdatedAt         time.Time
	CreatedAt         time.Time
	Url               string
	State             string
	Mergeable         string
	ReviewDecision    string
	Additions         int
	Deletions         int
	HeadRefName       string
	HeadRefOid        string // SHA of the head commit
	BaseRefName       string
	HeadRepository    struct {
		Name string
	}
	HeadRef struct {
		Name string
	}
	Repository       Repository
	Assignees        Assignees                   `graphql:"assignees(first: 3)"`
	ReviewThreads    ReviewThreadsWithResolution `graphql:"reviewThreads(last: 100)"`
	Reviews          ReviewsWithAuthorType       `graphql:"reviews(last: 100)"`
	ReviewRequests   ReviewRequestsNumber        `graphql:"reviewRequests"`
	IsDraft          bool
	IsInMergeQueue   bool
	Commits          LastCommitStatus `graphql:"commits(last: 1)"`
	Labels           PRLabels         `graphql:"labels(first: 6)"`
	MergeStateStatus MergeStateStatus `graphql:"mergeStateStatus"`
}

type LastCommitStatus struct {
	Nodes []struct {
		Commit struct {
			StatusCheckRollup struct {
				State graphql.String
			}
		}
	}
}

type CheckRun struct {
	Name       graphql.String
	Status     graphql.String
	Conclusion checks.CheckRunState
	CheckSuite struct {
		Creator struct {
			Login graphql.String
		}
		WorkflowRun struct {
			Workflow struct {
				Name graphql.String
			}
		}
	}
}

type StatusContext struct {
	Context graphql.String
	State   graphql.String
	Creator struct {
		Login graphql.String
	}
}

type CheckSuiteNode struct {
	Status     graphql.String
	Conclusion graphql.String

	App struct {
		Name graphql.String
	}

	WorkflowRun struct {
		Workflow struct {
			Name graphql.String
		}
	}
}

type CheckSuites struct {
	TotalCount graphql.Int
	Nodes      []CheckSuiteNode
}

type StatusCheckRollupStats struct {
	State    checks.CommitState
	Contexts struct {
		TotalCount                 graphql.Int
		CheckRunCount              graphql.Int
		CheckRunCountsByState      []ContextCountByState
		StatusContextCount         graphql.Int
		StatusContextCountsByState []ContextCountByState
	} `graphql:"contexts(last: 1)"`
}

type AllCommits struct {
	Nodes []struct {
		Commit struct {
			AbbreviatedOid  string
			CommittedDate   time.Time
			MessageHeadline string
			Author          struct {
				Name string
				User struct {
					Login string
				}
			}
			StatusCheckRollup StatusCheckRollupStats
		}
	}
}

type LastCommitWithStatusChecks struct {
	Nodes []struct {
		Commit struct {
			Deployments struct {
				Nodes []struct {
					Task        graphql.String
					Description graphql.String
				}
			} `graphql:"deployments(last: 10)"`
			CommitUrl         graphql.String
			StatusCheckRollup struct {
				State    graphql.String
				Contexts struct {
					TotalCount                 graphql.Int
					CheckRunCount              graphql.Int
					CheckRunCountsByState      []ContextCountByState
					StatusContextCount         graphql.Int
					StatusContextCountsByState []ContextCountByState
					Nodes                      []struct {
						Typename      graphql.String `graphql:"__typename"`
						CheckRun      CheckRun       `graphql:"... on CheckRun"`
						StatusContext StatusContext  `graphql:"... on StatusContext"`
					}
				} `graphql:"contexts(last: 100)"`
			}
			// CheckSuites are fetched separately from StatusCheckRollup because
			// workflows awaiting approval (conclusion ACTION_REQUIRED) and workflows
			// still queued have no CheckRun objects yet, so they don’t appear in
			// StatusCheckRollup.contexts.
			CheckSuites CheckSuites `graphql:"checkSuites(last: 20)"`
		}
	}
	TotalCount int
}

type CommentsWithBody struct {
	TotalCount graphql.Int
	Nodes      []Comment
}

type ContextCountByState = struct {
	Count graphql.Int
	State checks.CheckRunState
}

type Commits struct {
	Nodes []struct {
		Commit struct {
			Deployments struct {
				Nodes []struct {
					Task        graphql.String
					Description graphql.String
				}
			} `graphql:"deployments(last: 10)"`
			CommitUrl         graphql.String
			StatusCheckRollup struct {
				State graphql.String
			}
		}
	}
	TotalCount int
}

type Comment struct {
	Author struct {
		Login string
	}
	Body      string
	UpdatedAt time.Time
}

type ReviewComment struct {
	Author struct {
		Login string
	}
	Body      string
	UpdatedAt time.Time
	StartLine int
	Line      int
	DiffHunk  string
}

type ReviewComments struct {
	Nodes      []ReviewComment
	TotalCount int
}

type ReviewThreadsWithResolution struct {
	TotalCount int
	Nodes      []struct {
		IsResolved bool
	}
}

type Review struct {
	Author struct {
		Login    string
		Typename graphql.String `graphql:"__typename"`
	}
	Body      string
	State     string
	UpdatedAt time.Time
}

type ReviewsNumber struct {
	TotalCount int
}

type Reviews struct {
	TotalCount int
	Nodes      []Review
}

// ReviewsWithAuthorType is a lighter-weight alternative to Reviews for
// contexts (like the PRs list query) that only need each review's state and
// author account type - not the review body or timestamp.
type ReviewsWithAuthorType struct {
	TotalCount int
	Nodes      []struct {
		State  string
		Author struct {
			Login    string
			Typename graphql.String `graphql:"__typename"`
		}
	}
}

// ReviewSummary is a minimal, source-agnostic view of a single review used
// to compute review status independently of which query fetched it.
type ReviewSummary struct {
	Login    string
	Typename string
	State    string
}

func (r Review) ToReviewSummary() ReviewSummary {
	return ReviewSummary{
		Login:    r.Author.Login,
		Typename: string(r.Author.Typename),
		State:    r.State,
	}
}

func ReviewSummariesFromReviews(reviews []Review) []ReviewSummary {
	summaries := make([]ReviewSummary, 0, len(reviews))
	for _, review := range reviews {
		summaries = append(summaries, review.ToReviewSummary())
	}
	return summaries
}

func ReviewSummariesFromReviewsWithAuthorType(reviews ReviewsWithAuthorType) []ReviewSummary {
	summaries := make([]ReviewSummary, 0, len(reviews.Nodes))
	for _, node := range reviews.Nodes {
		summaries = append(summaries, ReviewSummary{
			Login:    node.Author.Login,
			Typename: string(node.Author.Typename),
			State:    node.State,
		})
	}
	return summaries
}

// PartitionByBotAuthor splits reviews into two slices: those authored by a
// bot (GitHub App) account, and everything else (human users, teams,
// mannequins, etc.).
func PartitionByBotAuthor(reviews []ReviewSummary) (human, bot []ReviewSummary) {
	for _, review := range reviews {
		if review.Typename == "Bot" {
			bot = append(bot, review)
		} else {
			human = append(human, review)
		}
	}
	return human, bot
}

// ComputeReviewStatus determines the aggregate review status for a set of
// reviews: the most decisive state per author (approved/changes-requested
// take priority over a later comment from the same author) is combined
// across authors, with changes-requested beating approved beating commented.
// Returns "APPROVED", "CHANGES_REQUESTED", "COMMENTED", or "" (no actionable
// reviews).
func ComputeReviewStatus(reviews []ReviewSummary) string {
	perAuthor := make(map[string]string)
	for _, review := range reviews {
		if review.State != "APPROVED" &&
			review.State != "CHANGES_REQUESTED" &&
			review.State != "COMMENTED" {
			continue
		}
		existing := perAuthor[review.Login]
		// Don't let a later COMMENTED review from the same author downgrade
		// an earlier APPROVED or CHANGES_REQUESTED.
		if review.State == "COMMENTED" &&
			(existing == "APPROVED" || existing == "CHANGES_REQUESTED") {
			continue
		}
		perAuthor[review.Login] = review.State
	}

	sawApproved := false
	sawCommented := false
	for _, state := range perAuthor {
		switch state {
		case "CHANGES_REQUESTED":
			return "CHANGES_REQUESTED"
		case "APPROVED":
			sawApproved = true
		case "COMMENTED":
			sawCommented = true
		}
	}
	if sawApproved {
		return "APPROVED"
	}
	if sawCommented {
		return "COMMENTED"
	}
	return ""
}

type ReviewThreadWithComments struct {
	Id               string
	IsOutdated       bool
	IsResolved       bool
	ViewerCanReply   bool
	ViewerCanResolve bool
	OriginalLine     int
	StartLine        int
	Line             int
	Path             string
	Comments         ReviewComments `graphql:"comments(first: 20)"`
}

type ReviewThreadsWithComments struct {
	Nodes []ReviewThreadWithComments
}

type ChangedFile struct {
	Additions  int
	Deletions  int
	Path       string
	ChangeType string
}

type ChangedFiles struct {
	TotalCount int
	Nodes      []ChangedFile
}

type RequestedReviewerUser struct {
	Login string `graphql:"login"`
}

type RequestedReviewerTeam struct {
	Slug string `graphql:"slug"`
	Name string `graphql:"name"`
}

type RequestedReviewerBot struct {
	Login string `graphql:"login"`
}

type RequestedReviewerMannequin struct {
	Login string `graphql:"login"`
}

type ReviewRequestNode struct {
	AsCodeOwner       bool `graphql:"asCodeOwner"`
	RequestedReviewer struct {
		User      RequestedReviewerUser      `graphql:"... on User"`
		Team      RequestedReviewerTeam      `graphql:"... on Team"`
		Bot       RequestedReviewerBot       `graphql:"... on Bot"`
		Mannequin RequestedReviewerMannequin `graphql:"... on Mannequin"`
	} `graphql:"requestedReviewer"`
}

type ReviewRequestsNumber struct {
	TotalCount int
}

type ReviewRequests struct {
	TotalCount int
	Nodes      []ReviewRequestNode
}

func (r ReviewRequestNode) GetReviewerDisplayName() string {
	if r.RequestedReviewer.User.Login != "" {
		return r.RequestedReviewer.User.Login
	}
	if r.RequestedReviewer.Team.Slug != "" {
		return r.RequestedReviewer.Team.Slug
	}
	if r.RequestedReviewer.Bot.Login != "" {
		return r.RequestedReviewer.Bot.Login
	}
	if r.RequestedReviewer.Mannequin.Login != "" {
		return r.RequestedReviewer.Mannequin.Login
	}
	return ""
}

func (r ReviewRequestNode) GetReviewerType() string {
	if r.RequestedReviewer.User.Login != "" {
		return "User"
	}
	if r.RequestedReviewer.Team.Slug != "" {
		return "Team"
	}
	if r.RequestedReviewer.Bot.Login != "" {
		return "Bot"
	}
	if r.RequestedReviewer.Mannequin.Login != "" {
		return "Mannequin"
	}
	return ""
}

func (r ReviewRequestNode) IsTeam() bool {
	return r.RequestedReviewer.Team.Slug != ""
}

type PRLabel struct {
	Color string
	Name  string
}

type PRLabels struct {
	Nodes []Label
}

type MergeStateStatus string

type PageInfo struct {
	HasNextPage bool
	StartCursor string
	EndCursor   string
}

func (data PullRequestData) GetAuthor(theme theme.Theme, showAuthorIcon bool) string {
	author := data.Author.Login
	if showAuthorIcon {
		author += fmt.Sprintf(" %s", GetAuthorRoleIcon(data.AuthorAssociation, theme))
	}
	return author
}

func (data PullRequestData) GetTitle() string {
	return data.Title
}

func (data PullRequestData) GetRepoNameWithOwner() string {
	return data.Repository.NameWithOwner
}

func (data PullRequestData) GetRepoNameAndOwner() (owner, repoName string) {
	return data.Repository.Owner.Login, data.Repository.Name
}

func (data PullRequestData) GetNumber() int {
	return data.Number
}

func (data PullRequestData) GetUrl() string {
	return data.Url
}

func (data PullRequestData) GetUpdatedAt() time.Time {
	return data.UpdatedAt
}

func (data PullRequestData) GetCreatedAt() time.Time {
	return data.CreatedAt
}

// UnresolvedThreadsCount returns the number of review threads that have not
// been resolved. Only the last 100 threads are fetched, so PRs with more
// review threads than that will undercount.
func (data PullRequestData) UnresolvedThreadsCount() int {
	count := 0
	for _, node := range data.ReviewThreads.Nodes {
		if !node.IsResolved {
			count++
		}
	}
	return count
}

// ToPullRequestData converts EnrichedPullRequestData to PullRequestData
// This is useful when we fetch a single PR and need basic PR fields
func (e EnrichedPullRequestData) ToPullRequestData() PullRequestData {
	return PullRequestData{
		Number:            e.Number,
		Title:             e.Title,
		Author:            e.Author,
		AuthorAssociation: e.AuthorAssociation,
		UpdatedAt:         e.UpdatedAt,
		CreatedAt:         e.CreatedAt,
		Url:               e.Url,
		State:             e.State,
		Mergeable:         e.Mergeable,
		ReviewDecision:    e.ReviewDecision,
		Additions:         e.Additions,
		Deletions:         e.Deletions,
		HeadRefName:       e.HeadRefName,
		BaseRefName:       e.BaseRefName,
		HeadRepository:    e.HeadRepository,
		HeadRef:           e.HeadRef,
		Repository:        e.Repository,
		Assignees:         e.Assignees,
		IsDraft:           e.IsDraft,
		Labels:            e.Labels,
		// Note: ReviewThreads, Reviews, ReviewRequests, Commits
		// have different types in EnrichedPullRequestData vs PullRequestData
		// We leave them as zero values since the enriched data will be used instead
	}
}

func makePullRequestsQuery(query string) string {
	return fmt.Sprintf("is:pr archived:false %s sort:updated", query)
}

type PullRequestsResponse struct {
	Prs        []PullRequestData
	TotalCount int
	PageInfo   PageInfo
}

var (
	client       *gh.GraphQLClient
	cachedClient *gh.GraphQLClient
)

func SetClient(c *gh.GraphQLClient) {
	client = c
	cachedClient = c
}

// ClearEnrichmentCache clears the cached GraphQL client used for fetching
// enriched PR/Issue data. Call this when refreshing to ensure fresh data.
func ClearEnrichmentCache() {
	cachedClient = nil
}

// IsEnrichmentCacheCleared returns true if the enrichment cache is cleared.
// This is primarily for testing purposes.
func IsEnrichmentCacheCleared() bool {
	return cachedClient == nil
}

func FetchPullRequests(query string, limit int, pageInfo *PageInfo) (PullRequestsResponse, error) {
	var err error
	if client == nil {
		if config.IsFeatureEnabled(config.FF_MOCK_DATA) {
			log.Info("using mock data", "server", "https://localhost:3000")
			http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
			client, err = gh.NewGraphQLClient(
				gh.ClientOptions{Host: "localhost:3000", AuthToken: "fake-token"},
			)
		} else {
			level := os.Getenv("LOG_LEVEL")
			opts := gh.ClientOptions{}
			if level == "debug" {
				logger := NewHTTPLogger(0)
				opts.Log = &logger
				opts.LogVerboseHTTP = true
				opts.LogColorize = true
			}
			client, err = gh.NewGraphQLClient(opts)
		}
	}

	if err != nil {
		return PullRequestsResponse{}, err
	}

	var queryResult struct {
		Search struct {
			Nodes []struct {
				PullRequest PullRequestData `graphql:"... on PullRequest"`
			}
			IssueCount int
			PageInfo   PageInfo
		} `graphql:"search(type: ISSUE, first: $limit, after: $endCursor, query: $query)"`
	}
	var endCursor *string
	if pageInfo != nil {
		endCursor = &pageInfo.EndCursor
	}
	variables := map[string]any{
		"query":     graphql.String(makePullRequestsQuery(query)),
		"limit":     graphql.Int(limit),
		"endCursor": (*graphql.String)(endCursor),
	}
	log.Debug("Fetching PRs", "query", query, "limit", limit, "endCursor", endCursor)
	err = client.Query("SearchPullRequests", &queryResult, variables)
	if err != nil {
		return PullRequestsResponse{}, err
	}
	log.Info("Successfully fetched PRs", "count", queryResult.Search.IssueCount)

	prs := make([]PullRequestData, 0, len(queryResult.Search.Nodes))
	for _, node := range queryResult.Search.Nodes {
		prs = append(prs, node.PullRequest)
	}

	return PullRequestsResponse{
		Prs:        prs,
		TotalCount: queryResult.Search.IssueCount,
		PageInfo:   queryResult.Search.PageInfo,
	}, nil
}

func FetchPullRequest(prUrl string) (EnrichedPullRequestData, error) {
	var err error
	if client == nil {
		client, err = gh.DefaultGraphQLClient()
		if err != nil {
			return EnrichedPullRequestData{}, err
		}
	}

	var queryResult struct {
		Resource struct {
			PullRequest EnrichedPullRequestData `graphql:"... on PullRequest"`
		} `graphql:"resource(url: $url)"`
	}
	parsedUrl, err := url.Parse(prUrl)
	if err != nil {
		return EnrichedPullRequestData{}, err
	}
	variables := map[string]any{
		"url": githubv4.URI{URL: parsedUrl},
	}
	log.Debug("Fetching PR", "url", prUrl)
	err = client.Query("FetchPullRequest", &queryResult, variables)
	if err != nil {
		return EnrichedPullRequestData{}, err
	}
	log.Info("Successfully fetched PR", "url", prUrl)

	return queryResult.Resource.PullRequest, nil
}

// FetchReviewThreads fetches just a pull request's review threads (id, path,
// line, resolution/permission flags, and comments), independent of the full
// enrichment query performed by FetchPullRequest. It always queries GitHub
// directly, bypassing any cached enrichment data, so callers get the
// PR's current thread state.
func FetchReviewThreads(prUrl string) ([]ReviewThreadWithComments, error) {
	var err error
	if client == nil {
		client, err = gh.DefaultGraphQLClient()
		if err != nil {
			return nil, err
		}
	}

	var queryResult struct {
		Resource struct {
			PullRequest struct {
				ReviewThreads ReviewThreadsWithComments `graphql:"reviewThreads(last: 50)"`
			} `graphql:"... on PullRequest"`
		} `graphql:"resource(url: $url)"`
	}
	parsedUrl, err := url.Parse(prUrl)
	if err != nil {
		return nil, err
	}
	variables := map[string]any{
		"url": githubv4.URI{URL: parsedUrl},
	}
	log.Debug("Fetching review threads", "url", prUrl)
	err = client.Query("FetchReviewThreads", &queryResult, variables)
	if err != nil {
		return nil, err
	}
	log.Info("Successfully fetched review threads", "url", prUrl)

	return queryResult.Resource.PullRequest.ReviewThreads.Nodes, nil
}
