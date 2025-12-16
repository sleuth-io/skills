package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/sleuth-io/skills/internal/assets"
	"github.com/sleuth-io/skills/internal/assets/detectors"
	"github.com/sleuth-io/skills/internal/config"
	"github.com/sleuth-io/skills/internal/logger"
	"github.com/sleuth-io/skills/internal/stats"
	vaultpkg "github.com/sleuth-io/skills/internal/vault"
)

// NewReportUsageCommand creates the report-usage command
func NewReportUsageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report-usage",
		Short: "Report asset usage from tool calls (PostToolUse hook)",
		Long: `Parse PostToolUse hook JSON from stdin, detect asset usage,
and report it to the vault. Intended to be called from Claude Code hooks.`,
		Hidden: true, // Hide from help output as it's for internal use
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReportUsage(cmd, args)
		},
	}

	return cmd
}

// PostToolUseEvent represents the JSON payload from Claude Code PostToolUse hook
type PostToolUseEvent struct {
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
}

// runReportUsage executes the report-usage command
func runReportUsage(cmd *cobra.Command, args []string) error {
	// Read JSON from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	// Parse hook event
	var event PostToolUseEvent
	if err := json.Unmarshal(data, &event); err != nil {
		// Silently exit on parse error (not a valid hook event)
		return nil
	}

	// Create all handlers for detection
	allHandlers := []detectors.UsageDetector{
		&detectors.SkillDetector{},
		&detectors.AgentDetector{},
		&detectors.CommandDetector{},
		&detectors.MCPDetector{},
		&detectors.MCPRemoteDetector{},
		&detectors.HookDetector{},
	}

	// Try to detect asset usage from each handler
	var assetName string
	var assetType string
	var detected bool

	for _, handler := range allHandlers {
		assetName, detected = handler.DetectUsageFromToolCall(event.ToolName, event.ToolInput)
		if detected {
			// Get asset type from handler
			if typedHandler, ok := handler.(detectors.AssetTypeDetector); ok {
				assetType = typedHandler.GetType()
			}
			break
		}
	}

	// If no handler detected usage, exit silently
	if !detected || assetName == "" {
		return nil
	}

	// Load tracker to check if asset is installed
	tracker, err := assets.LoadTracker()
	if err != nil {
		// Tracker doesn't exist, exit silently
		return nil
	}

	// Check if asset is in tracker
	var assetVersion string
	found := false
	for _, installed := range tracker.Assets {
		if installed.Name == assetName {
			assetVersion = installed.Version
			found = true
			break
		}
	}

	if !found {
		// Asset not installed by us, exit silently
		return nil
	}

	// Create usage event
	usageEvent := stats.UsageEvent{
		AssetName:    assetName,
		AssetVersion: assetVersion,
		AssetType:    assetType,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}

	// Enqueue event
	if err := stats.EnqueueEvent(usageEvent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to enqueue usage event: %v\n", err)
		return nil // Don't fail the hook
	}

	// Log successful usage tracking
	log := logger.Get()
	log.Info("asset usage tracked", "name", assetName, "version", assetVersion, "type", assetType)

	// Try to flush queue
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Load config to get repository
	cfg, err := config.Load()
	if err != nil {
		// Config not initialized, queue will be flushed later
		return nil
	}

	// Create vault instance
	vault, err := vaultpkg.NewFromConfig(cfg)
	if err != nil {
		// Unknown vault type, queue will be flushed later
		return nil
	}

	// Try to flush queue
	if err := stats.FlushQueue(ctx, vault); err != nil {
		// Flush failed, queue preserved for next attempt
		fmt.Fprintf(os.Stderr, "Warning: failed to flush usage stats: %v\n", err)
	}

	return nil
}
