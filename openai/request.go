package openai

import "github.com/olporslon/llmstreamer"

type RequestBody struct {
	Model     Model                 `json:"model"`
	Messages  []llmstreamer.Message `json:"messages"`
	MaxTokens int                   `json:"max_tokens"`
	Stream    bool                  `json:"stream"`
}

type StreamEvent struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	ServiceTier       string   `json:"service_tier,omitempty"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
	Choices           []Choice `json:"choices"`
	Obfuscation       string   `json:"obfuscation,omitempty"`
}

type Choice struct {
	Index        int         `json:"index"`
	Delta        Delta       `json:"delta"`
	Logprobs     interface{} `json:"logprobs"`
	FinishReason *string     `json:"finish_reason"`
}

type Delta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}
