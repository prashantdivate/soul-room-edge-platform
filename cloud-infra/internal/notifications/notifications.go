package notifications

import "errors"

type Message struct {
	TenantID string            `json:"tenant_id"`
	Type     string            `json:"type"`
	Target   string            `json:"target"`
	Body     string            `json:"body"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func Validate(message Message) error {
	if message.TenantID == "" || message.Type == "" || message.Target == "" {
		return errors.New("tenant, type, and target are required")
	}
	return nil
}
