package types

import (
	"context"
	"time"
)

// Queuer provides an interface for queuing digest jobs onto a streaming appliance
type Queuer interface {
	Queue(ctx context.Context, id string, start, stop time.Time) error
}
