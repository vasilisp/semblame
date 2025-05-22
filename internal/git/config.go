package git

import (
	"context"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/vasilisp/semblame/internal/shared"
)

type stringConverter[T any] struct {
	ToString   func(T) string
	FromString func(string) (T, error)
}

func configGet(ctx context.Context, repoPath, key string) (string, error) {
	key = "semblame." + key
	cmdGet := exec.CommandContext(ctx, "git", "-C", repoPath, "config", key)

	out, err := cmdGet.Output()
	if err != nil {
		// Check if the error is due to exit status 1 (key not set), in which case return empty string and no error
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}

		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func configSet(ctx context.Context, repoPath, key, value string) error {
	key = "semblame." + key
	cmdSet := exec.CommandContext(ctx, "git", "-C", repoPath, "config", key, value)
	return cmdSet.Run()
}

func ConfigGetWithDefaultString(ctx context.Context, repoPath, key string, defaultValue string) (string, error) {
	str, err := configGet(ctx, repoPath, key)
	if err != nil {
		return "", err
	}

	if str == "" {
		err := configSet(ctx, repoPath, key, defaultValue)
		if err != nil {
			return "", err
		}

		return defaultValue, nil
	}

	return str, nil
}

func ConfigGetWithDefault[T any](ctx context.Context, repoPath, key string, converter stringConverter[T], defaultValue T) (T, error) {
	val, err := configGet(ctx, repoPath, key)
	if err != nil {
		return defaultValue, err
	}

	if val == "" {
		configSet(ctx, repoPath, key, converter.ToString(defaultValue))
		return defaultValue, nil
	}

	result, err := converter.FromString(val)
	if err != nil {
		return defaultValue, err
	}

	return result, nil
}

func uint32Converter() stringConverter[uint32] {
	return stringConverter[uint32]{
		ToString: func(d uint32) string { return strconv.FormatUint(uint64(d), 10) },
		FromString: func(s string) (uint32, error) {
			val, err := strconv.ParseUint(s, 10, 32)
			if err != nil {
				return 0, err
			}
			return uint32(val), nil
		},
	}
}

func EmbeddingDimensions(ctx context.Context, repoPath string) uint32 {
	d, err := ConfigGetWithDefault(ctx, repoPath, "dimensions", uint32Converter(), 512)
	if err != nil {
		log.Fatalf("failed to get embedding dimensions: %v", err)
	}

	return d
}

func EmbeddingModel(ctx context.Context, repoPath string) string {
	m, err := ConfigGetWithDefaultString(ctx, repoPath, "model", "text-embedding-3-small")
	if err != nil {
		log.Fatalf("failed to get embedding model: %v", err)
	}

	return m
}

func boolConverter() stringConverter[bool] {
	return stringConverter[bool]{
		ToString:   func(b bool) string { return strconv.FormatBool(b) },
		FromString: func(s string) (bool, error) { return strconv.ParseBool(s) },
	}
}

func WriteNotes(ctx context.Context, repoPath string) bool {
	b, err := ConfigGetWithDefault(ctx, repoPath, "write-notes", boolConverter(), true)
	if err != nil {
		log.Fatalf("failed to get write-notes: %v", err)
	}

	return b
}

// RepoUUID retrieves or generates and sets a UUID at the given git config key.
func RepoUUID(ctx context.Context, repoPath string) uuid.UUID {
	val, err := configGet(ctx, repoPath, "uuid")
	if err != nil {
		log.Fatalf("failed to get repo UUID: %v", err)
	}

	if val != "" {
		id, err := uuid.Parse(val)
		if err != nil {
			log.Fatalf("failed to parse repo UUID: %v", err)
		}
		return id
	}

	id, err := uuid.NewRandom()
	if err != nil {
		log.Fatalf("failed to create repo UUID: %v", err)
	}

	err = configSet(ctx, repoPath, "uuid", id.String())
	if err != nil {
		log.Fatalf("failed to set repo UUID: %v", err)
	}

	return id
}

type Config struct {
	UUID       uuid.UUID
	Model      shared.EmbeddingModel
	Dimensions uint32
	RepoPath   string
	WriteNotes bool
}

func NewConfig(ctx context.Context, repoPath string) Config {
	return Config{
		UUID:       RepoUUID(ctx, repoPath),
		Model:      shared.EmbeddingModelFromString(EmbeddingModel(ctx, repoPath)),
		Dimensions: uint32(EmbeddingDimensions(ctx, repoPath)),
		RepoPath:   repoPath,
		WriteNotes: WriteNotes(ctx, repoPath),
	}
}
