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
	Get(key string) ([]byte, error)
}
