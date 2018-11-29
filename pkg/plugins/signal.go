package plugins

import (
	"os"
	"os/signal"
	"syscall"
)

// ExitSignals are conditions under which the application should exit and shutdown.
// If the exit path is abnormal, a non-nil error will be sent over the channel. Otherwise, for normal exit conditions, nil will be sent.
type ExitSignals []chan error

// OS listens for OS signals SIGINT and SIGTERM and writes to the channel if either of those scenarios are encountered, nil is sent over the channel.
func OS() chan error {
	c := make(chan os.Signal)
	exit := make(chan error)
	go func() {
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		exit <- nil
	}()
	return exit
}
