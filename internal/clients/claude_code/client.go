package claude_code

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sleuth-io/skills/internal/artifact"
	"github.com/sleuth-io/skills/internal/clients"
	"github.com/sleuth-io/skills/internal/clients/claude_code/handlers"
)

// Client implements the clients.Client interface for Claude Code
type Client struct {
	clients.BaseClient
}

// NewClient creates a new Claude Code client
func NewClient() *Client {
	return &Client{
		BaseClient: clients.NewBaseClient(
			"claude-code",
			"Claude Code",
			artifact.AllTypes(),
		),
	}
}

// IsInstalled checks if Claude Code is installed
func (c *Client) IsInstalled() bool {
	// Check multiple indicators
	if c.checkBinaryInPath() {
		return true
	}
	if c.checkConfigDirExists() {
		return true
	}
	if c.checkVSCodeExtension() {
		return true
	}
	return false
}

func (c *Client) checkBinaryInPath() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

func (c *Client) checkConfigDirExists() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	configDir := filepath.Join(home, ".claude")
	_, err = os.Stat(configDir)
	return err == nil
}

func (c *Client) checkVSCodeExtension() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	extPaths := []string{
		filepath.Join(home, ".vscode/extensions"),
		filepath.Join(home, ".vscode-server/extensions"),
	}

	for _, extPath := range extPaths {
		matches, _ := filepath.Glob(filepath.Join(extPath, "anthropic-ai.claude-code-*"))
		if len(matches) > 0 {
			return true
		}
	}
	return false
}

// GetVersion returns the Claude Code version
func (c *Client) GetVersion() string {
	cmd := exec.Command("claude", "--version")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(output)
}

// InstallArtifacts installs artifacts to Claude Code using client-specific handlers
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
		case artifact.TypeSkill:
			err = handlers.InstallSkill(ctx, bundle.ZipData, bundle.Metadata, targetBase)
		case artifact.TypeAgent:
			err = handlers.InstallAgent(ctx, bundle.ZipData, bundle.Metadata, targetBase)
		case artifact.TypeCommand:
			err = handlers.InstallCommand(ctx, bundle.ZipData, bundle.Metadata, targetBase)
		case artifact.TypeHook:
			err = handlers.InstallHook(ctx, bundle.ZipData, bundle.Metadata, targetBase)
		case artifact.TypeMCP:
			err = handlers.InstallMCP(ctx, bundle.ZipData, bundle.Metadata, targetBase)
		case artifact.TypeMCPRemote:
			err = handlers.InstallMCPRemote(ctx, bundle.ZipData, bundle.Metadata, targetBase)
		default:
			err = fmt.Errorf("unsupported artifact type: %s", bundle.Metadata.Artifact.Type.Key)
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

	return resp, nil
}

// UninstallArtifacts removes artifacts from Claude Code
func (c *Client) UninstallArtifacts(ctx context.Context, req clients.UninstallRequest) (clients.UninstallResponse, error) {
	resp := clients.UninstallResponse{
		Results: make([]clients.ArtifactResult, 0, len(req.Artifacts)),
	}

	targetBase := c.determineTargetBase(req.Scope)

	for _, art := range req.Artifacts {
		result := clients.ArtifactResult{
			ArtifactName: art.Name,
		}

		var err error
		switch art.Type {
		case artifact.TypeSkill:
			err = handlers.UninstallSkill(ctx, art.Name, targetBase)
		case artifact.TypeAgent:
			err = handlers.UninstallAgent(ctx, art.Name, targetBase)
		case artifact.TypeCommand:
			err = handlers.UninstallCommand(ctx, art.Name, targetBase)
		case artifact.TypeHook:
			err = handlers.UninstallHook(ctx, art.Name, targetBase)
		case artifact.TypeMCP:
			err = handlers.UninstallMCP(ctx, art.Name, targetBase)
		case artifact.TypeMCPRemote:
			err = handlers.UninstallMCPRemote(ctx, art.Name, targetBase)
		default:
			err = fmt.Errorf("unsupported artifact type: %s", art.Type.Key)
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
		return filepath.Join(home, ".claude")
	case clients.ScopeRepository:
		return filepath.Join(scope.RepoRoot, ".claude")
	case clients.ScopePath:
		return filepath.Join(scope.RepoRoot, scope.Path, ".claude")
	default:
		return filepath.Join(home, ".claude")
	}
}

