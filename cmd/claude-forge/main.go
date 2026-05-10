package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/michael-freling/claude-code-tools/internal/forge/auth"
	"github.com/michael-freling/claude-code-tools/internal/forge/claudecode"
	"github.com/michael-freling/claude-code-tools/internal/forge/config"
	"github.com/michael-freling/claude-code-tools/internal/forge/container"
	"github.com/michael-freling/claude-code-tools/internal/forge/project"
	"github.com/michael-freling/claude-code-tools/internal/forge/session"
	"github.com/michael-freling/claude-code-tools/internal/forgegh"
	"github.com/michael-freling/claude-code-tools/internal/gateway"
	"github.com/spf13/cobra"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	// Busybox-style multi-call binary: if invoked as "gh" or "forge-gh",
	// act as the forge-gh GitHub CLI wrapper.
	basename := filepath.Base(os.Args[0])
	if basename == "gh" || basename == "forge-gh" {
		client := forgegh.NewClient("http://gateway:8083")
		if err := client.Run(os.Args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Normal CLI mode.
	rootCmd := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "claude-forge",
		Short: "Launch and manage Claude Code sessions in Docker containers",
		Long: `claude-forge orchestrates Claude Code inside Docker containers with a
secure gateway proxy for GitHub access. It manages agent and gateway
containers, Docker networks, and session state.`,
		SilenceUsage: true,
	}

	rootCmd.AddCommand(
		newStartCmd(),
		newResumeCmd(),
		newStopCmd(),
		newStatusCmd(),
		newBuildCmd(),
		newAuthCmd(),
		newVersionCmd(),
		newGatewayCmd(),
		newForgeGHCmd(),
	)

	return rootCmd
}

