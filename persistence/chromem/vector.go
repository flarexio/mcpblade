package chromem

import (
	"context"

	"github.com/philippgille/chromem-go"

	"github.com/flarexio/mcpblade/vector"
)

func NewChromemVectorDB(cfg vector.Config) (vector.VectorDB, error) {
	var db *chromem.DB
	if !cfg.Persistent {
		db = chromem.NewDB()
	} else {
		d, err := chromem.NewPersistentDB(cfg.Path, false)
		if err != nil {
			return nil, err
		}

		db = d
	}

	return &chromemVectorDB{db}, nil
}

type chromemVectorDB struct {
	db *chromem.DB
}

func (vector *chromemVectorDB) Collection(name string) (vector.Collection, error) {
	// TODO: need to support other embedding models
	c, err := vector.db.GetOrCreateCollection(name, nil, nil)
	if err != nil {
		return nil, err
	}

	return &collection{c}, nil
}

type collection struct {
	collection *chromem.Collection
}

func (c *collection) AddDocument(ctx context.Context, doc vector.Document) error {
	document := chromem.Document{
		ID:        doc.ID,
		Metadata:  doc.Metadata,
		Embedding: doc.Embedding,
		Content:   doc.Content,
	}

	return c.collection.AddDocument(ctx, document)
}

func (c *collection) FindDocument(ctx context.Context, id string) (vector.Document, error) {
	document, err := c.collection.GetByID(ctx, id)
	if err != nil {
		return vector.Document{}, err
	}

	return vector.Document{
		ID:        document.ID,
		Metadata:  document.Metadata,
		Embedding: document.Embedding,
		Content:   document.Content,
	}, nil
}

func (c *collection) Query(ctx context.Context, query string, k int) ([]vector.Document, error) {
	if k > c.collection.Count() {
		k = c.collection.Count()
	}

	results, err := c.collection.Query(ctx, query, k, nil, nil)
	if err != nil {
		return nil, err
	}

	docs := make([]vector.Document, len(results))
	for i, result := range results {
		docs[i] = vector.Document{
			ID:        result.ID,
			Metadata:  result.Metadata,
			Embedding: result.Embedding,
			Content:   result.Content,
		}
	}

	return docs, nil
}
