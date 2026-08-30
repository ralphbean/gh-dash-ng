package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsCheckIgnored(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		patterns []string
		identity string
		want     bool
	}{
		{
			name:     "matching check",
			patterns: []string{"fullsend-ai-*/fullsend/dispatch*"},
			identity: "fullsend-ai-bot/fullsend/dispatch-pr",
			want:     true,
		},
		{
			name:     "different workflow",
			patterns: []string{"fullsend-ai-*/fullsend/dispatch*"},
			identity: "fullsend-ai-bot/build/dispatch-pr",
		},
		{
			name:     "one of several",
			patterns: []string{"lint", "octocat/ci/*"},
			identity: "octocat/ci/test",
			want:     true,
		},
		{name: "empty patterns", identity: "octocat/ci/test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, IsCheckIgnored(tt.patterns, tt.identity))
		})
	}
}

func TestIgnoredChecksConfigValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{name: "valid", pattern: "fullsend-ai-*/fullsend/dispatch*"},
		{name: "invalid", pattern: "[unterminated", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.yml")
			contents := "defaults:\n  ignoredChecks:\n    - '" + tt.pattern + "'\n"
			err := os.WriteFile(configPath, []byte(contents), 0o600)
			require.NoError(t, err)

			cfg, err := ParseConfig(Location{ConfigFlag: configPath, SkipGlobalConfig: true})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []string{tt.pattern}, cfg.Defaults.IgnoredChecks)
		})
	}
}
