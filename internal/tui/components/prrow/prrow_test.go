package prrow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	graphql "github.com/cli/shurcooL-graphql"
	checks "github.com/dlvhdr/x/gh-checks"
	"github.com/stretchr/testify/require"

	"github.com/dlvhdr/gh-dash/v4/internal/config"
	"github.com/dlvhdr/gh-dash/v4/internal/data"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/components/table"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/constants"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/context"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/theme"
)

func withTestStarStore(t *testing.T) *data.StarStore {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "gh-dash-star-test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	store := data.NewStarStoreForTesting(filepath.Join(tempDir, "starred.json"))
	restore := data.OverrideStarStoreForTesting(store)
	t.Cleanup(restore)
	return store
}

func newTestContext(t *testing.T) *context.ProgramContext {
	t.Helper()
	cfg, err := config.ParseConfig(config.Location{
		ConfigFlag:       "../../../config/testdata/test-config.yml",
		SkipGlobalConfig: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	thm := theme.ParseTheme(&cfg)
	return &context.ProgramContext{
		Config: &cfg,
		Theme:  thm,
		Styles: context.InitStyles(thm),
	}
}

func reviewThreadNodes(resolved ...bool) []struct{ IsResolved bool } {
	nodes := make([]struct{ IsResolved bool }, len(resolved))
	for i, r := range resolved {
		nodes[i] = struct{ IsResolved bool }{IsResolved: r}
	}
	return nodes
}

func TestRenderNumComments(t *testing.T) {
	tests := []struct {
		name     string
		pr       *PullRequest
		expected string
	}{
		{
			name:     "nil Primary returns dash",
			pr:       &PullRequest{Data: &Data{Primary: nil}},
			expected: "-",
		},
		{
			name: "no review threads renders blank",
			pr: &PullRequest{
				Data: &Data{Primary: &data.PullRequestData{}},
			},
			expected: "",
		},
		{
			name: "only resolved review threads renders blank",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						ReviewThreads: data.ReviewThreadsWithResolution{
							Nodes: reviewThreadNodes(true, true),
						},
					},
				},
			},
			expected: "",
		},
		{
			name: "unresolved review threads renders their count",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						ReviewThreads: data.ReviewThreadsWithResolution{
							Nodes: reviewThreadNodes(false, true, false),
						},
					},
				},
			},
			expected: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.pr.Ctx = newTestContext(t)
			result := tt.pr.renderNumComments()
			if !strings.Contains(result, tt.expected) {
				t.Errorf("renderNumComments() = %q, want to contain %q", result, tt.expected)
			}
			if tt.expected == "" && result != "" {
				t.Errorf("renderNumComments() = %q, want empty string", result)
			}
		})
	}
}

func TestRenderStar(t *testing.T) {
	tests := []struct {
		name     string
		pr       *PullRequest
		starred  bool
		expected string
	}{
		{
			name:     "nil Data returns empty string",
			pr:       &PullRequest{Data: nil},
			expected: "",
		},
		{
			name:     "nil Primary returns empty string",
			pr:       &PullRequest{Data: &Data{Primary: nil}},
			expected: "",
		},
		{
			name: "unstarred PR renders blank",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Number:     1,
						Repository: data.Repository{NameWithOwner: "owner/repo"},
					},
				},
			},
			starred:  false,
			expected: "",
		},
		{
			name: "starred PR renders the star glyph",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Number:     1,
						Repository: data.Repository{NameWithOwner: "owner/repo"},
					},
				},
			},
			starred:  true,
			expected: constants.StarIcon,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := withTestStarStore(t)
			if tt.starred {
				store.Star("pr:owner/repo#1")
			}

			tt.pr.Ctx = newTestContext(t)
			result := tt.pr.renderStar()
			if tt.expected == "" {
				require.Equal(t, "", result)
			} else {
				require.Contains(t, result, tt.expected)
			}
		})
	}
}

func reviewNodes(reviews ...struct {
	Login    string
	Typename string
	State    string
}) []struct {
	State  string
	Author struct {
		Login    string
		Typename graphql.String `graphql:"__typename"`
	}
} {
	nodes := make([]struct {
		State  string
		Author struct {
			Login    string
			Typename graphql.String `graphql:"__typename"`
		}
	}, len(reviews))
	for i, r := range reviews {
		nodes[i].State = r.State
		nodes[i].Author.Login = r.Login
		nodes[i].Author.Typename = graphql.String(r.Typename)
	}
	return nodes
}

