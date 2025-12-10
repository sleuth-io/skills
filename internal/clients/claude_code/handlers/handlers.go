package handlers

import (
	"context"
	"fmt"

	"github.com/sleuth-io/skills/internal/artifact"
	"github.com/sleuth-io/skills/internal/handlers"
	"github.com/sleuth-io/skills/internal/metadata"
)

// InstallSkill installs a skill artifact
func InstallSkill(ctx context.Context, zipData []byte, meta *metadata.Metadata, targetBase string) error {
	return installArtifact(ctx, zipData, meta, targetBase)
}

// UninstallSkill removes a skill artifact
func UninstallSkill(ctx context.Context, name string, targetBase string) error {
	return uninstallArtifact(ctx, name, artifact.TypeSkill, targetBase)
}

// InstallAgent installs an agent artifact
func InstallAgent(ctx context.Context, zipData []byte, meta *metadata.Metadata, targetBase string) error {
	return installArtifact(ctx, zipData, meta, targetBase)
}

// UninstallAgent removes an agent artifact
func UninstallAgent(ctx context.Context, name string, targetBase string) error {
	return uninstallArtifact(ctx, name, artifact.TypeAgent, targetBase)
}

// InstallCommand installs a command artifact
func InstallCommand(ctx context.Context, zipData []byte, meta *metadata.Metadata, targetBase string) error {
	return installArtifact(ctx, zipData, meta, targetBase)
}

// UninstallCommand removes a command artifact
func UninstallCommand(ctx context.Context, name string, targetBase string) error {
	return uninstallArtifact(ctx, name, artifact.TypeCommand, targetBase)
}

// InstallHook installs a hook artifact
func InstallHook(ctx context.Context, zipData []byte, meta *metadata.Metadata, targetBase string) error {
	return installArtifact(ctx, zipData, meta, targetBase)
}

// UninstallHook removes a hook artifact
func UninstallHook(ctx context.Context, name string, targetBase string) error {
	return uninstallArtifact(ctx, name, artifact.TypeHook, targetBase)
}

// InstallMCP installs an MCP artifact
func InstallMCP(ctx context.Context, zipData []byte, meta *metadata.Metadata, targetBase string) error {
	return installArtifact(ctx, zipData, meta, targetBase)
}

// UninstallMCP removes an MCP artifact
func UninstallMCP(ctx context.Context, name string, targetBase string) error {
	return uninstallArtifact(ctx, name, artifact.TypeMCP, targetBase)
}

// InstallMCPRemote installs a remote MCP artifact
func InstallMCPRemote(ctx context.Context, zipData []byte, meta *metadata.Metadata, targetBase string) error {
	return installArtifact(ctx, zipData, meta, targetBase)
}

// UninstallMCPRemote removes a remote MCP artifact
func UninstallMCPRemote(ctx context.Context, name string, targetBase string) error {
	return uninstallArtifact(ctx, name, artifact.TypeMCPRemote, targetBase)
}

// installArtifact is the common implementation for installing any artifact type
func installArtifact(ctx context.Context, zipData []byte, meta *metadata.Metadata, targetBase string) error {
	handler, err := handlers.NewHandler(meta)
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}
	return handler.Install(ctx, zipData, targetBase)
}

// uninstallArtifact is the common implementation for uninstalling any artifact type
func uninstallArtifact(ctx context.Context, name string, artifactType artifact.Type, targetBase string) error {
	// Create minimal metadata for removal
	meta := &metadata.Metadata{
		Artifact: metadata.Artifact{
			Name: name,
			Type: artifactType,
		},
	}
	handler, err := handlers.NewHandler(meta)
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}
	return handler.Remove(ctx, targetBase)
}
