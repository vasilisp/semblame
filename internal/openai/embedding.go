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

type EmbeddingClient struct {
	client              *openai.Client
	model               shared.EmbeddingModel
	embeddingDimensions uint32
}

func NewEmbeddingClient(model shared.EmbeddingModel, embeddingDimensions uint32) EmbeddingClient {
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

type EmbeddingJSON struct {
	Type       EmbeddingType
	Model      shared.EmbeddingModel
	Dimensions uint32
	File       string
	Vector     []float64
}

type embeddingJSON struct {
	Type       string
	Model      string
	Dimensions uint32
	File       string
	Vector     string
}

func (e *EmbeddingJSON) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 8*len(e.Vector))
	for i, v := range e.Vector {
		bits := math.Float64bits(v)
		binary.LittleEndian.PutUint64(buf[i*8:], bits)
	}

	emb := embeddingJSON{
		Type:       e.Type.String(),
		Model:      e.Model.String(),
		Dimensions: e.Dimensions,
		File:       e.File,
		Vector:     base64.StdEncoding.EncodeToString(buf),
	}

	return json.Marshal(emb)
}

func (e *EmbeddingJSON) UnmarshalJSON(data []byte) error {
	var emb embeddingJSON
	if err := json.Unmarshal(data, &emb); err != nil {
		return err
	}

	var typ EmbeddingType
	if err := typ.FromString(emb.Type); err != nil {
		return fmt.Errorf("invalid type: %v", err)
	}
	e.Type = typ

	e.Model = shared.EmbeddingModelFromString(emb.Model)
	e.File = emb.File
	e.Dimensions = emb.Dimensions

	buf, err := base64.StdEncoding.DecodeString(emb.Vector)
	if err != nil {
		return fmt.Errorf("failed to decode vector base64: %v", err)
	}

	vector := make([]float64, len(buf)/8)
	for i := range vector {
		bits := binary.LittleEndian.Uint64(buf[i*8:])
		vector[i] = math.Float64frombits(bits)
	}
	e.Vector = vector

	return nil
}
