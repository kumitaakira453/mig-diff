package git

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Migration represents a Django migration file.
type Migration struct {
	Number   int
	Name     string
	FullName string
	FilePath string
}

// migrationPattern matches migration files like "0001_initial.py"
var migrationPattern = regexp.MustCompile(`/migrations/(\d+)_(.*)\.py$`)

// GetCurrentBranch returns the name of the current git branch.
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetMigrations returns all migrations for a given branch and app.
func GetMigrations(branch, app string) ([]Migration, error) {
	// Build the path to search for migrations
	migrationPath := filepath.Join(app, "migrations")

	cmd := exec.Command("git", "ls-tree", branch, "-r", "--name-only", migrationPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	migrations := make([]Migration, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Match migration file pattern
		matches := migrationPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		// Parse migration number
		number, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		// Extract the full migration name (e.g., "0001_initial")
		baseName := filepath.Base(line)
		fullName := strings.TrimSuffix(baseName, ".py")

		migration := Migration{
			Number:   number,
			Name:     matches[2],
			FullName: fullName,
			FilePath: line,
		}

		migrations = append(migrations, migration)
	}

	// Sort migrations by number
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Number < migrations[j].Number
	})

	return migrations, nil
}
