package table

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dlvhdr/gh-dash/v4/internal/config"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/constants"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/context"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/theme"
	"github.com/dlvhdr/gh-dash/v4/internal/utils"
)

func groupedTestTable(t *testing.T, width int) Model {
	t.Helper()
	cfg, err := config.ParseConfig(config.Location{
		ConfigFlag:       "../../../config/testdata/test-config.yml",
		SkipGlobalConfig: true,
	})
	require.NoError(t, err)
	thm := theme.ParseTheme(&cfg)
	ctx := context.ProgramContext{Config: &cfg, Theme: thm, Styles: context.InitStyles(thm)}
	now := time.Now()
	m := NewModel(
		ctx,
		constants.Dimensions{Width: width, Height: 8},
		now,
		now,
		[]Column{{Title: "Title", Width: utils.IntPtr(width)}},
		[]Row{{"idle"}, {"active"}},
		"items",
		nil,
		"",
		false,
	)
	m.SetGroupHeaders(map[int]string{0: "○ No active agent", 1: "● Active agent"})
	return m
}

func TestGroupHeadersRenderWithoutBecomingSelectable(t *testing.T) {
	m := groupedTestTable(t, 40)
	view := m.View()
	require.True(t, strings.Contains(view, "No active agent"))
	require.True(t, strings.Contains(view, "Active agent"))
	require.Equal(t, 0, m.GetCurrItem())
	require.Equal(t, 1, m.NextItem())
	require.Equal(t, 0, m.PrevItem())
}

func TestGroupHeadersRenderInNarrowTable(t *testing.T) {
	m := groupedTestTable(t, 8)
	require.NotEmpty(t, m.View())
}
