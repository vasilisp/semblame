package openai

import (
	"context"
	"fmt"
	"log"

	"github.com/openai/openai-go"
	"github.com/vasilisp/semblame/internal/util"
)

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

func (m *EmbeddingModel) FromString(s string) error {
	switch s {
	case "text-embedding-ada-002":
		*m = EmbeddingModelAda002
		return nil
	case "text-embedding-3-small":
		*m = EmbeddingModel3Small
		return nil
	case "text-embedding-3-large":
		*m = EmbeddingModel3Large
		return nil
	default:
		return fmt.Errorf("invalid embedding model: %s", s)
	}
}

type EmbeddingClient struct {
	client              *openai.Client
	model               string
	embeddingDimensions uint16
}

func NewEmbeddingClient(model string, embeddingDimensions uint16) EmbeddingClient {
	util.Assert(embeddingDimensions > 0, "NewClient non-positive embeddingDimensions")

	client := openai.NewClient()

	return EmbeddingClient{
		client:              &client,
		model:               model,
		embeddingDimensions: embeddingDimensions,
	}
}

func splitTextIntoChunks(text string, chunkSize int) *[]string {
	var chunks []string
	runes := []rune(text) // Handle multi-byte characters

	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}

	return &chunks
}

func (c EmbeddingClient) Embed(str string) ([]float64, error) {
	util.Assert(str != "", "embed empty string")

	strings := *splitTextIntoChunks(str, 512)

	embedding, err := c.client.Embeddings.New(context.TODO(), openai.EmbeddingNewParams{
		Input:      openai.EmbeddingNewParamsInputUnion{OfArrayOfStrings: strings},
		Model:      c.model,
		Dimensions: openai.Opt(int64(c.embeddingDimensions)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %v", err)
	}

	if len(embedding.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	vector := embedding.Data[0].Embedding

	return vector, nil
}
