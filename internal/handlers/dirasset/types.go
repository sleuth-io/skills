package dirasset

import (
	"github.com/sleuth-io/skills/internal/asset"
)

// InstalledAssetInfo represents information about an installed asset
type InstalledAssetInfo struct {
	Name        string
	Description string
	Version     string
	Type        asset.Type
	InstallPath string
}
