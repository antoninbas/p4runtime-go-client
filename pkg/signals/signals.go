package signals

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

var capturedSignals = []os.Signal{syscall.SIGTERM, syscall.SIGINT}

// RegisterSignalHandlers registers a signal handler for capturedSignals and starts a goroutine that
// will block until a signal is received. The first signal received will cause the stopCh channel to
// be closed, giving the opportunity to the program to exist gracefully. If a second signal is
// received before then, we will force exit with code 1.
func RegisterSignalHandlers() <-chan struct{} {
	notifyCh := make(chan os.Signal, 2)
	stopCh := make(chan struct{})

	go func() {
		<-notifyCh
		close(stopCh)
		<-notifyCh
		log.Warning("Received second signal, will force exit")
		os.Exit(1)
	}()

	signal.Notify(notifyCh, capturedSignals...)

	return stopCh
}
