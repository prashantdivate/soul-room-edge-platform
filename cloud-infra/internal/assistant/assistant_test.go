package assistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSecretBoxAuthenticatesPurposeAndPersists(t *testing.T) {
	path := t.TempDir() + "/ai.key"
	first := New(path, time.Second, 5)
	cipherText, err := first.Seal("provider-secret", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cipherText, "provider-secret") {
		t.Fatal("ciphertext exposed the provider secret")
	}
	second := New(path, time.Second, 5)
	plainText, err := second.Open(cipherText, "tenant-a")
	if err != nil || plainText != "provider-secret" {
		t.Fatalf("persisted key did not decrypt the secret: %q %v", plainText, err)
	}
	if _, err := second.Open(cipherText, "tenant-b"); err == nil {
		t.Fatal("ciphertext opened for another tenant purpose")
	}
}

func TestOpenAIProviderUsesResponsesAPI(t *testing.T) {
	var path, authorization string
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		authorization = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&payload)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"{\"answer\":\"Fleet is healthy.\",\"actions\":[]}"}]}],"usage":{"input_tokens":80,"output_tokens":18}}`))
	}))
	defer server.Close()

	service := New("", 2*time.Second, 5)
	result, err := service.Complete(context.Background(), ProviderConfig{Provider: "openai", BaseURL: server.URL + "/v1", Model: "gpt-5-mini", APIKey: "openai-key"}, CompletionRequest{System: "system", Prompt: "status", MaxTokens: 90})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/v1/responses" || authorization != "Bearer openai-key" {
		t.Fatalf("unexpected OpenAI request: path=%q auth=%q", path, authorization)
	}
	if payload["model"] != "gpt-5-mini" || payload["max_output_tokens"] != float64(90) || payload["store"] != false {
		t.Fatalf("unexpected OpenAI payload: %#v", payload)
	}
	if result.Answer != "Fleet is healthy." || result.Usage.InputTokens != 80 || result.Usage.OutputTokens != 18 {
		t.Fatalf("unexpected OpenAI result: %+v", result)
	}
}

func TestAnthropicProviderUsesMessagesAPI(t *testing.T) {
	var path, apiKey, version string
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		apiKey = r.Header.Get("x-api-key")
		version = r.Header.Get("anthropic-version")
		_ = json.NewDecoder(r.Body).Decode(&payload)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"{\"answer\":\"One device needs attention.\",\"actions\":[]}"}],"usage":{"input_tokens":75,"output_tokens":21}}`))
	}))
	defer server.Close()

	service := New("", 2*time.Second, 5)
	result, err := service.Complete(context.Background(), ProviderConfig{Provider: "anthropic", BaseURL: server.URL, Model: "claude-haiku-5-5", APIKey: "anthropic-key"}, CompletionRequest{System: "system", Prompt: "status", MaxTokens: 90})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/v1/messages" || apiKey != "anthropic-key" || version != "2023-06-01" {
		t.Fatalf("unexpected Anthropic request: path=%q key=%q version=%q", path, apiKey, version)
	}
	if payload["model"] != "claude-haiku-5-5" || payload["max_tokens"] != float64(90) {
		t.Fatalf("unexpected Anthropic payload: %#v", payload)
	}
	if result.Answer != "One device needs attention." || result.Usage.InputTokens != 75 || result.Usage.OutputTokens != 21 {
		t.Fatalf("unexpected Anthropic result: %+v", result)
	}
}

func TestCompatibleProviderParsesStructuredResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer local-key" {
			t.Fatalf("unexpected request: %s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"answer\":\"Two devices are offline.\",\"actions\":[],\"follow_up_prompts\":[\"Show them\"]}"}}],"usage":{"prompt_tokens":120,"completion_tokens":25}}`))
	}))
	defer server.Close()

	service := New("", 2*time.Second, 5)
	result, err := service.Complete(context.Background(), ProviderConfig{Provider: "compatible", BaseURL: server.URL + "/v1", Model: "small-local", APIKey: "local-key"}, CompletionRequest{System: "system", Prompt: "status", MaxTokens: 100})
	if err != nil {
		t.Fatal(err)
	}
	if result.Answer != "Two devices are offline." || result.Usage.InputTokens != 120 || len(result.FollowUpPrompts) != 1 {
		t.Fatalf("unexpected completion: %+v", result)
	}
}

func TestProviderValidationRejectsMetadataAddress(t *testing.T) {
	err := ValidateConfiguration(ProviderConfig{Provider: "compatible", BaseURL: "http://169.254.169.254/v1", Model: "local"})
	if err == nil {
		t.Fatal("metadata service address was accepted")
	}
}

func TestRateLimitKeepsInteractiveHeadroom(t *testing.T) {
	service := New("", time.Second, 2)
	if !service.Allow("tenant:user") || !service.Allow("tenant:user") || service.Allow("tenant:user") {
		t.Fatal("per-user request limit was not enforced")
	}
	if !service.Allow("tenant:other-user") {
		t.Fatal("one user exhausted another user's request allowance")
	}
}
