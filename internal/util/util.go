package util

import (
	"log"
)

// Assert fails if the condition is false, printing the formatted message and exiting.
func Assert(condition bool, format string, args ...any) {
	if !condition {
		log.Fatalf(format, args...)
	}
}
