package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/olporslon/llmstreamer"
)

type AnthropicStreamer struct {
	ApiKey string
	Model  Model
}

func New(apiKey string, model Model) *AnthropicStreamer {
	return &AnthropicStreamer{
		ApiKey: apiKey,
		Model:  model,
	}
}

const url = "https://api.anthropic.com/v1/messages"

func (s *AnthropicStreamer) StreamChat(
	ctx context.Context,
	messages []llmstreamer.Message,
	cb *llmstreamer.StreamCallbacks,
) {
	if s.ApiKey == "" {
		if cb != nil && cb.OnError != nil {
			err := errors.New("invalid apiKey")
			cb.OnError(err)
		}
		return
	}

	model := s.Model
	if model == "" {
		model = ModelClaude3Opus
	}

	payload := RequestBody{
		Model:     model,
		Messages:  messages,
		MaxTokens: 1024,
		Stream:    true,
	}

	if err := streamAnthropic(ctx, payload, s.ApiKey, cb); err != nil {
		if cb != nil && cb.OnError != nil {
			cb.OnError(err)
		}
	}
}

func streamAnthropic(ctx context.Context, payload RequestBody, apiKey string, cb *llmstreamer.StreamCallbacks) error {
	client, req, err := prepareRequest(ctx, payload, apiKey)

	if err != nil {
		return err
	}

	if client == nil || req == nil {
		return errors.New("invalid client or request")
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	processStream(resp, cb)
	return nil
}

func prepareRequest(ctx context.Context, payload RequestBody, apiKey string) (*http.Client, *http.Request, error) {
	data, err := json.Marshal(payload)

	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))

	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{
		Timeout: 0,
	}

	return client, req, nil
}

func processStream(resp *http.Response, cb *llmstreamer.StreamCallbacks) {
	if resp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			cb.OnError(fmt.Errorf("non-200: %d, read body failed: %w", resp.StatusCode, err))
			return
		}
		cb.OnError(fmt.Errorf("non-200: %d, body: %s", resp.StatusCode, string(b)))
		return
	}

	reader := bufio.NewReader(resp.Body)

	var finalMessage string

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				cb.OnFinish(finalMessage)
				return
			}
			cb.OnError(fmt.Errorf("read failed: %w", err))
			return
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if bytes.HasPrefix(line, []byte("data: ")) {
			data := line[len("data: "):]

			var ev StreamEvent
			if err := json.Unmarshal(data, &ev); err != nil {
				cb.OnError(fmt.Errorf("failed to parse JSON: %w", err))
				continue
			}

			switch ev.Type {
			case Delta:
				if ev.Delta != nil && ev.Delta.Text != "" {
					if cb != nil && cb.OnContent != nil {
						finalMessage += ev.Delta.Text
						cb.OnContent(ev.Delta.Text)
					}
				}
			case Finish:
				if cb != nil && cb.OnFinish != nil {
					cb.OnFinish(finalMessage)
					return
				}
			default:
				// Ignore other event types for now
				// fmt.Printf("[unknown type: %s]\n", ev.Type)
			}
		}
	}
}
