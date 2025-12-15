package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sleuth-io/skills/internal/config"
	"github.com/sleuth-io/skills/internal/registry"
)

const (
	defaultSleuthServerURL = "https://skills.new"
)

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	var (
		repoType  string
		serverURL string
		repoURL   string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration (local path, Git repo, or Sleuth server)",
		Long: `Initialize skills configuration using a local directory, Git repository,
or Sleuth server as the artifact source.

By default, runs in interactive mode with local path as the default option.
Use flags for non-interactive mode.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, args, repoType, serverURL, repoURL)
		},
	}

	cmd.Flags().StringVar(&repoType, "type", "", "Repository type: 'path', 'git', or 'sleuth'")
	cmd.Flags().StringVar(&serverURL, "server-url", "", "Sleuth server URL (for type=sleuth)")
	cmd.Flags().StringVar(&repoURL, "repo-url", "", "Repository URL (git URL, file:// URL, or directory path)")

	return cmd
}

// runInit executes the init command
func runInit(cmd *cobra.Command, args []string, repoType, serverURL, repoURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	out := newOutputHelper(cmd)

	// Check if config already exists
	if config.Exists() {
		out.printErr("Configuration already exists.")
		response, _ := out.prompt("Overwrite existing configuration? (y/N): ")
		response = strings.ToLower(response)
		if response != "y" && response != "yes" {
			return fmt.Errorf("initialization cancelled")
		}
	}

	// Determine if we're in non-interactive mode
	nonInteractive := repoType != ""

	var err error
	if nonInteractive {
		err = runInitNonInteractive(cmd, ctx, repoType, serverURL, repoURL)
	} else {
		err = runInitInteractive(cmd, ctx)
	}

	if err != nil {
		return err
	}

	// Post-init steps (hooks and featured skills)
	runPostInit(cmd, ctx)

	return nil
}

// runPostInit runs common steps after successful initialization
func runPostInit(cmd *cobra.Command, ctx context.Context) {
	out := newOutputHelper(cmd)

	// Install hooks for all detected clients
	installAllClientHooks(ctx, out)

	// Offer to install featured skills
	promptFeaturedSkills(cmd, ctx)
}

// runInitInteractive runs the init command in interactive mode
func runInitInteractive(cmd *cobra.Command, ctx context.Context) error {
	out := newOutputHelper(cmd)

	out.println("Initialize Skills CLI")
	out.println()
	out.println("How will you use skills?")
	out.println("  1) Just for myself (default)")
	out.println("  2) Share with my team")
	out.println()

	choice, _ := out.promptWithDefault("Enter choice", "1")

	switch choice {
	case "1", "":
		return initPersonalRepository(cmd, ctx)
	case "2":
		return initTeamRepository(cmd, ctx)
	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}
}

// initPersonalRepository sets up a local repository in ~/.config/skills/repository
func initPersonalRepository(cmd *cobra.Command, ctx context.Context) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	repoPath := filepath.Join(home, ".config", "skills", "repository")
	return configurePathRepo(cmd, ctx, repoPath)
}

// initTeamRepository prompts for team repository options (git or sleuth)
func initTeamRepository(cmd *cobra.Command, ctx context.Context) error {
	out := newOutputHelper(cmd)

	out.println()
	out.println("Choose how to share with your team:")
	out.println("  1) Sleuth (default)")
	out.println("  2) Git repository")
	out.println()

	choice, _ := out.promptWithDefault("Enter choice", "1")

	switch choice {
	case "1", "":
		return initSleuthServer(cmd, ctx)
	case "2":
		return initGitRepository(cmd, ctx)
	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}
}

// runInitNonInteractive runs the init command in non-interactive mode
func runInitNonInteractive(cmd *cobra.Command, ctx context.Context, repoType, serverURL, repoURL string) error {
	switch repoType {
	case "sleuth":
		if serverURL == "" {
			serverURL = defaultSleuthServerURL
		}
		return authenticateSleuth(cmd, ctx, serverURL)

	case "git":
		if repoURL == "" {
			return fmt.Errorf("--repo-url is required for type=git")
		}
		return configureGitRepo(cmd, ctx, repoURL)

	case "path":
		if repoURL == "" {
			return fmt.Errorf("--repo-url is required for type=path")
		}
		return configurePathRepo(cmd, ctx, repoURL)

	default:
		return fmt.Errorf("invalid repository type: %s (must be 'path', 'git', or 'sleuth')", repoType)
	}
}

// initSleuthServer initializes Sleuth server configuration
func initSleuthServer(cmd *cobra.Command, ctx context.Context) error {
	out := newOutputHelper(cmd)

	out.println()
	serverURL, _ := out.promptWithDefault("Enter Sleuth server URL", defaultSleuthServerURL)

	return authenticateSleuth(cmd, ctx, serverURL)
}

// authenticateSleuth performs OAuth authentication with Sleuth server
func authenticateSleuth(cmd *cobra.Command, ctx context.Context, serverURL string) error {
	out := newOutputHelper(cmd)

	out.println()
	out.println("Authenticating with Sleuth server...")
	out.println()

	// Start OAuth device code flow
	oauthClient := config.NewOAuthClient(serverURL)
	deviceResp, err := oauthClient.StartDeviceFlow(ctx)
	if err != nil {
		return fmt.Errorf("failed to start authentication: %w", err)
	}

	// Display instructions
	out.println("To authenticate, please visit:")
	out.println()
	out.printf("  %s\n", deviceResp.VerificationURI)
	out.println()
	out.printf("And enter code: %s\n", deviceResp.UserCode)
	out.println()

	// Try to open browser
	browserURL := deviceResp.VerificationURIComplete
	if browserURL == "" {
		browserURL = deviceResp.VerificationURI
	}
	if err := config.OpenBrowser(browserURL); err == nil {
		out.println("(Browser opened automatically)")
	}

	out.println()
	out.println("Waiting for authorization...")

	// Poll for token
	tokenResp, err := oauthClient.PollForToken(ctx, deviceResp.DeviceCode)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Save configuration
	cfg := &config.Config{
		Type:          config.RepositoryTypeSleuth,
		RepositoryURL: serverURL,
		AuthToken:     tokenResp.AccessToken,
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	out.println()
	out.println("✓ Authentication successful!")
	out.println("Configuration saved.")

	return nil
}

// initGitRepository initializes Git repository configuration
func initGitRepository(cmd *cobra.Command, ctx context.Context) error {
	out := newOutputHelper(cmd)

	out.println()
	repoURL, _ := out.prompt("Enter Git repository URL: ")

	if repoURL == "" {
		return fmt.Errorf("repository URL is required")
	}

	return configureGitRepo(cmd, ctx, repoURL)
}

// configureGitRepo configures a Git repository
func configureGitRepo(cmd *cobra.Command, ctx context.Context, repoURL string) error {
	out := newOutputHelper(cmd)

	out.println()
	out.println("Configuring Git repository...")

	// Save configuration
	cfg := &config.Config{
		Type:          config.RepositoryTypeGit,
		RepositoryURL: repoURL,
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	out.println()
	out.println("✓ Configuration saved!")
	out.println("Git repository:", repoURL)

	return nil
}

// configurePathRepo configures a local path repository
func configurePathRepo(cmd *cobra.Command, ctx context.Context, repoPath string) error {
	out := newOutputHelper(cmd)

	out.println()
	out.println("Configuring local repository...")

	// Convert path to absolute path first
	var absPath string
	var err error
	if strings.HasPrefix(repoPath, "file://") {
		// Extract path from file:// URL and expand
		repoPath = strings.TrimPrefix(repoPath, "file://")
		absPath, err = expandPath(repoPath)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}
	} else {
		// Expand and normalize the path
		absPath, err = expandPath(repoPath)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}
	}

	// Show the absolute path that will be used
	out.println()
	out.printf("Repository directory: %s\n", absPath)

	// Check if directory exists, create if needed
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		out.println("Directory does not exist, creating...")
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		out.println("✓ Directory created")
	} else {
		out.println("✓ Directory exists")
	}

	// Convert to file:// URL
	repoURL := "file://" + absPath

	// Save configuration
	cfg := &config.Config{
		Type:          config.RepositoryTypePath,
		RepositoryURL: repoURL,
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	out.println()
	out.println("✓ Configuration saved!")

	return nil
}

// expandPath expands tilde and converts relative paths to absolute
func expandPath(path string) (string, error) {
	// Handle tilde
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absPath, nil
}

// promptFeaturedSkills offers to install featured skills after init
func promptFeaturedSkills(cmd *cobra.Command, ctx context.Context) {
	out := newOutputHelper(cmd)

	skills, err := registry.FeaturedSkills()
	if err != nil || len(skills) == 0 {
		return
	}

	var addedAny bool
	for {
		out.println()
		out.println("Would you like to install a featured skill?")
		out.println()

		for i, skill := range skills {
			out.printf("  %d) %s - %s\n", i+1, skill.Name, skill.Description)
		}
		out.println("  0) Done")
		out.println()

		choice, _ := out.promptWithDefault("Enter choice", "0")

		if choice == "0" || choice == "" {
			break
		}

		// Parse choice
		var idx int
		if _, err := fmt.Sscanf(choice, "%d", &idx); err != nil || idx < 1 || idx > len(skills) {
			out.println("Invalid choice")
			continue
		}

		skill := skills[idx-1]
		out.println()
		out.printf("Adding %s...\n", skill.Name)

		// Run the add command with the skill URL (skip install prompt, we'll do it at the end)
		if err := runAddSkipInstall(cmd, skill.URL); err != nil {
			out.printfErr("Failed to add skill: %v\n", err)
		} else {
			addedAny = true
		}
	}

	// If any skills were added, prompt to install once
	if addedAny {
		promptRunInstall(cmd, ctx, out)
	}
}
