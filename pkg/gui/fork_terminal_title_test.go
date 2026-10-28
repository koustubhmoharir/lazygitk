package gui

import (
	"path/filepath"
	"testing"
)

func TestForkTerminalTitle(t *testing.T) {
	home := filepath.Join(string(filepath.Separator), "home", "user")
	sep := string(filepath.Separator)

	testCases := []struct {
		name         string
		worktreePath string
		home         string
		want         string
	}{
		{
			name:         "exact home",
			worktreePath: home,
			home:         home,
			want:         "~",
		},
		{
			name:         "path under home",
			worktreePath: home + sep + "projects" + sep + "repo",
			home:         home,
			want:         "~" + sep + "projects" + sep + "repo",
		},
		{
			name:         "path outside home",
			worktreePath: filepath.Join(string(filepath.Separator), "srv", "repo"),
			home:         home,
			want:         filepath.Join(string(filepath.Separator), "srv", "repo"),
		},
		{
			name:         "sibling sharing a prefix with home",
			worktreePath: home + "name" + sep + "repo",
			home:         home,
			want:         home + "name" + sep + "repo",
		},
		{
			name:         "empty home",
			worktreePath: home + sep + "projects" + sep + "repo",
			home:         "",
			want:         home + sep + "projects" + sep + "repo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := forkTerminalTitle(tc.worktreePath, tc.home); got != tc.want {
				t.Errorf("forkTerminalTitle(%q, %q) = %q, want %q", tc.worktreePath, tc.home, got, tc.want)
			}
		})
	}
}
