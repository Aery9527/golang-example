package logger

import "log"

// Info logs an informational message.
func Info(msg string) {
	log.Println("[INFO]", msg)
}

// Error logs an error message.
func Error(msg string) {
	log.Println("[ERROR]", msg)
}
