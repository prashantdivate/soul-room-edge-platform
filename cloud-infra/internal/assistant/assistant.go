package assistant

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const maxProviderResponse = 1024 * 1024

type ProviderConfig struct {
	Provider string
	BaseURL  string
	Model    string
	APIKey   string
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionRequest struct {
	System    string
	Prompt    string
	History   []Message
	MaxTokens int
}

type Usage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
}

type ProposedAction struct {
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Rationale string `json:"rationale"`
	Risk      string `json:"risk"`
	DeviceID  string `json:"device_id,omitempty"`
	JobType   string `json:"job_type,omitempty"`
	Page      string `json:"page,omitempty"`
}

type Result struct {
	Answer          string           `json:"answer"`
	Actions         []ProposedAction `json:"actions,omitempty"`
	FollowUpPrompts []string         `json:"follow_up_prompts,omitempty"`
	Usage           Usage            `json:"usage,omitempty"`
}

type Service struct {
	box          *secretBox
	initErr      error
	client       *http.Client
	limit        int
	mu           sync.Mutex
	requestTimes map[string][]time.Time
}

func New(keyPath string, timeout time.Duration, requestsPerMinute int) *Service {
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	if requestsPerMinute <= 0 {
		requestsPerMinute = 30
	}
	box, err := loadSecretBox(keyPath)
	return &Service{
		box:          box,
		initErr:      err,
		client:       safeHTTPClient(timeout),
		limit:        requestsPerMinute,
		requestTimes: map[string][]time.Time{},
	}
}

func (s *Service) Ready() error {
	return s.initErr
}

func (s *Service) Allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	recent := s.requestTimes[key][:0]
	for _, stamp := range s.requestTimes[key] {
		if stamp.After(cutoff) {
			recent = append(recent, stamp)
		}
	}
	if len(recent) >= s.limit {
		s.requestTimes[key] = recent
		return false
	}
	s.requestTimes[key] = append(recent, now)
	return true
}

func (s *Service) Seal(plainText, purpose string) (string, error) {
	if s.initErr != nil {
		return "", s.initErr
	}
	return s.box.seal([]byte(plainText), []byte(purpose))
}

func (s *Service) Open(cipherText, purpose string) (string, error) {
	if s.initErr != nil {
		return "", s.initErr
	}
	plainText, err := s.box.open(cipherText, []byte(purpose))
	return string(plainText), err
}

func (s *Service) SealJSON(value any, purpose string) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return s.Seal(string(payload), purpose)
}

func (s *Service) OpenJSON(cipherText, purpose string, value any) error {
	payload, err := s.Open(cipherText, purpose)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(payload), value)
}

func (s *Service) Complete(ctx context.Context, config ProviderConfig, request CompletionRequest) (Result, error) {
	if err := ValidateConfiguration(config); err != nil {
		return Result{}, err
	}
	if request.MaxTokens <= 0 || request.MaxTokens > 4096 {
		request.MaxTokens = 1200
	}

	var text string
	var usage Usage
	var err error
	switch config.Provider {
	case "openai":
		text, usage, err = s.openAI(ctx, config, request)
	case "anthropic":
		text, usage, err = s.anthropic(ctx, config, request)
	case "compatible":
		text, usage, err = s.compatible(ctx, config, request)
	default:
		err = errors.New("unsupported AI provider")
	}
	if err != nil {
		return Result{}, err
	}
	result := parseResult(text)
	result.Usage = usage
	return result, nil
}

func ValidateConfiguration(config ProviderConfig) error {
	if config.Provider != "openai" && config.Provider != "anthropic" && config.Provider != "compatible" {
		return errors.New("provider must be openai, anthropic, or compatible")
	}
	if strings.TrimSpace(config.Model) == "" || len(config.Model) > 128 {
		return errors.New("a valid model name is required")
	}
	if config.Provider != "compatible" && strings.TrimSpace(config.APIKey) == "" {
		return errors.New("an API key is required for this provider")
	}
	return validateBaseURL(config.BaseURL)
}

