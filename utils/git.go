package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

// runGitCommand executes a git command and returns its trimmed output or an error.
// This function remains unexported as it's an internal detail of this package.
func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput() // Use CombinedOutput to capture stderr as well
	if err != nil {
		return "", fmt.Errorf("git command failed: %v\nOutput: %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

// IsGitClean checks if the git working directory is clean.
func IsGitClean() (bool, error) {
	output, err := runGitCommand("status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}
	return output == "", nil
}

// GetCurrentBranch returns the current git branch name.
func GetCurrentBranch() (string, error) {
	branch, err := runGitCommand("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	if branch == "HEAD" {
		return "", fmt.Errorf("currently in detached HEAD state")
	}
	return branch, nil
}

// CreateBranch creates and checks out a new branch.
func CreateBranch(name string) error {
	_, err := runGitCommand("checkout", "-b", name)
	if err != nil {
		return fmt.Errorf("failed to create branch '%s': %w", name, err)
	}
	return nil
}

// StageFile stages a specific file.
func StageFile(filePath string) error {
	_, err := runGitCommand("add", filePath)
	if err != nil {
		return fmt.Errorf("failed to stage file '%s': %w", filePath, err)
	}
	return nil
}

// CommitChanges creates a commit with the given message.
func CommitChanges(message string) error {
	_, err := runGitCommand("commit", "-m", message)
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}

// CheckoutBranch checks out an existing branch.
func CheckoutBranch(name string) error {
	_, err := runGitCommand("checkout", name)
	if err != nil {
		return fmt.Errorf("failed to checkout branch '%s': %w", name, err)
	}
	return nil
}
