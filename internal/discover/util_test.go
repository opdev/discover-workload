package discover

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// sliceBuffer is an io.Writer to be used for slog in tests.
type sliceBuffer struct {
	mu     sync.Mutex
	stored []string
}

func (b *sliceBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.stored = append(b.stored, string(p))
	return len(p), nil
}

func (b *sliceBuffer) String() string {
	buf := &strings.Builder{}
	for _, v := range b.stored {
		fmt.Fprint(buf, v)
	}

	return buf.String()
}

// NewSlogDiscardLogger returns a logger that does not emit logs. For use in
// tests.
func NewSlogDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
