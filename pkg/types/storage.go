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

// Digest is a compacted version of VPC flow logs
type Digest struct {
	// Body the digest log lines in the AWS VPC flow log format
	Body []byte
	// Size of the body
	Size int64
}

// Storage is an interface for accessing created digests
type Storage interface {
	// Get returns the digest for the given key. If download is set to true,
	// the digest contents are downloaded and returned. Otherwise, only the
	// size of the digest is returned.
	Get(key string, download bool) (Digest, error)
}
