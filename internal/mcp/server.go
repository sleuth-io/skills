package mcpserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sleuth-io/skills/internal/clients"
	"github.com/sleuth-io/skills/internal/gitutil"
)

// Server provides an MCP server that exposes skill operations
type Server struct {
	registry *clients.Registry
}

// NewServer creates a new MCP server
func NewServer(registry *clients.Registry) *Server {
	return &Server{
		registry: registry,
	}
}

// ReadSkillInput is the input type for read_skill tool
type ReadSkillInput struct {
	Name string `json:"name" jsonschema:"name of the skill to read"`
}

// fileRefPattern matches @filename or @path/to/file patterns in skill content
var fileRefPattern = regexp.MustCompile(`@([a-zA-Z0-9_\-./]+\.[a-zA-Z0-9]+)`)

// Run starts the MCP server over stdio
func (s *Server) Run(ctx context.Context) error {
	impl := &mcp.Implementation{
		Name:    "skills",
		Version: "1.0.0",
	}

	mcpServer := mcp.NewServer(impl, nil)

	// Register the read_skill tool - returns plain markdown text
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "read_skill",
		Description: "Read a skill's full instructions and content. Returns the skill content as markdown with @file references resolved to absolute paths.",
	}, s.handleReadSkill)

	// Run over stdio
	return mcpServer.Run(ctx, &mcp.StdioTransport{})
}

// handleReadSkill handles the read_skill tool invocation
// Returns plain markdown text with @file references resolved to absolute paths
func (s *Server) handleReadSkill(ctx context.Context, req *mcp.CallToolRequest, input ReadSkillInput) (*mcp.CallToolResult, any, error) {
	if input.Name == "" {
		return nil, nil, fmt.Errorf("skill name is required")
	}

	// Determine scope from current working directory
	scope, err := s.detectScope(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to detect scope: %w", err)
	}

	// Try each installed client until we find the skill
	installedClients := s.registry.DetectInstalled()
	for _, client := range installedClients {
		content, err := client.ReadSkill(ctx, input.Name, scope)
		if err == nil {
			// Resolve @file references to absolute paths
			resolvedContent := resolveFileReferences(content.Content, content.BaseDir)

			// Return plain markdown text
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: resolvedContent},
				},
			}, nil, nil
		}
	}

	return nil, nil, fmt.Errorf("skill not found: %s", input.Name)
}

// resolveFileReferences replaces @file references with absolute paths
// Only replaces if the file actually exists at the resolved path
func resolveFileReferences(content string, baseDir string) string {
	return fileRefPattern.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the relative path (everything after @)
		relativePath := match[1:] // Remove the @ prefix

		// Build absolute path
		absolutePath := filepath.Join(baseDir, relativePath)

		// Only replace if the file exists
		if _, err := os.Stat(absolutePath); err == nil {
			return "@" + absolutePath
		}

		// File doesn't exist, leave the reference unchanged
		return match
	})
}

// detectScope determines the current scope using gitutil
func (s *Server) detectScope(ctx context.Context) (*clients.InstallScope, error) {
	gitContext, err := gitutil.DetectContext(ctx)
	if err != nil {
		return nil, err
	}

	if !gitContext.IsRepo {
		return &clients.InstallScope{Type: clients.ScopeGlobal}, nil
	}

	if gitContext.RelativePath == "." {
		return &clients.InstallScope{
			Type:     clients.ScopeRepository,
			RepoRoot: gitContext.RepoRoot,
			RepoURL:  gitContext.RepoURL,
		}, nil
	}

	return &clients.InstallScope{
		Type:     clients.ScopePath,
		RepoRoot: gitContext.RepoRoot,
		RepoURL:  gitContext.RepoURL,
		Path:     gitContext.RelativePath,
	}, nil
}
