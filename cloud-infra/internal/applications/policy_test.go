package applications

import (
	"encoding/json"
	"testing"
)

func TestComposePolicyRejectsUnsafeManifest(t *testing.T) {
	raw := json.RawMessage(`{"services":{"bad":{"image":"repo/app:1","managed":true,"privileged":true}}}`)
	if err := ValidateCompose(raw, false); err == nil {
		t.Fatal("expected privileged manifest rejection")
	}
	raw = json.RawMessage(`{"services":{"ok":{"image":"repo/app:1","managed":true,"volumes":["/var/lib/soul-room/managed/app:/data"]}}}`)
	if err := ValidateCompose(raw, false); err != nil {
		t.Fatal(err)
	}
}
