package handlers

import (
	"github.com/sleuth-io/skills/internal/artifact"
)

// InstalledArtifactInfo represents information about an installed artifact
type InstalledArtifactInfo struct {
	Name        string
	Version     string
	Type        artifact.Type
	InstallPath string
}
