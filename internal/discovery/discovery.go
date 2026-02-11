package discovery

import (
	"context"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/schema"
)

// Discoverer discovers the schema of a source database.
type Discoverer interface {
	// Connect establishes a read-only connection to the source database.
	Connect(ctx context.Context) error

	// Discover extracts the full schema from the source database.
	Discover(ctx context.Context) (*schema.Schema, error)

	// Close closes the database connection.
	Close() error
}

// New creates a Discoverer for the given source configuration.
func New(cfg *config.SourceConfig) (Discoverer, error) {
	switch cfg.Type {
	case "postgresql":
		return NewPostgres(cfg)
	case "oracle":
		return NewOracle(cfg)
	default:
		return nil, &UnsupportedDBError{DBType: cfg.Type}
	}
}

// UnsupportedDBError is returned when the source DB type is not supported.
type UnsupportedDBError struct {
	DBType string
}

func (e *UnsupportedDBError) Error() string {
	return "unsupported database type: " + e.DBType
}
