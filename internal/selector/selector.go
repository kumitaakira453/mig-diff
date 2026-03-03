package selector

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
)

// BranchInfo holds branch information with commit details.
type BranchInfo struct {
	Name               string
	Author             string
	CommitHash         string
	CommitMessage      string // Full commit message for Details
	CommitMessageShort string // Truncated for list display
	CommitDate         time.Time
	RelativeTime       string
}

// SelectBranch shows an interactive branch selector and returns the selected branch.
func SelectBranch() (string, error) {
	branches, err := getBranchesWithInfo()
	if err != nil {
		return "", err
	}

	currentBranch, _ := getCurrentBranch()

	// Filter out current branch
	var filteredBranches []BranchInfo
	for _, b := range branches {
		if b.Name != currentBranch {
			filteredBranches = append(filteredBranches, b)
		}
	}

	if len(filteredBranches) == 0 {
		return "", nil
	}

	// Sort by commit date (most recent first)
	sort.Slice(filteredBranches, func(i, j int) bool {
		return filteredBranches[i].CommitDate.After(filteredBranches[j].CommitDate)
	})

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "▶ {{ .Name | cyan }}  {{ .RelativeTime | faint }}  {{ .Author | yellow }} • {{ .CommitHash | green }} • {{ .CommitMessageShort | faint }}",
		Inactive: "  {{ .Name | cyan }}  {{ .RelativeTime | faint }}  {{ .Author | yellow }} • {{ .CommitHash | green }} • {{ .CommitMessageShort | faint }}",
		Selected: "✔ {{ .Name | cyan }}",
		Details: `
--------- Details ----------
{{ "Branch:" | faint }}	{{ .Name }}
{{ "Author:" | faint }}	{{ .Author }}
{{ "Commit:" | faint }}	{{ .CommitHash }}
{{ "Message:" | faint }}	{{ .CommitMessage }}
{{ "Time:" | faint }}	{{ .RelativeTime }}`,
	}

	prompt := promptui.Select{
		Label:     "Select target branch",
		Items:     filteredBranches,
		Templates: templates,
		Size:      15,
		Searcher: func(input string, index int) bool {
			branch := filteredBranches[index]
			input = strings.ToLower(input)
			return strings.Contains(strings.ToLower(branch.Name), input) ||
				strings.Contains(strings.ToLower(branch.Author), input) ||
				strings.Contains(strings.ToLower(branch.CommitMessage), input)
		},
		StartInSearchMode: true,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return filteredBranches[idx].Name, nil
}

// getBranchesWithInfo returns branches with commit information.
func getBranchesWithInfo() ([]BranchInfo, error) {
	// Use git for-each-ref for more reliable output
	// Format: refname:short<SEP>authorname<SEP>commithash<SEP>subject<SEP>committerdate:unix
	// Use %09 (tab) as separator in git format
	format := "%(refname:short)%09%(authorname)%09%(objectname:short)%09%(subject)%09%(committerdate:unix)"
	cmd := exec.Command("git", "for-each-ref", "--format="+format, "refs/heads/")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var branches []BranchInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "\t", 5)
		if len(parts) < 1 || parts[0] == "" {
			continue
		}

		branch := BranchInfo{
			Name: parts[0],
		}

		if len(parts) > 1 {
			branch.Author = parts[1]
		}
		if len(parts) > 2 {
			branch.CommitHash = parts[2]
		}
		if len(parts) > 3 {
			branch.CommitMessage = parts[3]
			branch.CommitMessageShort = truncateString(parts[3], 40)
		}
		if len(parts) > 4 {
			var unixTime int64
			fmt.Sscanf(parts[4], "%d", &unixTime)
			if unixTime > 0 {
				branch.CommitDate = time.Unix(unixTime, 0)
				branch.RelativeTime = formatRelativeTime(branch.CommitDate)
			}
		}

		branches = append(branches, branch)
	}

	return branches, nil
}

// getCurrentBranch returns the current branch name.
func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// truncateString truncates a string to maxLen runes, adding "..." if truncated.
// Uses runes to properly handle multi-byte characters (e.g., Japanese).
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

// formatRelativeTime formats a time as a relative time string.
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "たった今"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return fmt.Sprintf("%d分前", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return fmt.Sprintf("%d時間前", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d日前", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		return fmt.Sprintf("%d週間前", weeks)
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		return fmt.Sprintf("%dヶ月前", months)
	default:
		years := int(diff.Hours() / 24 / 365)
		return fmt.Sprintf("%d年前", years)
	}
}
