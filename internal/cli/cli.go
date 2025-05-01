package cli

import (
	"context"
	"os"

	"github.com/vasilisp/semblame/internal/db"
	"github.com/vasilisp/semblame/internal/git"
	"github.com/vasilisp/semblame/internal/openai"
)

type embeddingDimensions uint16

func ingest(ctx context.Context, repoPath string) error {
	model := git.EmbeddingModel(ctx, repoPath)
	dimensions := git.EmbeddingDimensions(ctx, repoPath)
	uuid := git.RepoUUID(ctx, repoPath)

	dbh := db.Open(ctx, uuid)
	defer dbh.Close()

	db.InitCommitEmbeddingsTable(dbh)

	client := openai.NewEmbeddingClient(model, dimensions)

	git.GitLog(ctx, repoPath, func(commitHash string, entry string) error {
		embedding, err := client.Embed(entry)
		if err != nil {
			return err
		}

		db.InsertCommitEmbedding(dbh, commitHash, embedding)
		return nil
	})

	return nil
}

func Main() {
	if len(os.Args) > 1 && os.Args[1] == "ingest" {
		repoPath := "."
		if len(os.Args) > 2 {
			repoPath = os.Args[2]
		}

		if err := ingest(context.Background(), repoPath); err != nil {
			panic(err)
		}
	}
}
