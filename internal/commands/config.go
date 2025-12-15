package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sleuth-io/skills/internal/artifact"
	"github.com/sleuth-io/skills/internal/artifacts"
	"github.com/sleuth-io/skills/internal/buildinfo"
	"github.com/sleuth-io/skills/internal/cache"
	"github.com/sleuth-io/skills/internal/clients"
	"github.com/sleuth-io/skills/internal/config"
	"github.com/sleuth-io/skills/internal/lockfile"
	"github.com/sleuth-io/skills/internal/utils"
)

// ConfigOutput represents the full config output for JSON serialization
type ConfigOutput struct {
	Version           VersionInfo      `json:"version"`
	Platform          PlatformInfo     `json:"platform"`
	Config            ConfigInfo       `json:"config"`
	Directories       DirectoryInfo    `json:"directories"`
	Clients           []ClientInfo     `json:"clients"`
	Artifacts         []ScopeArtifacts `json:"artifacts"`
	LockFileArtifacts []ScopeArtifacts `json:"lockFileArtifacts,omitempty"`
	RecentLogs        []string         `json:"recentLogs"`
}

type VersionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

type PlatformInfo struct {
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	WorkingDir string `json:"workingDir"`
}

type ConfigInfo struct {
	Path          string `json:"path"`
	Exists        bool   `json:"exists"`
	Type          string `json:"type,omitempty"`
	RepositoryURL string `json:"repositoryUrl,omitempty"`
	ServerURL     string `json:"serverUrl,omitempty"`
}

type DirectoryInfo struct {
	Config         string `json:"config"`
	Cache          string `json:"cache"`
	Artifacts      string `json:"artifacts"`
	GitRepos       string `json:"gitRepos"`
	LockFiles      string `json:"lockFiles"`
	InstalledState string `json:"installedState"`
	LogFile        string `json:"logFile"`
}

type ClientInfo struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Installed      bool     `json:"installed"`
	Version        string   `json:"version,omitempty"`
	Directory      string   `json:"directory"`
	HooksInstalled bool     `json:"hooksInstalled"`
	Supports       []string `json:"supports"`
}

type ScopeArtifacts struct {
	Scope           string         `json:"scope"`
	TrackerPath     string         `json:"trackerPath"`
	LockFileVersion string         `json:"lockFileVersion,omitempty"`
	InstalledAt     string         `json:"installedAt,omitempty"`
	Artifacts       []ArtifactInfo `json:"artifacts"`
}

type ArtifactInfo struct {
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Type    string   `json:"type"`
	Clients []string `json:"clients"`
}

// NewConfigCommand creates the config command
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Display configuration and installation status",
		Long:  "Shows current configuration, detected clients, installed artifacts, and paths for debugging and remote support.",
		RunE:  runConfig,
	}
	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().Bool("all", false, "Show all artifacts from lock file, not just those for current repo context")
	return cmd
}

func runConfig(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	showAll, _ := cmd.Flags().GetBool("all")

	output := gatherConfigInfo(showAll)

	if jsonOutput {
		return printJSON(output)
	}
	return printText(output, showAll)
}

func gatherConfigInfo(showAll bool) ConfigOutput {
	output := ConfigOutput{}

	// Version info
	output.Version = VersionInfo{
		Version: buildinfo.Version,
		Commit:  buildinfo.Commit,
		Date:    buildinfo.Date,
	}

	// Platform info
	cwd, _ := os.Getwd()
	output.Platform = PlatformInfo{
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		WorkingDir: cwd,
	}

	// Config info
	output.Config = gatherConfigDetails()

	// Directory info
	output.Directories = gatherDirectoryInfo()

	// Client info
	output.Clients = gatherClientInfo()

	// Installed artifacts (from tracker)
	output.Artifacts = gatherInstalledArtifacts()

	// Lock file artifacts (when --all is used)
	if showAll {
		output.LockFileArtifacts = gatherLockFileArtifacts()
	}

	// Recent logs
	output.RecentLogs = gatherRecentLogs(5)

	return output
}

func gatherConfigDetails() ConfigInfo {
	configPath, _ := utils.GetConfigFile()
	info := ConfigInfo{
		Path:   configPath,
		Exists: utils.FileExists(configPath),
	}

	if cfg, err := config.Load(); err == nil {
		info.Type = string(cfg.Type)
		info.RepositoryURL = cfg.RepositoryURL
		if cfg.Type == config.RepositoryTypeSleuth {
			info.ServerURL = cfg.GetServerURL()
		}
	}

	return info
}

