package anthropic

import "github.com/olporslon/llmstreamer"

type RequestBody struct {
	Model     Model                 `json:"model"`
	Messages  []llmstreamer.Message `json:"messages"`
	MaxTokens int                   `json:"max_tokens"`
	Stream    bool                  `json:"stream"`
}

type Type string

const (
	Start        Type = "message_start"
	ContentStart Type = "content_block_start"
	Delta        Type = "content_block_delta"
	Stop         Type = "content_block_stop"
	Finish       Type = "message_stop"
)

type StreamEvent struct {
	Type  Type       `json:"type"`
	Index int        `json:"index"`
	Delta *DeltaData `json:"delta,omitempty"`
}

type DeltaData struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
