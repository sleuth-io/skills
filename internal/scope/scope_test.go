package scope

import (
	"path/filepath"
	"testing"

	"github.com/sleuth-io/skills/internal/lockfile"
)

func TestMatchesAsset(t *testing.T) {
	tests := []struct {
		name  string
		scope *Scope
		asset *lockfile.Asset
		want  bool
	}{
		{
			name: "global asset always matches",
			scope: &Scope{
				Type: TypeGlobal,
			},
			asset: &lockfile.Asset{
				Name:   "test",
				Scopes: []lockfile.Scope{}, // Empty = global
			},
			want: true,
		},
		{
			name: "repo asset matches when in same repo",
			scope: &Scope{
				Type:    TypeRepo,
				RepoURL: "https://github.com/test/repo",
			},
			asset: &lockfile.Asset{
				Name: "test",
				Scopes: []lockfile.Scope{
					{Repo: "https://github.com/test/repo"},
				},
			},
			want: true,
		},
		{
			name: "repo asset doesn't match from global scope",
			scope: &Scope{
				Type: TypeGlobal,
			},
			asset: &lockfile.Asset{
				Name: "test",
				Scopes: []lockfile.Scope{
					{Repo: "https://github.com/test/repo"},
				},
			},
			want: false,
		},
		{
			name: "path asset matches when in matching path",
			scope: &Scope{
				Type:     TypePath,
				RepoURL:  "https://github.com/test/repo",
				RepoPath: "src/components",
			},
			asset: &lockfile.Asset{
				Name: "test",
				Scopes: []lockfile.Scope{
					{Repo: "https://github.com/test/repo", Paths: []string{"src/components"}},
				},
			},
			want: true,
		},
		{
			name: "path asset doesn't match when in different path",
			scope: &Scope{
				Type:     TypePath,
				RepoURL:  "https://github.com/test/repo",
				RepoPath: "src/utils",
			},
			asset: &lockfile.Asset{
				Name: "test",
				Scopes: []lockfile.Scope{
					{Repo: "https://github.com/test/repo", Paths: []string{"src/components"}},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewMatcher(tt.scope)
			if got := matcher.MatchesAsset(tt.asset); got != tt.want {
				t.Errorf("MatchesAsset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInstallLocations(t *testing.T) {
	repoRoot := "/home/user/repo"
	globalBase := "/home/user/.claude"

	tests := []struct {
		name      string
		asset     *lockfile.Asset
		scope     *Scope
		wantPaths []string
	}{
		{
			name: "global asset",
			asset: &lockfile.Asset{
				Name:   "test",
				Scopes: []lockfile.Scope{},
			},
			scope: &Scope{
				Type: TypeGlobal,
			},
			wantPaths: []string{globalBase},
		},
		{
			name: "repo asset",
			asset: &lockfile.Asset{
				Name: "test",
				Scopes: []lockfile.Scope{
					{Repo: "https://github.com/test/repo"},
				},
			},
			scope: &Scope{
				Type:    TypeRepo,
				RepoURL: "https://github.com/test/repo",
			},
			wantPaths: []string{filepath.Join(repoRoot, ".claude")},
		},
		{
			name: "path asset",
			asset: &lockfile.Asset{
				Name: "test",
				Scopes: []lockfile.Scope{
					{Repo: "https://github.com/test/repo", Paths: []string{"src/components"}},
				},
			},
			scope: &Scope{
				Type:     TypePath,
				RepoURL:  "https://github.com/test/repo",
				RepoPath: "src/components",
			},
			wantPaths: []string{filepath.Join(repoRoot, "src/components", ".claude")},
		},
		{
			name: "multiple paths",
			asset: &lockfile.Asset{
				Name: "test",
				Scopes: []lockfile.Scope{
					{Repo: "https://github.com/test/repo", Paths: []string{"src/components", "src/utils"}},
				},
			},
			scope: &Scope{
				Type:     TypePath,
				RepoURL:  "https://github.com/test/repo",
				RepoPath: "src/components",
			},
			wantPaths: []string{filepath.Join(repoRoot, "src/components", ".claude")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetInstallLocations(tt.asset, tt.scope, repoRoot, globalBase)
			if len(got) != len(tt.wantPaths) {
				t.Errorf("GetInstallLocations() returned %d paths, want %d", len(got), len(tt.wantPaths))
				return
			}
			for i, path := range got {
				if path != tt.wantPaths[i] {
					t.Errorf("GetInstallLocations()[%d] = %v, want %v", i, path, tt.wantPaths[i])
				}
			}
		})
	}
}
