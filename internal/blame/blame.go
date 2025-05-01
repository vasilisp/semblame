package blame

import (
	"context"
	"log"
	"os"

	"github.com/vasilisp/lingograph"
	"github.com/vasilisp/lingograph/extra"
	"github.com/vasilisp/lingograph/openai"
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
	actor := openai.NewActor(client, openai.GPT4oMini, "You are a helpful assistant that implements a fancier version of git blame. You will receive a list of code changes and a commit message. You will then return a list of the code changes that are most likely to be the cause of the commits followed by the user query. Please answer the query based on the code changes. Please focus on the most relevant commits and don't feel obliged to mention every single one.", nil)

	messages = append(messages, actor.Pipeline(extra.Echoln(os.Stdout, ""), false, 3))

	pipeline := lingograph.Chain(messages...)

	chat := lingograph.NewChat()

	pipeline.Execute(chat)
}