func TestRenderReviewStatusHumanAndBot(t *testing.T) {
	type reviewInput = struct {
		Login    string
		Typename string
		State    string
	}

	tests := []struct {
		name          string
		pr            *PullRequest
		wantHuman     string
		wantBot       string
		wantHumanIcon string
		wantBotIcon   string
	}{
		{
			name:      "nil Primary renders dash for both",
			pr:        &PullRequest{Data: &Data{Primary: nil}},
			wantHuman: "-",
			wantBot:   "-",
		},
		{
			name: "no reviews renders waiting for both",
			pr: &PullRequest{
				Data: &Data{Primary: &data.PullRequestData{}},
			},
			wantHumanIcon: "waiting",
			wantBotIcon:   "waiting",
		},
		{
			name: "human-only reviews",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Reviews: data.ReviewsWithAuthorType{
							Nodes: reviewNodes(
								reviewInput{Login: "alice", Typename: "User", State: "APPROVED"},
							),
						},
					},
				},
			},
			wantHumanIcon: "approved",
			wantBotIcon:   "waiting",
		},
		{
			name: "bot-only reviews",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Reviews: data.ReviewsWithAuthorType{
							Nodes: reviewNodes(
								reviewInput{
									Login:    "dependabot",
									Typename: "Bot",
									State:    "CHANGES_REQUESTED",
								},
							),
						},
					},
				},
			},
			wantHumanIcon: "waiting",
			wantBotIcon:   "changesRequested",
		},
		{
			name: "mixed human and bot reviews",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Reviews: data.ReviewsWithAuthorType{
							Nodes: reviewNodes(
								reviewInput{Login: "alice", Typename: "User", State: "COMMENTED"},
								reviewInput{
									Login:    "dependabot",
									Typename: "Bot",
									State:    "APPROVED",
								},
							),
						},
					},
				},
			},
			wantHumanIcon: "commented",
			wantBotIcon:   "approved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.pr.Ctx = newTestContext(t)

			human := tt.pr.renderReviewStatusHuman()
			bot := tt.pr.renderReviewStatusBot()

			if tt.wantHuman != "" {
				require.Equal(t, tt.wantHuman, human)
			}
			if tt.wantBot != "" {
				require.Equal(t, tt.wantBot, bot)
			}

			switch tt.wantHumanIcon {
			case "approved":
				require.Contains(t, human, constants.ApprovedIcon)
			case "changesRequested":
				require.Contains(t, human, constants.ChangesRequestedIcon)
			case "commented":
				require.Contains(t, human, tt.pr.Ctx.Styles.Common.CommentGlyph)
			case "waiting":
				require.Contains(t, human, tt.pr.Ctx.Styles.Common.WaitingGlyph)
			}

			switch tt.wantBotIcon {
			case "approved":
				require.Contains(t, bot, constants.ApprovedIcon)
			case "changesRequested":
				require.Contains(t, bot, constants.ChangesRequestedIcon)
			case "commented":
				require.Contains(t, bot, tt.pr.Ctx.Styles.Common.CommentGlyph)
			case "waiting":
				require.Contains(t, bot, tt.pr.Ctx.Styles.Common.WaitingGlyph)
			}
		})
	}
}