func gatherDirectoryInfo() DirectoryInfo {
	configDir, _ := utils.GetConfigDir()
	cacheDir, _ := cache.GetCacheDir()
	artifactsDir, _ := cache.GetArtifactCacheDir()
	gitReposDir, _ := cache.GetGitReposCacheDir()
	lockFilesDir, _ := cache.GetLockFileCacheDir()
	trackerDir, _ := cache.GetTrackerCacheDir()

	logFile := ""
	if cacheDir != "" {
		logFile = filepath.Join(cacheDir, "skills.log")
	}

	return DirectoryInfo{
		Config:         configDir,
		Cache:          cacheDir,
		Artifacts:      artifactsDir,
		GitRepos:       gitReposDir,
		LockFiles:      lockFilesDir,
		InstalledState: trackerDir,
		LogFile:        logFile,
	}
}

func gatherClientInfo() []ClientInfo {
	var clientInfos []ClientInfo

	allClients := clients.Global().GetAll()
	for _, client := range allClients {
		info := ClientInfo{
			ID:        client.ID(),
			Name:      client.DisplayName(),
			Installed: client.IsInstalled(),
			Version:   strings.TrimSpace(client.GetVersion()),
			Directory: getClientDirectory(client.ID()),
			Supports:  getClientSupportedTypes(client),
		}
		info.HooksInstalled = checkHooksInstalled(client.ID(), info.Directory)
		clientInfos = append(clientInfos, info)
	}

	return clientInfos
}

func getClientDirectory(clientID string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch clientID {
	case "claude-code":
		return filepath.Join(home, ".claude")
	case "cursor":
		return filepath.Join(home, ".cursor")
	default:
		return ""
	}
}

func getClientSupportedTypes(client clients.Client) []string {
	var supported []string
	for _, t := range artifact.AllTypes() {
		if client.SupportsArtifactType(t) {
			supported = append(supported, t.Key)
		}
	}
	return supported
}

func checkHooksInstalled(clientID, clientDir string) bool {
	if clientDir == "" {
		return false
	}

	switch clientID {
	case "claude-code":
		// Check settings.json for skills hooks
		settingsPath := filepath.Join(clientDir, "settings.json")
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			return false
		}
		return strings.Contains(string(data), "skills install")

	case "cursor":
		// Check hooks.json for skills hooks
		hooksPath := filepath.Join(clientDir, "hooks.json")
		data, err := os.ReadFile(hooksPath)
		if err != nil {
			return false
		}
		return strings.Contains(string(data), "skills install")
	}

	return false
}

func gatherInstalledArtifacts() []ScopeArtifacts {
	tracker, err := artifacts.LoadTracker()
	if err != nil || len(tracker.Artifacts) == 0 {
		return nil
	}

	trackerPath, _ := artifacts.GetTrackerPath()

	// Group artifacts by scope
	grouped := tracker.GroupByScope()

	var scopes []ScopeArtifacts
	for scopeName, arts := range grouped {
		scope := ScopeArtifacts{
			Scope:       scopeName,
			TrackerPath: trackerPath,
			Artifacts:   []ArtifactInfo{},
		}

		for _, art := range arts {
			scope.Artifacts = append(scope.Artifacts, ArtifactInfo{
				Name:    art.Name,
				Version: art.Version,
				Type:    "", // Type not stored in new tracker format
				Clients: art.Clients,
			})
		}

		scopes = append(scopes, scope)
	}

	return scopes
}

func gatherLockFileArtifacts() []ScopeArtifacts {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}

	// Load cached lock file
	lockFileData, err := cache.LoadLockFile(cfg.RepositoryURL)
	if err != nil || len(lockFileData) == 0 {
		return nil
	}

	lf, err := lockfile.Parse(lockFileData)
	if err != nil {
		return nil
	}

	grouped := lf.GroupByScope()

	var scopes []ScopeArtifacts
	for scopeName, arts := range grouped {
		scope := ScopeArtifacts{
			Scope:           scopeName,
			LockFileVersion: lf.Version,
			Artifacts:       []ArtifactInfo{},
		}

		for _, art := range arts {
			scope.Artifacts = append(scope.Artifacts, ArtifactInfo{
				Name:    art.Name,
				Version: art.Version,
				Type:    art.Type.Key,
				Clients: art.Clients,
			})
		}

		scopes = append(scopes, scope)
	}

	return scopes
}

func gatherRecentLogs(lines int) []string {
	cacheDir, err := cache.GetCacheDir()
	if err != nil {
		return nil
	}

	logPath := filepath.Join(cacheDir, "skills.log")
	return readLastLines(logPath, lines)
}

func readLastLines(path string, n int) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if len(allLines) <= n {
		return allLines
	}
	return allLines[len(allLines)-n:]
}

