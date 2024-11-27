package discover

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	notificationText = "discovery still running"
	shutdownText     = "notifier shutting down"
)

// StartNotifier peeriodically logs a message to logger. As it is possible that
// workloads may spin up and down at random intervals, we want the user to know
// that the tool is still running.
func StartNotifier(
	ctx context.Context,
	logger *slog.Logger,
	interval time.Duration,
	startAfter time.Duration,
) {
	defer logger.Debug(shutdownText)
	ticker := time.NewTicker(startAfter)
	done := make(chan bool)
	// canceledEarly tracks if the context finished before
	// we could start periodically notifying. If so, we
	// don't spin up the interval ticker and just finish up
	canceledEarly := false

	select {
	case <-ticker.C:
		logger.Info(notificationText)
		ticker.Reset(interval)
	case <-ctx.Done():
		canceledEarly = true
	}

	var wg sync.WaitGroup
	if !canceledEarly {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					logger.Info(notificationText)
				}
			}
		}()
	}

	<-ctx.Done()
	// context is done, start tear down.
	close(done)
	ticker.Stop()
	wg.Wait()
}
