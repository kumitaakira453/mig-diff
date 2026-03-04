package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/kumitaakira453/mig-diff/internal/config"
	"github.com/kumitaakira453/mig-diff/internal/diff"
	"github.com/kumitaakira453/mig-diff/internal/selector"
)

func main() {
	// --help is allowed outside a git repo
	if len(os.Args) >= 2 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		printHelp()
		return
	}

	if err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error: not a git repository. Run mig-diff inside a git repository.")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Apps) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no apps configured.")
		fmt.Fprintln(os.Stderr, "Create .mig-diff.yaml or ~/.config/mig-diff/config.yaml with 'apps' list.")
		os.Exit(1)
	}

	var targetBranch string

	if len(os.Args) < 2 {
		// No argument - show interactive branch selector
		branch, err := selector.SelectBranch()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if branch == "" {
			fmt.Println("No other branches available")
			os.Exit(0)
		}
		targetBranch = branch
	} else {
		targetBranch = os.Args[1]
	}

	if err := diff.Run(cfg, targetBranch); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: mig-diff <target-branch>")
	fmt.Println("       mig-diff --help")
}

func printHelp() {
	fmt.Println("mig-diff - Show Django migrations to rollback before switching branches")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  mig-diff                  Interactive branch selector")
	fmt.Println("  mig-diff <target-branch>  Compare current branch with target and show rollback commands")
	fmt.Println("  mig-diff --help, -h       Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mig-diff                  Select branch interactively")
	fmt.Println("  mig-diff main             Compare with main branch")
	fmt.Println("  mig-diff develop          Compare with develop branch")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  - .mig-diff.yaml (repository-specific)")
	fmt.Println("  - ~/.config/mig-diff/config.yaml (global)")
	fmt.Println()
	fmt.Println("Config options:")
	fmt.Println("  apps:        List of Django apps to check")
	fmt.Println("  migrate_cmd: Migration command (default: python manage.py migrate)")
}