func printJSON(output ConfigOutput) error {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func printText(output ConfigOutput, showAll bool) error {
	fmt.Println("Skills CLI Configuration")
	fmt.Println("========================")
	fmt.Println()

	// Version
	fmt.Printf("Version: %s (commit: %s, built: %s)\n", output.Version.Version, output.Version.Commit, output.Version.Date)
	fmt.Printf("Platform: %s/%s\n", output.Platform.OS, output.Platform.Arch)
	fmt.Printf("Working Directory: %s\n", output.Platform.WorkingDir)
	fmt.Println()

	// Configuration
	fmt.Println("Configuration")
	fmt.Println("-------------")
	existsStr := "exists"
	if !output.Config.Exists {
		existsStr = "not found"
	}
	fmt.Printf("Config File: %s (%s)\n", output.Config.Path, existsStr)
	if output.Config.Type != "" {
		fmt.Printf("Type: %s\n", output.Config.Type)
	}
	if output.Config.RepositoryURL != "" {
		fmt.Printf("Repository URL: %s\n", output.Config.RepositoryURL)
	}
	if output.Config.ServerURL != "" {
		fmt.Printf("Server URL: %s\n", output.Config.ServerURL)
	}
	fmt.Println()

	// Directories
	fmt.Println("Directories")
	fmt.Println("-----------")
	fmt.Printf("Config: %s\n", output.Directories.Config)
	fmt.Printf("Cache: %s\n", output.Directories.Cache)
	fmt.Printf("  └─ artifacts/\n")
	fmt.Printf("  └─ git-repos/\n")
	fmt.Printf("  └─ lockfiles/\n")
	fmt.Printf("  └─ installed-state/\n")
	fmt.Printf("Log File: %s\n", output.Directories.LogFile)
	fmt.Println()

	// Clients
	fmt.Println("Detected Clients")
	fmt.Println("----------------")
	for _, client := range output.Clients {
		fmt.Printf("%s:\n", client.Name)
		installedStr := "no"
		if client.Installed {
			installedStr = "yes"
		}
		fmt.Printf("  Installed: %s\n", installedStr)
		if client.Version != "" {
			fmt.Printf("  Version: %s\n", client.Version)
		}
		fmt.Printf("  Directory: %s\n", client.Directory)
		hooksStr := "no"
		if client.HooksInstalled {
			hooksStr = "yes"
		}
		fmt.Printf("  Hooks: %s\n", hooksStr)
		fmt.Printf("  Supports: %s\n", strings.Join(client.Supports, ", "))
		fmt.Println()
	}

	// Recent logs
	if len(output.RecentLogs) > 0 {
		fmt.Println("Recent Logs (last 5 lines)")
		fmt.Println("--------------------------")
		for _, line := range output.RecentLogs {
			fmt.Println(line)
		}
		fmt.Println()
	}

	// Installed artifacts
	if len(output.Artifacts) > 0 {
		fmt.Println("Installed Artifacts")
		fmt.Println("-------------------")
		for _, scope := range output.Artifacts {
			fmt.Printf("%s Scope:\n", scope.Scope)
			fmt.Printf("  Tracker: %s\n", scope.TrackerPath)
			if scope.LockFileVersion != "" {
				fmt.Printf("  Lock Version: %s\n", scope.LockFileVersion)
			}
			if scope.InstalledAt != "" {
				fmt.Printf("  Installed At: %s\n", scope.InstalledAt)
			}
			fmt.Printf("  Artifacts: %d\n", len(scope.Artifacts))
			for _, art := range scope.Artifacts {
				clientsStr := ""
				if len(art.Clients) > 0 {
					clientsStr = fmt.Sprintf(" → %s", strings.Join(art.Clients, ", "))
				}
				fmt.Printf("    - %s (%s) [%s]%s\n", art.Name, art.Version, art.Type, clientsStr)
			}
			fmt.Println()
		}
	} else {
		fmt.Println("Installed Artifacts")
		fmt.Println("-------------------")
		fmt.Println("No artifacts installed.")
		fmt.Println()
	}

	// Lock file artifacts (when --all is used)
	if showAll && len(output.LockFileArtifacts) > 0 {
		fmt.Println("Lock File Artifacts (all scopes)")
		fmt.Println("---------------------------------")
		for _, scope := range output.LockFileArtifacts {
			fmt.Printf("%s:\n", scope.Scope)
			for _, art := range scope.Artifacts {
				clientsStr := ""
				if len(art.Clients) > 0 {
					clientsStr = fmt.Sprintf(" [%s]", strings.Join(art.Clients, ", "))
				}
				fmt.Printf("  - %s (%s) %s%s\n", art.Name, art.Version, art.Type, clientsStr)
			}
		}
		fmt.Println()
	}

	return nil
}
