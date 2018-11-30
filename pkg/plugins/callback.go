package plugins

import (
	"context"

	"bitbucket.org/atlassian/logevent"
)

// LogCallback logs the error using logevent Logger
func LogCallback(ctx context.Context, status int, e error) {
	logger := logevent.FromContext(ctx)
	if status >= 500 {
		logger.Error(e.Error())
		return
	}
	logger.Info(e.Error())
}