func validateBaseURL(raw string) error {
	if len(raw) > 512 {
		return errors.New("provider URL is too long")
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("provider URL must be an HTTP or HTTPS URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("provider URL cannot contain credentials, a query, or a fragment")
	}
	if ip := net.ParseIP(parsed.Hostname()); ip != nil && blockedIP(ip) {
		return errors.New("provider URL resolves to a prohibited network address")
	}
	return nil
}

func (s *Service) openAI(ctx context.Context, config ProviderConfig, request CompletionRequest) (string, Usage, error) {
	body := map[string]any{
		"model":             config.Model,
		"instructions":      request.System,
		"input":             conversationText(request),
		"max_output_tokens": request.MaxTokens,
		"store":             false,
	}
	var response struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	err := s.doJSON(ctx, strings.TrimRight(config.BaseURL, "/")+"/responses", body, map[string]string{"Authorization": "Bearer " + config.APIKey}, &response)
	if err != nil {
		return "", Usage{}, err
	}
	text := response.OutputText
	if text == "" {
		for _, item := range response.Output {
			for _, content := range item.Content {
				if content.Type == "output_text" || content.Type == "text" {
					text += content.Text
				}
			}
		}
	}
	if strings.TrimSpace(text) == "" {
		return "", Usage{}, errors.New("AI provider returned an empty response")
	}
	return text, Usage{InputTokens: response.Usage.InputTokens, OutputTokens: response.Usage.OutputTokens}, nil
}

func (s *Service) anthropic(ctx context.Context, config ProviderConfig, request CompletionRequest) (string, Usage, error) {
	messages := make([]map[string]string, 0, len(request.History)+1)
	for _, message := range request.History {
		messages = append(messages, map[string]string{"role": message.Role, "content": message.Content})
	}
	messages = append(messages, map[string]string{"role": "user", "content": request.Prompt})
	body := map[string]any{"model": config.Model, "max_tokens": request.MaxTokens, "system": request.System, "messages": messages}
	var response struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	headers := map[string]string{"x-api-key": config.APIKey, "anthropic-version": "2023-06-01"}
	err := s.doJSON(ctx, strings.TrimRight(config.BaseURL, "/")+"/v1/messages", body, headers, &response)
	if err != nil {
		return "", Usage{}, err
	}
	var text strings.Builder
	for _, content := range response.Content {
		if content.Type == "text" {
			text.WriteString(content.Text)
		}
	}
	if strings.TrimSpace(text.String()) == "" {
		return "", Usage{}, errors.New("AI provider returned an empty response")
	}
	return text.String(), Usage{InputTokens: response.Usage.InputTokens, OutputTokens: response.Usage.OutputTokens}, nil
}

func (s *Service) compatible(ctx context.Context, config ProviderConfig, request CompletionRequest) (string, Usage, error) {
	messages := []map[string]string{{"role": "system", "content": request.System}}
	for _, message := range request.History {
		messages = append(messages, map[string]string{"role": message.Role, "content": message.Content})
	}
	messages = append(messages, map[string]string{"role": "user", "content": request.Prompt})
	body := map[string]any{"model": config.Model, "messages": messages, "max_tokens": request.MaxTokens, "temperature": 0.2}
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	headers := map[string]string{}
	if config.APIKey != "" {
		headers["Authorization"] = "Bearer " + config.APIKey
	}
	err := s.doJSON(ctx, strings.TrimRight(config.BaseURL, "/")+"/chat/completions", body, headers, &response)
	if err != nil {
		return "", Usage{}, err
	}
	if len(response.Choices) == 0 || strings.TrimSpace(response.Choices[0].Message.Content) == "" {
		return "", Usage{}, errors.New("AI provider returned an empty response")
	}
	return response.Choices[0].Message.Content, Usage{InputTokens: response.Usage.PromptTokens, OutputTokens: response.Usage.CompletionTokens}, nil
}

func (s *Service) doJSON(ctx context.Context, endpoint string, payload any, headers map[string]string, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Soul-Room-AI-Assistant/1.0")
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("AI provider request failed: %w", err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxProviderResponse+1))
	if err != nil {
		return errors.New("failed to read AI provider response")
	}
	if len(raw) > maxProviderResponse {
		return errors.New("AI provider response exceeded the size limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var problem struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(raw, &problem)
		message := bounded(problem.Error.Message, 240)
		if message == "" {
			message = http.StatusText(response.StatusCode)
		}
		return fmt.Errorf("AI provider returned HTTP %d: %s", response.StatusCode, message)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return errors.New("AI provider returned invalid JSON")
	}
	return nil
}

func conversationText(request CompletionRequest) string {
	var text strings.Builder
	for _, message := range request.History {
		text.WriteString(strings.ToUpper(message.Role))
		text.WriteString(": ")
		text.WriteString(message.Content)
		text.WriteString("\n\n")
	}
	text.WriteString("USER: ")
	text.WriteString(request.Prompt)
	return text.String()
}

func parseResult(raw string) Result {
	trimmed := strings.TrimSpace(raw)
	start, end := strings.Index(trimmed, "{"), strings.LastIndex(trimmed, "}")
	var result Result
	if start >= 0 && end > start && json.Unmarshal([]byte(trimmed[start:end+1]), &result) == nil && strings.TrimSpace(result.Answer) != "" {
		result.Answer = bounded(strings.TrimSpace(result.Answer), 12000)
		if len(result.Actions) > 4 {
			result.Actions = result.Actions[:4]
		}
		if len(result.FollowUpPrompts) > 4 {
			result.FollowUpPrompts = result.FollowUpPrompts[:4]
		}
		return result
	}
	return Result{Answer: bounded(trimmed, 12000)}
}

func bounded(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

type secretBox struct {
	aead cipher.AEAD
}

func loadSecretBox(path string) (*secretBox, error) {
	key := make([]byte, 32)
	if path == "" {
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		return newSecretBox(key)
	}
	stored, err := os.ReadFile(path)
	if err == nil {
		if len(stored) != 32 {
			return nil, errors.New("AI secret key must contain exactly 32 bytes")
		}
		return newSecretBox(stored)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, os.ErrExist) {
		stored, readErr := os.ReadFile(path)
		if readErr != nil || len(stored) != 32 {
			return nil, errors.New("failed to load concurrently created AI secret key")
		}
		return newSecretBox(stored)
	}
	if err != nil {
		return nil, err
	}
	if _, err = file.Write(key); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	_ = os.Chmod(path, 0600)
	return newSecretBox(key)
}

func newSecretBox(key []byte) (*secretBox, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &secretBox{aead: aead}, nil
}

func (b *secretBox) seal(plainText, additionalData []byte) (string, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	cipherText := b.aead.Seal(nonce, nonce, plainText, additionalData)
	return base64.RawURLEncoding.EncodeToString(cipherText), nil
}

func (b *secretBox) open(encoded string, additionalData []byte) ([]byte, error) {
	cipherText, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(cipherText) < b.aead.NonceSize() {
		return nil, errors.New("invalid encrypted value")
	}
	nonce, payload := cipherText[:b.aead.NonceSize()], cipherText[b.aead.NonceSize():]
	plainText, err := b.aead.Open(nil, nonce, payload, additionalData)
	if err != nil {
		return nil, errors.New("encrypted value could not be authenticated")
	}
	return plainText, nil
}

func safeHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           safeDialContext(dialer),
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: timeout,
		IdleConnTimeout:       30 * time.Second,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("AI provider redirects are disabled")
		},
	}
}

func safeDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil || len(addresses) == 0 {
			return nil, errors.New("AI provider hostname could not be resolved")
		}
		for _, address := range addresses {
			if blockedIP(address.IP) {
				return nil, errors.New("AI provider resolved to a prohibited network address")
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].IP.String(), port))
	}
}

func blockedIP(ip net.IP) bool {
	return ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}
