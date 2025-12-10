package handlers

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/sleuth-io/skills/internal/metadata"
)

// MCPRemoteHandler handles MCP remote artifact installation for Cursor
// MCP remote artifacts contain only configuration, no server code
type MCPRemoteHandler struct {
	metadata *metadata.Metadata
}

// NewMCPRemoteHandler creates a new MCP remote handler
func NewMCPRemoteHandler(meta *metadata.Metadata) *MCPRemoteHandler {
	return &MCPRemoteHandler{metadata: meta}
}

// Install installs the MCP remote configuration (no extraction needed)
func (h *MCPRemoteHandler) Install(ctx context.Context, zipData []byte, targetBase string) error {
	mcpConfigPath := filepath.Join(targetBase, "mcp.json")

	// Read existing mcp.json
	config, err := ReadMCPConfig(mcpConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read mcp.json: %w", err)
	}

	// Generate MCP entry from metadata (no path conversion for remote)
	entry := h.generateMCPEntry()

	// Add to config
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]interface{})
	}
	config.MCPServers[h.metadata.Artifact.Name] = entry

	// Write updated mcp.json
	if err := WriteMCPConfig(mcpConfigPath, config); err != nil {
		return fmt.Errorf("failed to write mcp.json: %w", err)
	}

	return nil
}

// Remove uninstalls the MCP remote configuration
func (h *MCPRemoteHandler) Remove(ctx context.Context, targetBase string) error {
	mcpConfigPath := filepath.Join(targetBase, "mcp.json")

	// Read existing mcp.json
	config, err := ReadMCPConfig(mcpConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read mcp.json: %w", err)
	}

	// Remove entry
	delete(config.MCPServers, h.metadata.Artifact.Name)

	// Write updated mcp.json
	if err := WriteMCPConfig(mcpConfigPath, config); err != nil {
		return fmt.Errorf("failed to write mcp.json: %w", err)
	}

	return nil
}

func (h *MCPRemoteHandler) generateMCPEntry() map[string]interface{} {
	mcpConfig := h.metadata.MCP

	// For remote MCPs, commands are external (npx, docker, etc.)
	// No path conversion needed
	args := make([]interface{}, len(mcpConfig.Args))
	for i, arg := range mcpConfig.Args {
		args[i] = arg
	}

	entry := map[string]interface{}{
		"command": mcpConfig.Command,
		"args":    args,
	}

	// Add env if present
	if len(mcpConfig.Env) > 0 {
		entry["env"] = mcpConfig.Env
	}

	return entry
}
