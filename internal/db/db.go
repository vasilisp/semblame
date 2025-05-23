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
	"github.com/vasilisp/semblame/internal/shared"
)

const createCommitsTableSQL = `
CREATE TABLE IF NOT EXISTS commit_embeddings (
    commit_hash TEXT PRIMARY KEY,
    embedding VECTOR
);
`

const createFilesTableSQL = `
CREATE TABLE IF NOT EXISTS file_embeddings (
    file_path TEXT PRIMARY KEY,
    embedding VECTOR
);
`

// InitTables initializes the commit_embeddings and file_embeddings tables.
func InitTables(db *sql.DB) {
	_, err := db.Exec(createCommitsTableSQL)
	if err != nil {
		log.Fatalf("failed to create commit_embeddings table: %v", err)
	}

	_, err = db.Exec(createFilesTableSQL)
	if err != nil {
		log.Fatalf("failed to create file_embeddings table: %v", err)
	}
}

func Open(ctx context.Context, uuid uuid.UUID) *sql.DB {
	sqlite_vec.Auto()

	db, err := sql.Open("sqlite3", filepath.Join("/home/vasilis/.semblame", uuid.String()+".sqlite"))
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	InitTables(db)

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

func InsertFileEmbedding(db *sql.DB, filePath string, embedding []float64) {
	floats := make([]float32, len(embedding))
	for i, v := range embedding {
		floats[i] = float32(v)
	}

	blob, err := sqlite_vec.SerializeFloat32(floats)
	if err != nil {
		log.Fatalf("failed to serialize file embedding: %v", err)
	}

	_, err = db.Exec(
		"INSERT OR REPLACE INTO file_embeddings (file_path, embedding) VALUES (?, ?)",
		filePath, blob,
	)
	if err != nil {
		log.Fatalf("failed to insert file embedding: %v", err)
	}
}

func QueryCommitEmbeddings(db *sql.DB, embedding []float64, n int) ([]shared.Match, error) {
	floats := make([]float32, len(embedding))
	for i, v := range embedding {
		floats[i] = float32(v)
	}

	blob, err := sqlite_vec.SerializeFloat32(floats)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize query embedding: %v", err)
	}

	rows, err := db.Query(`
		SELECT commit_hash, vec_distance_cosine(embedding, ?) as distance
		FROM commit_embeddings
		ORDER BY distance ASC
		LIMIT ?
	`, blob, n)
	if err != nil {
		return nil, fmt.Errorf("failed to query commit embeddings: %v", err)
	}
	defer rows.Close()

	var results []shared.Match
	for rows.Next() {
		var commitHash string
		var distance float64
		if err := rows.Scan(&commitHash, &distance); err != nil {
			return nil, fmt.Errorf("failed to scan query result: %v", err)
		}
		results = append(results, shared.Match{
			CommitHash: commitHash,
			Distance:   distance,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %v", err)
	}

	return results, nil
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
