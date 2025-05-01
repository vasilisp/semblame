package blame

import (
	"context"
	"log"
	"os"

	"github.com/vasilisp/lingograph"
	"github.com/vasilisp/lingograph/extra"
	"github.com/vasilisp/lingograph/openai"
	"github.com/vasilisp/semblame/internal/data"
	"github.com/vasilisp/semblame/internal/git"
	"github.com/vasilisp/semblame/internal/shared"
)

func commitMessages(ctx context.Context, repoPath string, matches []shared.Match) ([]lingograph.Pipeline, error) {
	result := make([]lingograph.Pipeline, len(matches))

	for i, match := range matches {
		commitContent, err := git.GetCommit(ctx, repoPath, match.CommitHash)
		if err != nil {
			return nil, err
		}

		result[i] = lingograph.UserPrompt(
			commitContent,
			false,
		)
	}

	return result, nil
}

func Blame(ctx context.Context, repoPath string, matches []shared.Match, query string) {
	messages, err := commitMessages(ctx, repoPath, matches)
	if err != nil {
		log.Fatalf("failed to get commit messages: %v", err)
	}

	messages = append(messages, lingograph.UserPrompt(query, false))

	client := openai.NewClient(openai.APIKeyFromEnv())
	actor := openai.NewActor(client, openai.GPT41Mini, data.SystemPrompt, nil)

	messages = append(messages, actor.Pipeline(extra.Echoln(os.Stdout, ""), false, 3))

	pipeline := lingograph.Chain(messages...)

	chat := lingograph.NewChat()

	pipeline.Execute(chat)
}
