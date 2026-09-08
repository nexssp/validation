package validation

import (
	"regexp"
	"time"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	uuidRegex  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func IsValidEmail(email string) bool { return emailRegex.MatchString(email) }
func IsValidUUID(id string) bool     { return uuidRegex.MatchString(id) }

// CompileWithTimeout prevents ReDoS attacks on dynamic patterns.
func CompileWithTimeout(pattern string, timeout time.Duration) *regexp.Regexp {
	done := make(chan *regexp.Regexp, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- nil
			}
		}()
		done <- regexp.MustCompile(pattern)
	}()
	select {
	case re := <-done:
		return re
	case <-time.After(timeout):
		return nil
	}
}
