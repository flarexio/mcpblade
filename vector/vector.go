package vector

import "context"

type Config struct {
	Enabled    bool   `yaml:"enabled"`
	Persistent bool   `yaml:"persistent"`
	Path       string `yaml:"path"`
	Collection string `yaml:"collection"`
}

type VectorDB interface {
	Collection(name string) (Collection, error)
}

type Collection interface {
	AddDocument(ctx context.Context, doc Document) error
	FindDocument(ctx context.Context, id string) (Document, error)
	Query(ctx context.Context, query string, k int) ([]Document, error)
}

type Document struct {
	ID        string            `json:"id"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Content   string            `json:"content"`
	Embedding []float32         `json:"embedding,omitempty"`
}
