package openmetrics

import (
	"fmt"
	"os"
	"runtime"
)

// ErrorHandler handles errors.
type ErrorHandler func(error)

// WarnOnError implements the ErrorHandler interface and writes warnings to
// os.Stderr.
func WarnOnError(err error) {
	if _, file, line, ok := runtime.Caller(2); ok {
		fmt.Fprintf(os.Stderr, "[openmetrics] %s (%s:%d)\n", err, file, line)
		return
	}
	fmt.Fprintf(os.Stderr, "[openmetrics] %s\n", err)
}

// PanicOnError implements the ErrorHandler interface and panics on errors.
func PanicOnError(err error) {
	panic(err)
}
