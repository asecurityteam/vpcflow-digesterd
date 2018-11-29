package plugins

import (
	"os"
	"os/signal"
	"syscall"
)

// SignalHandler is a function which accepts a channel as input, and will write to the channel when an exit condition is encountered.
// If a fatal error is encountered a non-nil error should be written. Otherwise for normal exit cases, nil should be sent to the channel.
type SignalHandler func(exit chan error)

// OSHandler listens for OS signals SIGINT and SIGTERM and writes to the channel if either of those scenarios are encountered, nil is sent over the channel.
func OSHandler(exit chan error) {
	var c = make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	exit <- nil
}
