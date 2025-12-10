package cursor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sleuth-io/skills/internal/artifact"
	"github.com/sleuth-io/skills/internal/clients"
	"github.com/sleuth-io/skills/internal/clients/cursor/handlers"
	"github.com/sleuth-io/skills/internal/metadata"
)

// Client implements the clients.Client interface for Cursor
type Client struct {
	clients.BaseClient
}

// NewClient creates a new Cursor client
func NewClient() *Client {
	return &Client{
		BaseClient: clients.NewBaseClient(
			"cursor",
			"Cursor",
			[]artifact.Type{
				artifact.TypeMCP,
				artifact.TypeMCPRemote,
				artifact.TypeSkill, // Transform to commands
				artifact.TypeCommand,
				artifact.TypeHook, // Supported via hooks.json
			},
		),
	}
}

// IsInstalled checks if Cursor is installed by checking for .cursor directory
func (c *Client) IsInstalled() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	// Check for .cursor directory (primary indicator)
	configDir := filepath.Join(home, ".cursor")
	if stat, err := os.Stat(configDir); err == nil {
		return stat.IsDir()
	}

	return false
}

// GetVersion returns the Cursor version
func (c *Client) GetVersion() string {
	// Cursor doesn't have a standard --version command
	// Could check package.json in extension directory if needed
	return ""
}

// InstallArtifacts installs artifacts to Cursor using client-specific handlers
func (c *Client) InstallArtifacts(ctx context.Context, req clients.InstallRequest) (clients.InstallResponse, error) {
	resp := clients.InstallResponse{
		Results: make([]clients.ArtifactResult, 0, len(req.Artifacts)),
	}

	// Determine target directory based on scope
	targetBase := c.determineTargetBase(req.Scope)

	// Ensure target directory exists
	if err := os.MkdirAll(targetBase, 0755); err != nil {
		return resp, fmt.Errorf("failed to create target directory: %w", err)
	}

	// Install each artifact using appropriate handler
	for _, bundle := range req.Artifacts {
		result := clients.ArtifactResult{
			ArtifactName: bundle.Artifact.Name,
		}

		var err error
		switch bundle.Metadata.Artifact.Type {
		case artifact.TypeMCP:
			handler := handlers.NewMCPHandler(bundle.Metadata)
			err = handler.Install(ctx, bundle.ZipData, targetBase)
		case artifact.TypeMCPRemote:
			handler := handlers.NewMCPRemoteHandler(bundle.Metadata)
			err = handler.Install(ctx, bundle.ZipData, targetBase)
		case artifact.TypeSkill:
			// Install skill to .cursor/skills/ (not transformed to command)
			handler := handlers.NewSkillHandler(bundle.Metadata)
			err = handler.Install(ctx, bundle.ZipData, targetBase)
		case artifact.TypeCommand:
			handler := handlers.NewCommandHandler(bundle.Metadata)
			err = handler.Install(ctx, bundle.ZipData, targetBase)
		case artifact.TypeHook:
			handler := handlers.NewHookHandler(bundle.Metadata)
			err = handler.Install(ctx, bundle.ZipData, targetBase)
		default:
			result.Status = clients.StatusSkipped
			result.Message = fmt.Sprintf("Unsupported artifact type: %s", bundle.Metadata.Artifact.Type.Key)
			resp.Results = append(resp.Results, result)
			continue
		}

		if err != nil {
			result.Status = clients.StatusFailed
			result.Error = err
			result.Message = fmt.Sprintf("Installation failed: %v", err)
		} else {
			result.Status = clients.StatusSuccess
			result.Message = fmt.Sprintf("Installed to %s", targetBase)
		}

		resp.Results = append(resp.Results, result)
	}

	// After all artifacts installed, configure skills support
	// Don't fail the entire installation if rules/MCP config fails
	_ = c.configureSkillsSupport(req.Artifacts, req.Scope)

	return resp, nil
}