// generateSessionID returns 8 random hex characters for use as a session identifier.
func generateSessionID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// getGitConfig reads a git config value from the host's git configuration.
func getGitConfig(key string) string {
	cmd := exec.Command("git", "config", "--get", key)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// startSession runs the common logic for the start and resume commands.
// It creates the Docker network, starts gateway and agent containers, attaches
// to the agent container's TTY, and cleans up on exit.
func startSession(skipPermissions, worktree bool, prompt, resumeID string, continueSession bool) error {
	// 1. Current working directory.
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// 2. Load config.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "claude-forge")
	cfg, err := config.Load(configDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 3. Identify project.
	proj, err := project.Identify(cwd)
	if err != nil {
		return fmt.Errorf("failed to identify project: %w", err)
	}
	fmt.Printf("Project: %s/%s (%s)\n", proj.Owner, proj.Repo, proj.Dir)

	// 4. Resolve credentials.
	claudeDir := filepath.Join(homeDir, ".claude")
	creds, err := auth.Resolve(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to resolve credentials: %w", err)
	}
	fmt.Printf("Auth: %s\n", creds.AuthType)

	// 5. Generate session ID.
	sessionID, err := generateSessionID()
	if err != nil {
		return err
	}

	// 6. Construct names.
	networkName := fmt.Sprintf("forge_net_%s_%s", proj.ID, sessionID)
	agentName := fmt.Sprintf("forge-agent-%s-%s", proj.ID, sessionID)
	gatewayName := fmt.Sprintf("forge-gateway-%s-%s", proj.ID, sessionID)

	// 7. Create session directory.
	sessionDir := filepath.Join(homeDir, ".claude-forge", proj.ID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// 8. Read git user config.
	gitUserName := getGitConfig("user.name")
	gitUserEmail := getGitConfig("user.email")

	// 9. Write/update gitconfig.
	ccOpts := claudecode.Options{
		GitUserName:  gitUserName,
		GitUserEmail: gitUserEmail,
	}
	if err := claudecode.WriteGitconfig(configDir, ccOpts); err != nil {
		return fmt.Errorf("failed to write gitconfig: %w", err)
	}
	fmt.Println("Gitconfig: updated")

	// 10. Ensure settings.json exists.
	if err := claudecode.EnsureSettings(configDir); err != nil {
		return fmt.Errorf("failed to ensure settings.json: %w", err)
	}

	// 11. Create Docker client.
	dockerClient, err := container.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	ctx := context.Background()

	// 12. Pull images if not present locally.
	for _, img := range []string{cfg.Images.Agent, cfg.Images.Gateway} {
		exists, err := dockerClient.ImageExists(ctx, img)
		if err != nil {
			return fmt.Errorf("failed to check image %s: %w", img, err)
		}
		if !exists {
			fmt.Printf("Pulling image: %s\n", img)
			if err := dockerClient.PullImage(ctx, img); err != nil {
				return fmt.Errorf("failed to pull image %s: %w", img, err)
			}
		}
	}

	// 13. Create Docker network.
	fmt.Printf("Creating network: %s\n", networkName)
	if _, err := dockerClient.CreateNetwork(ctx, networkName); err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	// Set up cleanup function for network and containers.
	cleanup := func() {
		fmt.Println("\nCleaning up...")
		cleanupCtx := context.Background()
		_ = dockerClient.StopContainer(cleanupCtx, agentName)
		_ = dockerClient.RemoveContainer(cleanupCtx, agentName)
		_ = dockerClient.StopContainer(cleanupCtx, gatewayName)
		_ = dockerClient.RemoveContainer(cleanupCtx, gatewayName)
		_ = dockerClient.RemoveNetwork(cleanupCtx, networkName)
		fmt.Println("Cleanup complete.")
	}

	// 14. Start gateway container.
	fmt.Printf("Starting gateway: %s\n", gatewayName)
	sshDir := filepath.Join(homeDir, ".ssh")
	ghConfigDir := filepath.Join(homeDir, ".config", "gh")
	if _, err := dockerClient.StartGateway(ctx, container.GatewayOptions{
		Name:        gatewayName,
		Image:       cfg.Images.Gateway,
		NetworkName: networkName,
		SSHDir:      sshDir,
		GHConfigDir: ghConfigDir,
		Owner:       proj.Owner,
		Repo:        proj.Repo,
	}); err != nil {
		cleanup()
		return fmt.Errorf("failed to start gateway: %w", err)
	}

	// 15. Build agent command args.
	var agentCmd []string
	if skipPermissions {
		agentCmd = append(agentCmd, "--dangerously-skip-permissions")
	}
	if worktree {
		agentCmd = append(agentCmd, "--worktree")
	}
	if resumeID != "" {
		agentCmd = append(agentCmd, "--resume", resumeID)
	} else if continueSession {
		agentCmd = append(agentCmd, "--continue")
	}
	if prompt != "" {
		agentCmd = append(agentCmd, "-p", prompt)
	}

	// Build environment variables.
	agentEnv := map[string]string{
		"HOME":                "/home/user",
		"GIT_TERMINAL_PROMPT": "0",
		"FORGE_PROJECT_OWNER": proj.Owner,
		"FORGE_PROJECT_REPO":  proj.Repo,
	}
	if creds.AuthType == "api_key" {
		agentEnv["ANTHROPIC_API_KEY"] = creds.Token
	} else {
		agentEnv["CLAUDE_CODE_OAUTH_TOKEN"] = creds.Token
	}

	// Start agent container.
	fmt.Printf("Starting agent: %s\n", agentName)
	if _, err := dockerClient.StartAgent(ctx, container.AgentOptions{
		Name:        agentName,
		Image:       cfg.Images.Agent,
		NetworkName: networkName,
		ProjectDir:  proj.Dir,
		SessionDir:  sessionDir,
		ClaudeDir:   claudeDir,
		ConfigDir:   configDir,
		HomeDir:     homeDir,
		Env:         agentEnv,
		Cmd:         agentCmd,
	}); err != nil {
		cleanup()
		return fmt.Errorf("failed to start agent: %w", err)
	}

	// 16. Attach to the agent container's TTY using docker attach.
	fmt.Println("Claude Code is ready. Attaching to session...")
	attachCmd := exec.Command("docker", "attach", agentName)
	attachCmd.Stdin = os.Stdin
	attachCmd.Stdout = os.Stdout
	attachCmd.Stderr = os.Stderr
	// docker attach returns an error when the container exits, which is expected.
	_ = attachCmd.Run()

	// 17. Clean up after the user exits.
	cleanup()
	return nil
}

// newStartCmd creates the "start" subcommand.
func newStartCmd() *cobra.Command {
	var (
		worktree          bool
		noSkipPermissions bool
		prompt            string
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new Claude Code session in a Docker container",
		Long: `Start launches a new Claude Code agent and gateway in Docker containers.
By default, --dangerously-skip-permissions is enabled. Use --no-skip-permissions
to disable it.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			skipPermissions := !noSkipPermissions
			return startSession(skipPermissions, worktree, prompt, "", false)
		},
	}

	cmd.Flags().BoolVar(&worktree, "worktree", false, "Enable worktree mode for Claude Code")
	cmd.Flags().BoolVar(&noSkipPermissions, "no-skip-permissions", false, "Disable --dangerously-skip-permissions")
	cmd.Flags().StringVarP(&prompt, "prompt", "p", "", "Initial prompt to send to Claude Code")

	return cmd
}

// newResumeCmd creates the "resume" subcommand.
func newResumeCmd() *cobra.Command {
	var list bool

	cmd := &cobra.Command{
		Use:   "resume [session-id]",
		Short: "Resume a past Claude Code session",
		Long: `Resume a previous session by ID, or use --list to see available sessions.
If no session ID is given and --list is not set, the most recent session
is continued.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			proj, err := project.Identify(cwd)
			if err != nil {
				return fmt.Errorf("failed to identify project: %w", err)
			}

			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			sessionDir := filepath.Join(homeDir, ".claude-forge", proj.ID)

			if list {
				sessions, err := session.List(sessionDir)
				if err != nil {
					return fmt.Errorf("failed to list sessions: %w", err)
				}
				if len(sessions) == 0 {
					fmt.Println("No sessions found.")
					return nil
				}

				w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
				fmt.Fprintln(w, "SESSION ID\tCREATED\tFIRST MESSAGE")
				for _, s := range sessions {
					firstMsg := s.FirstMsg
					if len(firstMsg) > 60 {
						firstMsg = firstMsg[:57] + "..."
					}
					fmt.Fprintf(w, "%s\t%s\t%s\n", s.ID, s.CreatedAt.Format(time.RFC3339), firstMsg)
				}
				return w.Flush()
			}

			if len(args) == 1 {
				// Resume a specific session.
				return startSession(true, false, "", args[0], false)
			}

			// No session ID: continue the most recent session.
			return startSession(true, false, "", "", true)
		},
	}

	cmd.Flags().BoolVar(&list, "list", false, "List available sessions")

	return cmd
}

// newStopCmd creates the "stop" subcommand.
func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop running Claude Code containers for the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			proj, err := project.Identify(cwd)
			if err != nil {
				return fmt.Errorf("failed to identify project: %w", err)
			}

			dockerClient, err := container.NewClient()
			if err != nil {
				return fmt.Errorf("failed to create Docker client: %w", err)
			}
			defer dockerClient.Close()

			ctx := context.Background()
			containers, err := dockerClient.ListForgeContainers(ctx)
			if err != nil {
				return fmt.Errorf("failed to list containers: %w", err)
			}

			// Filter containers matching this project.
			var matched []container.ContainerInfo
			for _, c := range containers {
				if strings.Contains(c.Name, proj.ID) {
					matched = append(matched, c)
				}
			}

			if len(matched) == 0 {
				fmt.Println("No running containers found for this project.")
				return nil
			}

			// Stop and remove containers.
			for _, c := range matched {
				fmt.Printf("Stopping: %s\n", c.Name)
				if err := dockerClient.StopContainer(ctx, c.Name); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to stop %s: %v\n", c.Name, err)
				}
				if err := dockerClient.RemoveContainer(ctx, c.Name); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to remove %s: %v\n", c.Name, err)
				}
			}

			// Remove networks matching the project ID.
			// Network names follow: forge_net_<project-id>_<session-id>
			// We rely on docker network rm by name; extract unique session IDs from container names.
			sessionIDs := make(map[string]bool)
			for _, c := range matched {
				// Container name: forge-agent-<project-id>-<session-id> or forge-gateway-<project-id>-<session-id>
				name := c.Name
				name = strings.TrimPrefix(name, "forge-agent-")
				name = strings.TrimPrefix(name, "forge-gateway-")
				// What remains is <project-id>-<session-id>
				// The session ID is the last 8 hex characters.
				if len(name) >= 8 {
					sid := name[len(name)-8:]
					sessionIDs[sid] = true
				}
			}

			for sid := range sessionIDs {
				netName := fmt.Sprintf("forge_net_%s_%s", proj.ID, sid)
				fmt.Printf("Removing network: %s\n", netName)
				if err := dockerClient.RemoveNetwork(ctx, netName); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to remove network %s: %v\n", netName, err)
				}
			}

			fmt.Println("Stopped.")
			return nil
		},
	}
}

