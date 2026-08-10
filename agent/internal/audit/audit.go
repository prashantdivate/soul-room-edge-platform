package audit

import (
	"encoding/json"
	"log"
	"time"
)

type Event struct {
	Time   time.Time         `json:"time"`
	Name   string            `json:"name"`
	Fields map[string]string `json:"fields,omitempty"`
}

func Log(name string, fields map[string]string) {
	b, _ := json.Marshal(Event{Time: time.Now(), Name: name, Fields: fields})
	log.Printf("audit=%s", string(b))
}
