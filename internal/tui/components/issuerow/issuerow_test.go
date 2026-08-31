package issuerow

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dlvhdr/gh-dash/v4/internal/config"
	"github.com/dlvhdr/gh-dash/v4/internal/data"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/constants"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/context"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/theme"
)

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

func TestRenderNeedsAttention(t *testing.T) {
	tests := []struct {
		name     string
		issue    *Issue
		user     string
		expected string
	}{
		{
			name:     "empty user returns empty",
			issue:    &Issue{Data: data.IssueData{}},
			user:     "",
			expected: "",
		},
		{
			name:     "no comments returns empty",
			issue:    &Issue{Data: data.IssueData{}},
			user:     "me",
			expected: "",
		},
		{
			name: "last comment by current user returns empty",
			issue: &Issue{
				Data: data.IssueData{
					Comments: data.IssueComments{
						Nodes: []data.IssueComment{
							{Author: struct{ Login string }{Login: "other"}},
							{Author: struct{ Login string }{Login: "me"}},
						},
					},
				},
			},
			user:     "me",
			expected: "",
		},
		{
			name: "last comment by another user shows eyes",
			issue: &Issue{
				Data: data.IssueData{
					Comments: data.IssueComments{
						Nodes: []data.IssueComment{
							{Author: struct{ Login string }{Login: "me"}},
							{Author: struct{ Login string }{Login: "other"}},
						},
					},
				},
			},
			user:     "me",
			expected: constants.EyesIcon,
		},
		{
			name: "single comment by another user shows eyes",
			issue: &Issue{
				Data: data.IssueData{
					Comments: data.IssueComments{
						Nodes: []data.IssueComment{
							{Author: struct{ Login string }{Login: "other"}},
						},
					},
				},
			},
			user:     "me",
			expected: constants.EyesIcon,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newTestContext(t)
			ctx.User = tt.user
			tt.issue.Ctx = ctx
			result := tt.issue.renderNeedsAttention()
			if tt.expected == "" {
				require.Equal(t, "", result)
			} else {
				require.Contains(t, result, tt.expected)
			}
		})
	}
}
