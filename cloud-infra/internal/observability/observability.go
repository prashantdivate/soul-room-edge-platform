package observability

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func JSONLog(level, message string, fields map[string]any) {
	if fields == nil {
		fields = map[string]any{}
	}
	fields["level"] = level
	fields["message"] = message
	fields["time"] = time.Now().Format(time.RFC3339Nano)
	b, _ := json.Marshal(fields)
	log.Print(string(b))
}

func Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func Metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte("# HELP soul_room_up Service health\n# TYPE soul_room_up gauge\nsoul_room_up 1\n"))
}
