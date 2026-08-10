package applications

import "testing"

func TestParseFlatpakUpdate(t *testing.T) {
	payload := []byte(`{"campaign_id":"ota-1","ref":"org.example.Axon","remote":"factory","commit":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","repository_url":"https://updates.example.test/factory.flatpakrepo"}`)
	update, err := ParseFlatpakUpdate(payload)
	if err != nil {
		t.Fatalf("valid update rejected: %v", err)
	}
	if update.Ref != "org.example.Axon" || update.Remote != "factory" {
		t.Fatalf("unexpected update: %+v", update)
	}
	if update.RepositoryURL == "" {
		t.Fatal("repository URL was not parsed")
	}
}

func TestParseFlatpakUpdateRejectsInsecureRepository(t *testing.T) {
	if _, err := ParseFlatpakUpdate([]byte(`{"campaign_id":"ota-1","ref":"org.example.App","remote":"factory","repository_url":"http://updates.example.test/factory.flatpakrepo"}`)); err == nil {
		t.Fatal("insecure repository accepted")
	}
}

func TestParseFlatpakUpdateRejectsCommandText(t *testing.T) {
	if _, err := ParseFlatpakUpdate([]byte(`{"campaign_id":"ota-1","ref":"org.example.App;reboot","remote":"factory"}`)); err == nil {
		t.Fatal("unsafe reference accepted")
	}
}
