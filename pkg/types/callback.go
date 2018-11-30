package types

import (
	"context"
)

// ErrorCallback is used to inject custom error handling functionality into the application
type ErrorCallback func(ctx context.Context, status int, e error)