// UninstallArtifacts removes artifacts from Cursor
func (c *Client) UninstallArtifacts(ctx context.Context, req clients.UninstallRequest) (clients.UninstallResponse, error) {
	resp := clients.UninstallResponse{
		Results: make([]clients.ArtifactResult, 0, len(req.Artifacts)),
	}

	targetBase := c.determineTargetBase(req.Scope)

	for _, art := range req.Artifacts {
		result := clients.ArtifactResult{
			ArtifactName: art.Name,
		}

		// Create minimal metadata for removal
		meta := &metadata.Metadata{
			Artifact: metadata.Artifact{
				Name: art.Name,
				Type: art.Type,
			},
		}

		var err error
		switch art.Type {
		case artifact.TypeMCP, artifact.TypeMCPRemote:
			handler := handlers.NewMCPHandler(meta)
			err = handler.Remove(ctx, targetBase)
		case artifact.TypeSkill:
			handler := handlers.NewSkillHandler(meta)
			err = handler.Remove(ctx, targetBase)
		case artifact.TypeCommand:
			handler := handlers.NewCommandHandler(meta)
			err = handler.Remove(ctx, targetBase)
		case artifact.TypeHook:
			handler := handlers.NewHookHandler(meta)
			err = handler.Remove(ctx, targetBase)
		default:
			result.Status = clients.StatusSkipped
			result.Message = fmt.Sprintf("Unsupported artifact type: %s", art.Type.Key)
			resp.Results = append(resp.Results, result)
			continue
		}

		if err != nil {
			result.Status = clients.StatusFailed
			result.Error = err
		} else {
			result.Status = clients.StatusSuccess
			result.Message = "Uninstalled successfully"
		}

		resp.Results = append(resp.Results, result)
	}

	return resp, nil
}

// determineTargetBase returns the installation directory based on scope
func (c *Client) determineTargetBase(scope *clients.InstallScope) string {
	home, _ := os.UserHomeDir()

	switch scope.Type {
	case clients.ScopeGlobal:
		return filepath.Join(home, ".cursor")
	case clients.ScopeRepository:
		return filepath.Join(scope.RepoRoot, ".cursor")
	case clients.ScopePath:
		return filepath.Join(scope.RepoRoot, scope.Path, ".cursor")
	default:
		return filepath.Join(home, ".cursor")
	}
}

// configureSkillsSupport generates rules files and registers MCP server
func (c *Client) configureSkillsSupport(artifacts []*clients.ArtifactBundle, scope *clients.InstallScope) error {
	targetBase := c.determineTargetBase(scope)

	// 1. Generate rules file with skill descriptions
	if err := c.generateSkillsRulesFile(artifacts, targetBase); err != nil {
		return fmt.Errorf("failed to generate rules file: %w", err)
	}

	// 2. Register skills MCP server (global only)
	if scope.Type == clients.ScopeGlobal {
		if err := c.registerSkillsMCPServer(); err != nil {
			return fmt.Errorf("failed to register MCP server: %w", err)
		}
	}

	return nil
}

// generateSkillsRulesFile creates .cursor/rules/skills/RULE.md with skill metadata
func (c *Client) generateSkillsRulesFile(artifacts []*clients.ArtifactBundle, targetBase string) error {
	rulesDir := filepath.Join(targetBase, "rules", "skills")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return err
	}

	rulePath := filepath.Join(rulesDir, "RULE.md")

	// Build skill list (only skills, not commands/mcps/etc)
	var skillsList string
	skillCount := 0
	for _, bundle := range artifacts {
		if bundle.Metadata.Artifact.Type == artifact.TypeSkill {
			skillCount++
			skillsList += fmt.Sprintf("\n<skill>\n<name>%s</name>\n<description>%s</description>\n</skill>\n",
				bundle.Artifact.Name, bundle.Metadata.Artifact.Description)
		}
	}

	// If no skills, don't create rules file
	if skillCount == 0 {
		return nil
	}

	// Generate complete RULE.md with frontmatter
	content := fmt.Sprintf(`---
description: "Available skills for AI assistance"
alwaysApply: true
---

<!-- AUTO-GENERATED by Sleuth Skills - Do not edit manually -->
<!-- Run 'skills install' to regenerate this file -->

## Available Skills

You have access to the following skills. When a user's task matches a skill, use the %sread_skill%s MCP tool to load full instructions.

<available_skills>
%s
</available_skills>

**Usage**: Invoke %sread_skill(name: "skill-name")%s via the MCP tool when needed.
`, "`", "`", skillsList, "`", "`")

	return os.WriteFile(rulePath, []byte(content), 0644)
}

// registerSkillsMCPServer adds skills MCP server to ~/.cursor/mcp.json
func (c *Client) registerSkillsMCPServer() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	mcpConfigPath := filepath.Join(home, ".cursor", "mcp.json")

	// Read existing mcp.json
	config, err := handlers.ReadMCPConfig(mcpConfigPath)
	if err != nil {
		return err
	}

	// Only add if missing (don't overwrite existing entry)
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]interface{})
	}

	if _, exists := config.MCPServers["skills"]; exists {
		// Already configured, don't overwrite
		return nil
	}

	// Get path to skills binary
	skillsBinary, err := os.Executable()
	if err != nil {
		return err
	}

	// Add skills MCP server entry
	config.MCPServers["skills"] = map[string]interface{}{
		"command": skillsBinary,
		"args":    []string{"serve"},
	}

	return handlers.WriteMCPConfig(mcpConfigPath, config)
}

func init() {
	// Auto-register on package import
	clients.Register(NewClient())
}
