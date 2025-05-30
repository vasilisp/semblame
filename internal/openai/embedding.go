package openai

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"

	"github.com/openai/openai-go"
	"github.com/vasilisp/semblame/internal/shared"
	"github.com/vasilisp/semblame/internal/util"
)

type EmbeddingClient interface {
	Embed(str string) ([]float64, error)
	seal()
}

type embeddingClient struct {
	client              *openai.Client
	model               shared.EmbeddingModel
	embeddingDimensions uint32
}

func NewEmbeddingClient(model shared.EmbeddingModel, embeddingDimensions uint32) EmbeddingClient {
	util.Assert(embeddingDimensions > 0, "NewClient non-positive embeddingDimensions")

	client := openai.NewClient()

	return &embeddingClient{
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

func (c *embeddingClient) Embed(str string) ([]float64, error) {
	util.Assert(str != "", "embed empty string")

	strings := *splitTextIntoChunks(str, 512)

	embedding, err := c.client.Embeddings.New(context.TODO(), openai.EmbeddingNewParams{
		Input:      openai.EmbeddingNewParamsInputUnion{OfArrayOfStrings: strings},
		Model:      openai.EmbeddingModel(c.model.String()),
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

func (c *embeddingClient) seal() {}

type EmbeddingType uint8

const (
	EmbeddingTypeCommit EmbeddingType = iota
	EmbeddingTypeFile
)

func (t *EmbeddingType) FromString(s string) error {
	switch s {
	case "commit":
		*t = EmbeddingTypeCommit
		return nil
	case "file":
		*t = EmbeddingTypeFile
		return nil
	default:
		return fmt.Errorf("invalid embedding type: %s", s)
	}
}

func (t EmbeddingType) String() string {
	switch t {
	case EmbeddingTypeCommit:
		return "commit"
	case EmbeddingTypeFile:
		return "file"
	default:
		log.Fatalf("invalid embedding type: %d", t)
		return ""
	}
}

type EmbeddingJSON interface {
	EmbeddingModel() shared.EmbeddingModel
	EmbeddingDimensions() uint32
	EmbeddingVector() ([]float64, error)
	EmbeddingFile() string
}

func MakeEmbeddingJSON(typ EmbeddingType, model shared.EmbeddingModel, dimensions uint32, file string, vector []float64) EmbeddingJSON {
	util.Assert(len(vector) > 0, "MakeEmbeddingJSON empty vector")
	util.Assert(dimensions > 0, "MakeEmbeddingJSON non-positive dimensions")

	bufVector := make([]byte, 8*len(vector))
	for i, v := range vector {
		bits := math.Float64bits(v)
		binary.LittleEndian.PutUint64(bufVector[i*8:], bits)
	}

	return &embeddingJSON{
		Type:       typ.String(),
		Model:      model.String(),
		Dimensions: dimensions,
		File:       file,
		Vector:     base64.StdEncoding.EncodeToString(bufVector),
	}
}

type embeddingJSON struct {
	Type       string `json:"type"`
	Model      string `json:"model"`
	Dimensions uint32 `json:"dimensions"`
	File       string `json:"file"`
	Vector     string `json:"vector"`
}

func UnmarshalJSON(data []byte) (EmbeddingJSON, error) {
	var emb embeddingJSON
	if err := json.Unmarshal(data, &emb); err != nil {
		return nil, err
	}

	return &emb, nil
}

func (e *embeddingJSON) EmbeddingModel() shared.EmbeddingModel {
	return shared.EmbeddingModelFromString(e.Model)
}

func (e *embeddingJSON) EmbeddingDimensions() uint32 {
	return e.Dimensions
}

func (e *embeddingJSON) EmbeddingVector() ([]float64, error) {
	bufVector, err := base64.StdEncoding.DecodeString(e.Vector)
	if err != nil {
		return nil, err
	}

	vector := make([]float64, len(bufVector)/8)
	for i := range vector {
		bits := binary.LittleEndian.Uint64(bufVector[i*8:])
		vector[i] = math.Float64frombits(bits)
	}

	return vector, nil
}

func (e *embeddingJSON) EmbeddingFile() string {
	return e.File
}
