package discover

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestStartNotifier_NotificationCountGivenTimespan(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Millisecond)
	defer cancel()

	logBuffer := &sliceBuffer{}
	logger := slog.New(slog.NewTextHandler(logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug}))

	StartNotifier(ctx, logger, 2*time.Millisecond, 2*time.Millisecond)

	minWanted := 2 // number of emitted logs, not including shutdown
	if len(logBuffer.stored) < minWanted {
		t.Logf("log contents\n%s", logBuffer.String())
		t.Errorf(
			"Did not see mininum expected number of log lines given the time frame and interval values.\nGot: %d\nWant: %d\n",
			len(logBuffer.stored),
			minWanted,
		)
	}

	// This case may need to change  should the surrounding logs be updated to
	// meet future needs.
	var count int
	for _, line := range logBuffer.stored {
		if strings.Contains(line, notificationText) {
			count++
		}
	}

	if count != minWanted {
		t.Logf("expected notification message: \"%s\"", notificationText)
		t.Logf("log contents\n%s", logBuffer.String())
		t.Errorf(
			"Did not find exact number of notification messages.\nGot: %d\nWant: %d\n",
			count,
			minWanted,
		)
	}
}

func TestStartNotifier_EarlyCancelation(t *testing.T) {
	// a shorter timeout than the start point guarantees a cancelation before
	// any notifications will be emitted.
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Millisecond)
	defer cancel()

	logBuffer := &sliceBuffer{}
	logger := slog.New(slog.NewTextHandler(logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug}))

	StartNotifier(ctx, logger, 2*time.Millisecond, 2*time.Millisecond)
	expectedNotificationCount := 0
	var actualNotificationCount int
	for _, line := range logBuffer.stored {
		if strings.Contains(line, notificationText) {
			actualNotificationCount++
		}
	}

	if actualNotificationCount > expectedNotificationCount {
		t.Logf("log contents\n%s", logBuffer.String())
		t.Errorf(
			"Found too many notifications emitted by the notifier\nGot: %d\nWant: %d\n",
			actualNotificationCount,
			expectedNotificationCount,
		)
	}
}
