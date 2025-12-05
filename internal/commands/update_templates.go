package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/sleuth-io/skills/internal/config"
	"github.com/sleuth-io/skills/internal/repository"
)

// NewUpdateTemplatesCommand creates the update-templates command (hidden)
func NewUpdateTemplatesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "update-templates",
		Hidden: true,
		Short:  "Update templates in Git repository if needed",
		Long: `Check and update templates (install.sh, README.md) in the Git repository
if they are outdated or missing. Only updates files if their version is older
than the current template version. This is a hidden maintenance command.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdateTemplates(cmd, args)
		},
	}

	return cmd
}

func runUpdateTemplates(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	out := newOutputHelper(cmd)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Only works with git repositories
	if cfg.Type != config.RepositoryTypeGit {
		return fmt.Errorf("update-templates only works with git repositories (current type: %s)", cfg.Type)
	}

	if cfg.RepositoryURL == "" {
		return fmt.Errorf("git repository URL not configured")
	}

	// Create repository instance
	repo, err := repository.NewGitRepository(cfg.RepositoryURL)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	// Update templates if needed (with auto-commit)
	// Note: For git repos, this will update files, then commit and push
	updatedFiles, err := repo.UpdateTemplates(ctx, true)
	if err != nil {
		return fmt.Errorf("failed to update templates: %w", err)
	}

	if len(updatedFiles) == 0 {
		out.println("No templates needed updating")
	} else {
		out.println("✓ Templates updated successfully")
		out.println("Updated files:")
		for _, file := range updatedFiles {
			out.printf("  - %s\n", file)
		}
	}

	return nil
}
