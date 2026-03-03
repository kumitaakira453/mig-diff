package selector

import (
	"os/exec"
	"strings"

	"github.com/manifoldco/promptui"
)

// SelectBranch shows an interactive branch selector and returns the selected branch.
func SelectBranch() (string, error) {
	branches, err := getLocalBranches()
	if err != nil {
		return "", err
	}

	currentBranch, _ := getCurrentBranch()

	// Filter out current branch
	var filteredBranches []string
	for _, b := range branches {
		if b != currentBranch {
			filteredBranches = append(filteredBranches, b)
		}
	}

	if len(filteredBranches) == 0 {
		return "", nil
	}

	prompt := promptui.Select{
		Label: "Select target branch",
		Items: filteredBranches,
		Size:  15,
		Searcher: func(input string, index int) bool {
			branch := filteredBranches[index]
			return strings.Contains(strings.ToLower(branch), strings.ToLower(input))
		},
		StartInSearchMode: true,
	}

	_, result, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return result, nil
}

// getLocalBranches returns a list of local branch names.
func getLocalBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var branches []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
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