func TestGetStatusChecksRollup(t *testing.T) {
	tests := []struct {
		name     string
		pr       *PullRequest
		expected checks.CommitState
	}{
		{
			name:     "nil Data returns Unknown",
			pr:       &PullRequest{Data: nil},
			expected: checks.CommitStateUnknown,
		},
		{
			name:     "nil Primary returns Unknown",
			pr:       &PullRequest{Data: &Data{Primary: nil}},
			expected: checks.CommitStateUnknown,
		},
		{
			name: "empty Commits returns Unknown",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Commits: data.LastCommitStatus{
							Nodes: []struct {
								Commit struct {
									StatusCheckRollup struct {
										State graphql.String
									}
								}
							}{},
						},
					},
				},
			},
			expected: checks.CommitStateUnknown,
		},
		{
			name: "SUCCESS state returns Success",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Commits: data.LastCommitStatus{
							Nodes: []struct {
								Commit struct {
									StatusCheckRollup struct {
										State graphql.String
									}
								}
							}{
								{
									Commit: struct {
										StatusCheckRollup struct {
											State graphql.String
										}
									}{
										StatusCheckRollup: struct {
											State graphql.String
										}{
											State: "SUCCESS",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: checks.CommitStateSuccess,
		},
		{
			name: "FAILURE state returns Failure",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Commits: data.LastCommitStatus{
							Nodes: []struct {
								Commit struct {
									StatusCheckRollup struct {
										State graphql.String
									}
								}
							}{
								{
									Commit: struct {
										StatusCheckRollup struct {
											State graphql.String
										}
									}{
										StatusCheckRollup: struct {
											State graphql.String
										}{
											State: "FAILURE",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: checks.CommitStateFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pr.GetStatusChecksRollup()
			if result != tt.expected {
				t.Errorf("GetStatusChecksRollup() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRenderLabels(t *testing.T) {
	tests := []struct {
		name         string
		pr           *PullRequest
		isSelected   bool
		wantContains []string
		wantNewlines int
	}{
		{
			name: "nil Data returns empty string",
			pr: &PullRequest{
				Data: nil,
				Ctx:  &context.ProgramContext{},
			},
		},
		{
			name: "nil Primary returns empty string",
			pr: &PullRequest{
				Data: &Data{Primary: nil},
				Ctx:  &context.ProgramContext{},
			},
		},
		{
			name: "empty labels returns empty string",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Labels: data.PRLabels{
							Nodes: []data.Label{},
						},
					},
				},
				Ctx: &context.ProgramContext{},
			},
		},
		{
			name: "single label returns non-empty string",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Labels: data.PRLabels{
							Nodes: []data.Label{
								{Name: "bug", Color: "FF0000"},
							},
						},
					},
				},
				Ctx: &context.ProgramContext{
					Config: &config.Config{
						Theme: &config.ThemeConfig{},
					},
				},
				Columns: []table.Column{
					{Title: constants.LabelsIcon, ComputedWidth: 20},
				},
			},
			wantContains: []string{"bug"},
		},
		{
			name: "compact labels keep overflow summary on one line",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Labels: data.PRLabels{
							Nodes: []data.Label{
								{Name: "bug", Color: "FF0000"},
								{Name: "fix", Color: "00FF00"},
								{Name: "chore", Color: "0000FF"},
							},
						},
					},
				},
				Ctx: &context.ProgramContext{
					Config: &config.Config{
						Theme: &config.ThemeConfig{
							Ui: config.UIThemeConfig{
								Table: config.TableUIThemeConfig{Compact: true},
							},
						},
					},
				},
				Columns: []table.Column{
					{Title: constants.LabelsIcon, ComputedWidth: 12},
				},
			},
			wantContains: []string{"bug", "fix", "+1"},
			wantNewlines: 0,
		},
		{
			name: "selected labels keep content on selected rows",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Labels: data.PRLabels{
							Nodes: []data.Label{
								{Name: "bug", Color: "FF0000"},
								{Name: "fix", Color: "00FF00"},
							},
						},
					},
				},
				Ctx: &context.ProgramContext{
					Config: &config.Config{
						Theme: &config.ThemeConfig{},
					},
				},
				Columns: []table.Column{
					{Title: constants.LabelsIcon, ComputedWidth: 20},
				},
			},
			isSelected:   true,
			wantContains: []string{"bug", "fix"},
			wantNewlines: 0,
		},
		{
			name: "full labels keep overflow summary across two lines",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Labels: data.PRLabels{
							Nodes: []data.Label{
								{Name: "bug", Color: "FF0000"},
								{Name: "fix", Color: "00FF00"},
								{Name: "chore", Color: "0000FF"},
								{Name: "feature", Color: "AAAAAA"},
							},
						},
					},
				},
				Ctx: &context.ProgramContext{
					Config: &config.Config{
						Theme: &config.ThemeConfig{},
					},
				},
				Columns: []table.Column{
					{Title: constants.LabelsIcon, ComputedWidth: 14},
				},
			},
			wantContains: []string{"bug", "fix", "chore", "+1"},
			wantNewlines: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.pr.Ctx.Theme.SelectedBackground = compat.AdaptiveColor{
				Light: lipgloss.Color("7"),
				Dark:  lipgloss.Color("7"),
			}
			result := tt.pr.renderLabels(tt.isSelected)

			// For nil/empty cases, expect empty string
			if tt.pr.Data == nil ||
				tt.pr.Data.Primary == nil ||
				len(tt.pr.Data.Primary.Labels.Nodes) == 0 {
				if result != "" {
					t.Errorf("renderLabels() = %q, want empty string", result)
				}
				return
			}

			if result == "" {
				t.Errorf(
					"renderLabels() returned empty string for %d labels",
					len(tt.pr.Data.Primary.Labels.Nodes),
				)
			}

			if strings.Count(result, "\n") != tt.wantNewlines {
				t.Errorf(
					"renderLabels() newline count = %d, want %d",
					strings.Count(result, "\n"),
					tt.wantNewlines,
				)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("renderLabels() = %q, want substring %q", result, want)
				}
			}
		})
	}
}

