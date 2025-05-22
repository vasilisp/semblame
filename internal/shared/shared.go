package shared

import "log"

type Match struct {
	CommitHash string
	Distance   float64
}

type EmbeddingModel uint8

const (
	EmbeddingModelAda002 EmbeddingModel = iota
	EmbeddingModel3Small
	EmbeddingModel3Large
)

func (m EmbeddingModel) String() string {
	switch m {
	case EmbeddingModelAda002:
		return "text-embedding-ada-002"
	case EmbeddingModel3Small:
		return "text-embedding-3-small"
	case EmbeddingModel3Large:
		return "text-embedding-3-large"
	default:
		log.Fatalf("invalid embedding model: %d", m)
		return ""
	}
}

func EmbeddingModelFromString(s string) EmbeddingModel {
	switch s {
	case "text-embedding-ada-002":
		return EmbeddingModelAda002
	case "text-embedding-3-small":
		return EmbeddingModel3Small
	case "text-embedding-3-large":
		return EmbeddingModel3Large
	default:
		log.Fatalf("invalid embedding model: %s", s)
		return EmbeddingModel3Large
	}
}
