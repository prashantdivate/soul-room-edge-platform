package observability

import "log"

func Info(msg string, fields ...any) {
	log.Printf("level=info msg=%q fields=%v", msg, fields)
}

func Error(msg string, fields ...any) {
	log.Printf("level=error msg=%q fields=%v", msg, fields)
}
