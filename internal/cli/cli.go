package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/vasilisp/semblame/internal/db"
	"github.com/vasilisp/semblame/internal/git"
	"github.com/vasilisp/semblame/internal/openai"
)

type embeddingDimensions uint16

func ingest(ctx context.Context, repoPath string) error {
	config := git.NewConfig(ctx, repoPath)

	dbh := db.Open(ctx, config.UUID)
	defer dbh.Close()

	db.InitCommitEmbeddingsTable(dbh)

	client := openai.NewEmbeddingClient(config.Model, config.Dimensions)

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

func similarityQuery(ctx context.Context, repoPath, query string) error {
	config := git.NewConfig(ctx, repoPath)

	dbh := db.Open(ctx, config.UUID)
	defer dbh.Close()

	client := openai.NewEmbeddingClient(config.Model, config.Dimensions)

	embedding, err := client.Embed(query)
	if err != nil {
		return err
	}

	results, err := db.QueryCommitEmbeddings(dbh, embedding, 10)
	if err != nil {
		return err
	}

	for _, result := range results {
		fmt.Printf("%s: %f\n", result.CommitHash, result.Distance)
	}

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
		return
	}

	if len(os.Args) >= 4 && os.Args[1] == "query" {
		repoPath := os.Args[2]
		query := os.Args[3]

		if err := similarityQuery(context.Background(), repoPath, query); err != nil {
			panic(err)
		}

		return
	}
}
