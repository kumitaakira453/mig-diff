package diff

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/lipgloss"
	"github.com/kumitaakira453/mig-diff/internal/config"
	"github.com/kumitaakira453/mig-diff/internal/git"
)

// CLI colors
var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF"))
	appStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFF00"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8700"))
	migStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87"))
	targetStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF00"))
	commandStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#87D7FF")).Bold(true)
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#808080"))
	boxStyle     = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#87D7FF")).
			Padding(0, 1)
)

// Result holds the diff result for a single app.
type Result struct {
	App                  string
	MigrationsToRollback []git.Migration
	RollbackTarget       *git.Migration
	RollbackCommand      string
	NeedRollback         bool
}

// Run compares migrations between the current branch and a target branch
// for all configured apps and prints the results.
func Run(cfg *config.Config, targetBranch string) error {
	if len(cfg.Apps) == 0 {
		return fmt.Errorf("no apps configured. Please create .mig-diff.yaml with 'apps' list")
	}

	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	fmt.Printf("%s %s → %s\n",
		titleStyle.Render("Comparing migrations:"),
		warningStyle.Render(currentBranch),
		targetStyle.Render(targetBranch))
	fmt.Println(mutedStyle.Render(strings.Repeat("─", 50)))
	fmt.Println()

	var results []*Result
	var commands []string

	for _, app := range cfg.Apps {
		result, err := compareBranchMigrations(cfg, currentBranch, targetBranch, app)
		if err != nil {
			fmt.Printf("%s %s\n", appStyle.Render("App:"), appStyle.Render(app))
			fmt.Printf("  %s\n\n", errorStyle.Render(fmt.Sprintf("Error: %v", err)))
			continue
		}

		results = append(results, result)
		printAppResult(result)

		if result.NeedRollback && result.RollbackCommand != "" {
			commands = append(commands, result.RollbackCommand)
		}
	}

	if len(commands) > 0 {
		fmt.Println(titleStyle.Render("Commands to run:"))
		fmt.Println()

		allCommands := strings.Join(commands, "\n")
		fmt.Println(boxStyle.Render(allCommands))
		fmt.Println()

		if err := clipboard.WriteAll(allCommands); err == nil {
			fmt.Println(successStyle.Render("Commands copied to clipboard!"))
		}
		fmt.Println()

		if shouldRunCommands() {
			fmt.Println(warningStyle.Render("Running rollback commands..."))
			fmt.Println()

			for _, result := range results {
				if result.NeedRollback && result.RollbackCommand != "" {
					if err := executeCommand(result.App, result.RollbackCommand); err != nil {
						fmt.Printf("%s %s\n", errorStyle.Render("x"), errorStyle.Render(err.Error()))
					}
				}
			}
		}
	}

	return nil
}

func compareBranchMigrations(cfg *config.Config, currentBranch, targetBranch, app string) (*Result, error) {
	currentMigs, err := git.GetMigrations(currentBranch, app)
	if err != nil {
		return nil, fmt.Errorf("failed to get migrations from %s: %w", currentBranch, err)
	}

	targetMigs, err := git.GetMigrations(targetBranch, app)
	if err != nil {
		return nil, fmt.Errorf("failed to get migrations from %s: %w", targetBranch, err)
	}

	targetSet := make(map[string]struct{})
	for _, m := range targetMigs {
		targetSet[m.FullName] = struct{}{}
	}

	var migrationsToRollback []git.Migration
	for _, m := range currentMigs {
		if _, exists := targetSet[m.FullName]; !exists {
			migrationsToRollback = append(migrationsToRollback, m)
		}
	}

	sort.Slice(migrationsToRollback, func(i, j int) bool {
		return migrationsToRollback[i].Number > migrationsToRollback[j].Number
	})

	var rollbackTarget *git.Migration
	var rollbackCommand string
	if len(migrationsToRollback) > 0 {
		for i := len(currentMigs) - 1; i >= 0; i-- {
			m := currentMigs[i]
			if _, exists := targetSet[m.FullName]; exists {
				rollbackTarget = &m
				break
			}
		}

		if rollbackTarget != nil {
			rollbackCommand = formatRollbackCommand(cfg.MigrateCmd, app, rollbackTarget.FullName)
		} else {
			rollbackCommand = formatRollbackCommand(cfg.MigrateCmd, app, "zero")
		}
	}

	return &Result{
		App:                  app,
		MigrationsToRollback: migrationsToRollback,
		RollbackTarget:       rollbackTarget,
		RollbackCommand:      rollbackCommand,
		NeedRollback:         len(migrationsToRollback) > 0,
	}, nil
}

func printAppResult(result *Result) {
	fmt.Printf("%s %s\n", appStyle.Render("App:"), appStyle.Render(result.App))

	if !result.NeedRollback {
		fmt.Printf("  %s\n", successStyle.Render("No rollback needed (branches have same migrations)"))
		fmt.Println()
		return
	}

	fmt.Printf("  %s\n", warningStyle.Render("Current branch has migrations not in target:"))
	for _, m := range result.MigrationsToRollback {
		fmt.Printf("    %s %s\n", errorStyle.Render("x"), migStyle.Render(m.FullName))
	}

	if result.RollbackTarget != nil {
		fmt.Printf("  %s %s\n",
			mutedStyle.Render("Rollback to:"),
			targetStyle.Render(result.RollbackTarget.FullName))
	} else {
		fmt.Printf("  %s %s\n",
			mutedStyle.Render("Rollback to:"),
			errorStyle.Render("zero (no common migrations)"))
	}

	fmt.Println()
}

func executeCommand(app, command string) error {
	fmt.Printf("%s %s\n", appStyle.Render("Running:"), commandStyle.Render(command))
	fmt.Println(mutedStyle.Render(strings.Repeat("─", 40)))

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = "."

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	go streamOutput(stdout, os.Stdout, "")
	go streamOutput(stderr, os.Stderr, errorStyle.Render(""))

	if err := cmd.Wait(); err != nil {
		fmt.Println()
		fmt.Printf("%s %s: %v\n", errorStyle.Render("x"), appStyle.Render(app), err)
		return err
	}

	fmt.Println()
	fmt.Printf("%s %s\n", successStyle.Render("Done:"), appStyle.Render(app))
	fmt.Println()

	return nil
}

func streamOutput(r io.Reader, w io.Writer, prefix string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if prefix != "" {
			fmt.Fprintf(w, "%s%s\n", prefix, scanner.Text())
		} else {
			fmt.Fprintln(w, scanner.Text())
		}
	}
}

func formatRollbackCommand(migrateCmd, app, target string) string {
	return fmt.Sprintf("%s %s %s", migrateCmd, app, target)
}

func shouldRunCommands() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(warningStyle.Render("Run these commands? [y/N]: "))

	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}
