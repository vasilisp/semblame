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

func ingestNote(ctx context.Context, config *git.Config, repoPath, commitHash string) ([]float64, map[string][]float64, error) {
	var commitEmbedding []float64
	fileEmbeddings := make(map[string][]float64)

	err := git.GetCommitNoteWithCallback(ctx, repoPath, commitHash, func(line []byte) {
		embeddingJSON, err := openai.UnmarshalJSON(line)
		if err != nil {
			log.Fatalf("failed to unmarshal note: %v", err)
		}

		if embeddingJSON.EmbeddingModel() != config.Model || embeddingJSON.EmbeddingDimensions() != config.Dimensions {
			return
		}

		embedding, err := embeddingJSON.EmbeddingVector()
		if err != nil {
			log.Fatalf("failed to get embedding vector: %v", err)
		}

		if embeddingJSON.EmbeddingFile() != "" {
			fileEmbeddings[embeddingJSON.EmbeddingFile()] = embedding
		} else {
			commitEmbedding = embedding
		}
	})
	if err != nil {
		return nil, nil, err
	}

	return commitEmbedding, fileEmbeddings, nil
}

func ingest(ctx context.Context, repoPath string) error {
	config := git.NewConfig(ctx, repoPath)

	dbh := db.Open(ctx, config.UUID)
	defer dbh.Close()

	db.InitTables(dbh)

	client := openai.NewEmbeddingClient(config.Model, config.Dimensions)

	err := git.GitLog(ctx, repoPath, func(commitHash string, entry string) error {
		embedding, fileEmbeddings, err := ingestNote(ctx, &config, repoPath, commitHash)
		if err != nil {
			return err
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
		for filePath, fileEmbedding := range fileEmbeddings {
			db.InsertFileEmbedding(dbh, filePath, fileEmbedding)
		}

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
