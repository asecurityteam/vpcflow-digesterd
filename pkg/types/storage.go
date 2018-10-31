package types

import (
	"fmt"
)

// ErrInProgress indicates that a digest is in the process of being created
type ErrInProgress struct {
	Key string
}

func (e *ErrInProgress) Error() string {
	return fmt.Sprintf("digest %s is being created", e.Key)
}

// Storage is an interface for accessing created digests
type Storage interface {
	// Get returns the digest for the given key.
	Get(key string) ([]byte, error)

	// Exists returns true if the digest exists, but does not download the digest body.
	Exists(key string) (bool, error)
}
