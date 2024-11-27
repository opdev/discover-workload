package apperrors

import (
	"context"
	"errors"
)

// IsTimeout
func IsTimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}
