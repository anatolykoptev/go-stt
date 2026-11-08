package stt

import (
	"errors"
	"fmt"
)

// Error represents an STT API error with HTTP status code.
type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	return fmt.Sprintf("stt: HTTP %d: %s", e.StatusCode, e.Message)
}

// IsTransient returns true for errors that may succeed on retry (429, 502-504).
func (e *Error) IsTransient() bool {
	switch e.StatusCode {
	case 429, 502, 503, 504:
		return true
	}
	return false
}

// ErrResponseTooLarge is returned when a downloaded audio file exceeds the cap.
var ErrResponseTooLarge = errors.New("stt: download exceeds size limit")
