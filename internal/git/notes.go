package git

import (
	"bufio"
	"context"
	"os/exec"

	"github.com/vasilisp/semblame/internal/util"
)

// GetCommitNoteWithCallback streams the note attached to a given commit hash (if any) line by line,
// calling the provided callback for each line. If no note is found, the callback is not called.
// Returns an error if the command fails for other reasons.
func GetCommitNoteWithCallback(ctx context.Context, repoPath, commitHash string, onLine func([]byte)) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "notes", "show", commitHash)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	found := false
	for scanner.Scan() {
		found = true
		onLine(scanner.Bytes())
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		// If the note does not exist, git notes show exits with status 1 and no output.
		// We treat this as "no note" rather than an error.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 && !found {
			return nil
		}
		return err
	}
	return nil
}

func SetCommitNote(ctx context.Context, repoPath, commitHash, note string) error {
	util.Assert(note != "", "note is empty")

	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "notes", "add", "-f", "-m", note, commitHash)
	return cmd.Run()
}
