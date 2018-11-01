package types

import (
	"context"
	"fmt"
	"io"
)

// InProgress indicates that a digest is in the process of being created
type InProgress struct {
	Key string
}

func (e InProgress) Error() string {
	return fmt.Sprintf("digest %s is being created", e.Key)
}

// NotFound represents a resource lookup that failed due to a missing record.
type NotFound struct {
	ID string
}

func (e NotFound) Error() string {
	return fmt.Sprintf("digest %s was not found", e.ID)
}

// Storage is an interface for accessing created digests. It is the caller's responsibility to call Close on the Reader when done.
type Storage interface {
	// Get returns the digest for the given key.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Exists returns true if the digest exists, but does not download the digest body.
	Exists(ctx context.Context, key string) (bool, error)

	// Store stores the digest
	Store(ctx context.Context, key string, data io.ReadCloser) error
}
