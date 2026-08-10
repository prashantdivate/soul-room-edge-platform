package containers

import "testing"

func TestComposePolicyRejectsUnsafe(t *testing.T) {
	raw := []byte(`{"services":{"bad":{"image":"x","managed":true,"privileged":true}}}`)
	if err := ValidateCompose(raw, Policy{}); err == nil {
		t.Fatal("expected privileged rejection")
	}
	raw = []byte(`{"services":{"ok":{"image":"x","managed":true,"volumes":["/var/lib/edge-agent/data:/data"]}}}`)
	if err := ValidateCompose(raw, Policy{AllowedHostPathMounts: []string{"/var/lib/edge-agent"}}); err != nil {
		t.Fatal(err)
	}
}
