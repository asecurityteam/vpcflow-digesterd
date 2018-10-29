package types

import (
	"context"

	"bitbucket.org/atlassian/httplog"
	"bitbucket.org/atlassian/logevent"
)

// LoggerProvider extracts a logger from context.
type LoggerProvider func(ctx context.Context) logevent.Logger

// LogEventProvider extracts a log event from the context, pre-populated with request metadata
type LogEventProvider func(ctx context.Context) httplog.Event