// newStatusCmd creates the "status" subcommand.
func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show running Claude Code containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			dockerClient, err := container.NewClient()
			if err != nil {
				return fmt.Errorf("failed to create Docker client: %w", err)
			}
			defer dockerClient.Close()

			ctx := context.Background()
			containers, err := dockerClient.ListForgeContainers(ctx)
			if err != nil {
				return fmt.Errorf("failed to list containers: %w", err)
			}

			if len(containers) == 0 {
				fmt.Println("No running forge containers.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tIMAGE\tSTATUS\tCREATED")
			for _, c := range containers {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					c.Name,
					c.Image,
					c.Status,
					c.Created.Format(time.RFC3339),
				)
			}
			return w.Flush()
		},
	}
}

// newBuildCmd creates the "build" subcommand.
func newBuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Pull or rebuild Claude Code Docker images",
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			configDir := filepath.Join(homeDir, ".config", "claude-forge")
			cfg, err := config.Load(configDir)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			dockerClient, err := container.NewClient()
			if err != nil {
				return fmt.Errorf("failed to create Docker client: %w", err)
			}
			defer dockerClient.Close()

			ctx := context.Background()

			fmt.Printf("Pulling agent image: %s\n", cfg.Images.Agent)
			if err := dockerClient.PullImage(ctx, cfg.Images.Agent); err != nil {
				return fmt.Errorf("failed to pull agent image: %w", err)
			}
			fmt.Println("Agent image pulled.")

			fmt.Printf("Pulling gateway image: %s\n", cfg.Images.Gateway)
			if err := dockerClient.PullImage(ctx, cfg.Images.Gateway); err != nil {
				return fmt.Errorf("failed to pull gateway image: %w", err)
			}
			fmt.Println("Gateway image pulled.")

			fmt.Println("All images up to date.")
			return nil
		},
	}
}