func TestRenderMergeStatus(t *testing.T) {
	tests := []struct {
		name     string
		pr       *PullRequest
		wantIcon string
	}{
		{
			name:     "nil Primary returns empty string",
			pr:       &PullRequest{Data: &Data{Primary: nil}},
			wantIcon: "",
		},
		{
			name: "CONFLICTING mergeable renders failure icon",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Mergeable: "CONFLICTING",
					},
				},
			},
			wantIcon: constants.FailureIcon,
		},
		{
			name: "CLEAN merge state renders success icon",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Mergeable:        "MERGEABLE",
						MergeStateStatus: "CLEAN",
					},
				},
			},
			wantIcon: constants.SuccessIcon,
		},
		{
			name: "BLOCKED merge state renders blocked icon",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Mergeable:        "MERGEABLE",
						MergeStateStatus: "BLOCKED",
					},
				},
			},
			wantIcon: constants.BlockedIcon,
		},
		{
			name: "BEHIND merge state renders behind icon",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Mergeable:        "MERGEABLE",
						MergeStateStatus: "BEHIND",
					},
				},
			},
			wantIcon: constants.BehindIcon,
		},
		{
			name: "CONFLICTING takes precedence over CLEAN merge state",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Mergeable:        "CONFLICTING",
						MergeStateStatus: "CLEAN",
					},
				},
			},
			wantIcon: constants.FailureIcon,
		},
		{
			name: "unknown merge state renders empty string",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Mergeable:        "UNKNOWN",
						MergeStateStatus: "UNKNOWN",
					},
				},
			},
			wantIcon: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.pr.Ctx = newTestContext(t)
			result := tt.pr.renderMergeStatus()
			if tt.wantIcon == "" {
				require.Equal(t, "", result)
			} else {
				require.Contains(t, result, tt.wantIcon)
			}
		})
	}
}

func TestRenderMergeStateStatus(t *testing.T) {
	tests := []struct {
		name     string
		pr       *PullRequest
		expected string
	}{
		{
			name: "CONFLICTING renders conflict text",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Mergeable: "CONFLICTING",
					},
				},
			},
			expected: "Conflicting",
		},
		{
			name: "CLEAN renders up-to-date text",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						MergeStateStatus: "CLEAN",
					},
				},
			},
			expected: "Up-to-date",
		},
		{
			name: "BLOCKED renders blocked text",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						MergeStateStatus: "BLOCKED",
					},
				},
			},
			expected: "Blocked",
		},
		{
			name: "BEHIND renders behind text",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						MergeStateStatus: "BEHIND",
					},
				},
			},
			expected: "Behind",
		},
		{
			name: "CONFLICTING takes precedence over merge state",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						Mergeable:        "CONFLICTING",
						MergeStateStatus: "CLEAN",
					},
				},
			},
			expected: "Conflicting",
		},
		{
			name: "unknown state renders empty",
			pr: &PullRequest{
				Data: &Data{
					Primary: &data.PullRequestData{
						MergeStateStatus: "UNKNOWN",
					},
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pr.RenderMergeStateStatus()
			if tt.expected == "" {
				require.Equal(t, "", result)
			} else {
				require.Contains(t, result, tt.expected)
			}
		})
	}
}
