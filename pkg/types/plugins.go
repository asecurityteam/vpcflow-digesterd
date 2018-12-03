package types

// ExitSignals are conditions under which the application should exit and shutdown.
// If the exit path is abnormal, a non-nil error will be sent over the channel. Otherwise, for normal exit conditions, nil will be sent.
type ExitSignals []chan error