// newAuthCmd creates the "auth" subcommand.
func newAuthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "auth",
		Short: "Verify Claude Code authentication credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			claudeDir := filepath.Join(homeDir, ".claude")

			creds, err := auth.Resolve(claudeDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "No credentials found: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Auth type: %s\n", creds.AuthType)
			// Mask the token for display: show first 8 and last 4 characters.
			token := creds.Token
			if len(token) > 12 {
				fmt.Printf("Token: %s...%s\n", token[:8], token[len(token)-4:])
			} else {
				fmt.Println("Token: [present]")
			}
			return nil
		},
	}
}

// newVersionCmd creates the "version" subcommand.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the claude-forge version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("claude-forge version %s\n", version)
		},
	}
}

// newGatewayCmd creates the "gateway" subcommand for running inside the
// gateway container.
func newGatewayCmd() *cobra.Command {
	var (
		owner     string
		repo      string
		proxyAddr string
		apiAddr   string
	)

	cmd := &cobra.Command{
		Use:   "gateway",
		Short: "Start the gateway server (for container use)",
		Long: `Start the gateway proxy and API server. This is typically invoked as the
entrypoint of the gateway container, not by end users directly.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if owner == "" || repo == "" {
				return fmt.Errorf("--owner and --repo are required")
			}

			srv, err := gateway.NewServer(gateway.ProxyConfig{
				AllowedOwner: owner,
				AllowedRepo:  repo,
			})
			if err != nil {
				return fmt.Errorf("failed to create gateway server: %w", err)
			}

			fmt.Printf("Gateway starting: proxy=%s api=%s owner=%s repo=%s\n", proxyAddr, apiAddr, owner, repo)
			return srv.Run(proxyAddr, apiAddr)
		},
	}

	cmd.Flags().StringVar(&owner, "owner", "", "Allowed GitHub repository owner")
	cmd.Flags().StringVar(&repo, "repo", "", "Allowed GitHub repository name")
	cmd.Flags().StringVar(&proxyAddr, "proxy-addr", ":8080", "Address for the git proxy server")
	cmd.Flags().StringVar(&apiAddr, "api-addr", ":8083", "Address for the API server")

	return cmd
}

// newForgeGHCmd creates the "forge-gh" subcommand as an explicit alternative
// to the os.Args[0] detection for running as the GitHub CLI wrapper.
func newForgeGHCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "forge-gh",
		Short:              "Act as GitHub CLI wrapper (for container use)",
		Long:               `Proxy GitHub CLI commands through the gateway API server. This is used inside the agent container as an alternative to the busybox-style os.Args[0] detection.`,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := forgegh.NewClient("http://gateway:8083")
			return client.Run(args)
		},
	}
}
