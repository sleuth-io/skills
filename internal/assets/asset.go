package assets

import (
	"context"

	"github.com/sleuth-io/skills/internal/lockfile"
	"github.com/sleuth-io/skills/internal/metadata"
)

// InstallRequest represents a request to install assets
type InstallRequest struct {
	LockFile    *lockfile.LockFile
	ClientName  string // Client to filter by (e.g., "claude-code")
	Scope       *Scope // Current scope context
	TargetBase  string // Base directory for installation (e.g., ~/.claude/)
	CacheDir    string // Cache directory for assets
	Concurrency int    // Max concurrent downloads (default: 10)
}

// InstallResult represents the result of an installation
type InstallResult struct {
	Installed []string // Successfully installed assets
	Failed    []string // Failed assets
	Errors    []error  // Errors encountered
}

// Scope represents the current working context for scope matching
type Scope struct {
	Type     string // "global", "repo", or "path"
	RepoURL  string // Repository URL (if in a repo)
	RepoPath string // Path relative to repo root (if applicable)
}

// AssetWithMetadata combines lockfile asset with parsed metadata
type AssetWithMetadata struct {
	Asset    *lockfile.Asset
	Metadata *metadata.Metadata
	ZipData  []byte
}

// DownloadTask represents a single asset download task
type DownloadTask struct {
	Asset *lockfile.Asset
	Index int
}

// DownloadResult represents the result of downloading an asset
type DownloadResult struct {
	Asset    *lockfile.Asset
	ZipData  []byte
	Metadata *metadata.Metadata
	Error    error
	Index    int
}

// InstallTask represents a single asset installation task
type InstallTask struct {
	Asset    *lockfile.Asset
	ZipData  []byte
	Metadata *metadata.Metadata
}

// Fetcher defines the interface for fetching assets
type Fetcher interface {
	// FetchAsset downloads a single asset
	FetchAsset(ctx context.Context, asset *lockfile.Asset) (zipData []byte, meta *metadata.Metadata, err error)

	// FetchAssets downloads multiple assets in parallel
	FetchAssets(ctx context.Context, assets []*lockfile.Asset, concurrency int) ([]DownloadResult, error)
}

// Installer defines the interface for installing assets
type Installer interface {
	// Install installs a single asset
	Install(ctx context.Context, asset *lockfile.Asset, zipData []byte, metadata *metadata.Metadata) error

	// InstallAll installs multiple assets in dependency order
	InstallAll(ctx context.Context, assets []*AssetWithMetadata) (*InstallResult, error)

	// Remove removes a single asset
	Remove(ctx context.Context, asset *lockfile.Asset) error
}
