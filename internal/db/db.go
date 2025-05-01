package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"path/filepath"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const createCommitsTableSQL = `
CREATE TABLE IF NOT EXISTS commit_embeddings (
    commit_hash TEXT PRIMARY KEY,
    embedding VECTOR
);
`

// InitCommitEmbeddingsTable initializes the commit_embeddings table for mapping git commits to embeddings.
func InitCommitEmbeddingsTable(db *sql.DB) {
	_, err := db.Exec(createCommitsTableSQL)
	if err != nil {
		log.Fatalf("failed to create commit_embeddings table: %v", err)
	}
}

func Open(ctx context.Context, uuid uuid.UUID) *sql.DB {
	sqlite_vec.Auto()

	db, err := sql.Open("sqlite3", filepath.Join("/home/vasilis/.semblame", uuid.String()+".sqlite"))
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	InitCommitEmbeddingsTable(db)

	return db
}

// InsertCommitEmbedding inserts or replaces a commit hash and its embedding vector.
func InsertCommitEmbedding(db *sql.DB, commitHash string, embedding []float64) {
	// Serialize the embedding slice into the vector BLOB format.
	floats := make([]float32, len(embedding))
	for i, v := range embedding {
		floats[i] = float32(v)
	}

	blob, err := sqlite_vec.SerializeFloat32(floats)
	if err != nil {
		log.Fatalf("failed to serialize commit embedding: %v", err)
	}

	_, err = db.Exec(
		"INSERT OR REPLACE INTO commit_embeddings (commit_hash, embedding) VALUES (?, ?)",
		commitHash, blob,
	)
	if err != nil {
		log.Fatalf("failed to insert commit embedding: %v", err)
	}
}

// GetCommitEmbedding retrieves the embedding vector for a given commit hash.
func GetCommitEmbedding(db *sql.DB, commitHash string) ([]float64, error) {
	row := db.QueryRow("SELECT embedding FROM commit_embeddings WHERE commit_hash = ?", commitHash)
	var embedding []float64
	err := row.Scan(&embedding)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit embedding: %w", err)
	}
	return embedding, nil
}
