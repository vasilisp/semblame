package git

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
)

// GitLog runs 'git log -p' in the specified repository path and invokes the
// provided handler for each complete log entry.
func GitLog(ctx context.Context, repoPath string, entryHandler func(commitHash string, entry string) error) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "log", "-p", "--reverse")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	var builder strings.Builder
	var currentCommit string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "commit ") {
			// If we have a previous entry, handle it
			if builder.Len() > 0 && currentCommit != "" {
				if err := entryHandler(currentCommit, builder.String()); err != nil {
					if cmd.Process != nil {
						cmd.Process.Kill()
					}
					return err
				}
				builder.Reset()
			}
			// Extract commit hash
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				currentCommit = fields[1]
			} else {
				currentCommit = ""
			}
		}
		builder.WriteString(line + "\n")
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	if builder.Len() > 0 && currentCommit != "" {
		if err := entryHandler(currentCommit, builder.String()); err != nil {
			return err
		}
	}

	return cmd.Wait()
}

// GetCommit returns the contents of a commit using `git show -p <commitHash>`.
// It returns the output as a string, or an error if the command fails.
func GetCommit(ctx context.Context, repoPath, commitHash string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "show", "-p", commitHash)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(out), nil
}
