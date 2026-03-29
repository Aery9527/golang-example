package logs

import (
	"fmt"
	"os"
)

func stderrFallback(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "[LOGS_INTERNAL] %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "[LOGS_INTERNAL] %s\n", msg)
	}
}

func internalWarn(msg string, kvs ...any) {
	fmt.Fprintf(os.Stderr, "[LOGS_INTERNAL_WARN] %s %v\n", msg, kvs)
}
