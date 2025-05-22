package cli

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/vasilisp/semblame/internal/blame"
	"github.com/vasilisp/semblame/internal/db"
	"github.com/vasilisp/semblame/internal/git"
	"github.com/vasilisp/semblame/internal/openai"
	"github.com/vasilisp/semblame/internal/shared"
	"github.com/vasilisp/semblame/internal/util"
)

type embeddingDimensions uint16

func ingest(ctx context.Context, repoPath string) error {
	config := git.NewConfig(ctx, repoPath)

	dbh := db.Open(ctx, config.UUID)
	defer dbh.Close()

	db.InitCommitEmbeddingsTable(dbh)

	client := openai.NewEmbeddingClient(config.Model, config.Dimensions)

	err := git.GitLog(ctx, repoPath, func(commitHash string, entry string) error {
		note, err := git.GetCommitNote(ctx, repoPath, commitHash)
		if err != nil {
			return err
		}

		var embedding []float64

		if note != "" {
			embeddingJSON, err := openai.UnmarshalJSON([]byte(note))
			if err != nil {
				return err
			}

			if embeddingJSON.EmbeddingModel() != config.Model || embeddingJSON.EmbeddingDimensions() != config.Dimensions {
				embedding = nil
			} else {
				embedding, err = embeddingJSON.EmbeddingVector()
				if err != nil {
					return err
				}
			}
		}

		if embedding == nil {
			embedding, err = client.Embed(entry)
			if err != nil {
				return err
			}

			util.Assert(config.Dimensions > 0, "dimensions are not set")

			if config.WriteNotes {
				embeddingJSON := openai.MakeEmbeddingJSON(openai.EmbeddingTypeCommit, config.Model, config.Dimensions, "", embedding)

				noteBytes, err := json.Marshal(embeddingJSON)
				if err != nil {
					return err
				}

				git.SetCommitNote(ctx, repoPath, commitHash, string(noteBytes))
			}
		}

		db.InsertCommitEmbedding(dbh, commitHash, embedding)

		return nil
	})
	if err != nil {
		log.Fatalf("failed to ingest: %v", err)
	}

	return nil
}

func similarityQuery(ctx context.Context, repoPath, query string) []shared.Match {
	config := git.NewConfig(ctx, repoPath)

	dbh := db.Open(ctx, config.UUID)
	defer dbh.Close()

	client := openai.NewEmbeddingClient(config.Model, config.Dimensions)

	embedding, err := client.Embed(query)
	if err != nil {
		log.Fatalf("failed to embed query: %v", err)
	}

	results, err := db.QueryCommitEmbeddings(dbh, embedding, 10)
	if err != nil {
		log.Fatalf("failed to query commit embeddings: %v", err)
	}

	return results
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

		results := similarityQuery(context.Background(), repoPath, query)

		blame.Blame(context.Background(), repoPath, results, query)
	}
}
