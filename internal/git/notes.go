package git

import (
	"context"
	"os/exec"
	"strings"
)

// GetCommitNote retrieves the note attached to a given commit hash (if any) using `git notes show <commitHash>`.
// It returns the note as a string, or an empty string if no note is found, or an error if the command fails for other reasons.
func GetCommitNote(ctx context.Context, repoPath, commitHash string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "notes", "show", commitHash)

	out, err := cmd.Output()
	if err != nil {
		// If the note does not exist, git notes show exits with status 1 and no output.
		// We treat this as "no note" rather than an error.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", err
	}

	return strings.TrimRight(string(out), "\n"), nil
}

func SetCommitNote(ctx context.Context, repoPath, commitHash, note string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "notes", "add", commitHash, note)
	return cmd.Run()
}
