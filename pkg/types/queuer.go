package types

import (
	"time"
)

// Queuer provides an interface for queuing digest jobs onto a streaming appliance
type Queuer interface {
	Queue(id string, start, stop time.Time) error
}
