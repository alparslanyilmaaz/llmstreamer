package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/alparslanyilmaaz/llmstreamer"
)

type errReadCloser struct{}

func (errReadCloser) Read(p []byte) (int, error) { return 0, errors.New("boom") }
func (errReadCloser) Close() error               { return nil }

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestNew(t *testing.T) {
	s := New("my-key", ModelClaude3Opus)
	if s == nil {
		t.Fatalf("New returned nil")
	}
	if s.ApiKey != "my-key" {
		t.Fatalf("expected ApiKey 'my-key', got %q", s.ApiKey)
	}
	if s.Model != ModelClaude3Opus {
		t.Fatalf("expected Model %v, got %v", ModelClaude3Opus, s.Model)
	}
}

func TestStreamChat_InvalidApiKeyCallsOnError(t *testing.T) {
	s := New("", "")

	var gotErr error
	cb := &llmstreamer.StreamCallbacks{
		OnError: func(err error) { gotErr = err },
	}

	s.StreamChat(context.Background(), nil, cb)

	if gotErr == nil {
		t.Fatalf("expected OnError to be called when ApiKey is empty")
	}
}

func TestStreamChat_DefaultModel(t *testing.T) {
	s := New("test-key", "")

	orig := http.DefaultTransport
	defer func() { http.DefaultTransport = orig }()

	body := "" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"ok\"}}\n" +
		"data: {\"type\":\"message_stop\"}\n"

	http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(req.Body)
		var p RequestBody
		if err := json.Unmarshal(b, &p); err != nil {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("bad"))}, nil
		}
		if p.Model != ModelClaude3Opus {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("bad model"))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})

	var final string
	cb := &llmstreamer.StreamCallbacks{
		OnContent: func(s string) {},
		OnFinish:  func(f string) { final = f },
		OnError:   func(err error) { t.Fatalf("unexpected error: %v", err) },
	}

	s.StreamChat(context.Background(), nil, cb)

	if final != "ok" {
		t.Fatalf("expected final 'ok', got %q", final)
	}
}

func TestStreamChat_TransportError(t *testing.T) {
	s := New("test-key", "")

	orig := http.DefaultTransport
	defer func() { http.DefaultTransport = orig }()

	http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("transport failure")
	})

	var gotErr error
	cb := &llmstreamer.StreamCallbacks{
		OnError: func(err error) { gotErr = err },
	}

	s.StreamChat(context.Background(), nil, cb)

	if gotErr == nil {
		t.Fatalf("expected OnError due to transport error")
	}
}

func TestStreamAnthropic_Success(t *testing.T) {
	payload := RequestBody{
		Model:     ModelClaude3Opus,
		Messages:  []llmstreamer.Message{{Role: llmstreamer.RoleUser, Content: "hello"}},
		MaxTokens: 5,
		Stream:    true,
	}

	apiKey := "test-key"

	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()

	body := "" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"Hi\"}}\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\" there\"}}\n" +
		"data: {\"type\":\"message_stop\"}\n"

	http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			return &http.Response{StatusCode: 405, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})

	var called bool
	cb := &llmstreamer.StreamCallbacks{
		OnContent: func(s string) { called = true },
		OnFinish:  func(s string) { called = true },
		OnError:   func(err error) { t.Fatalf("unexpected OnError: %v", err) },
	}

	err := streamAnthropic(context.Background(), payload, apiKey, cb)
	if err != nil {
		t.Fatalf("streamAnthropic returned error: %v", err)
	}
	if !called {
		t.Fatalf("expected at least one callback to be called")
	}
}

func TestPrepareRequest_Success(t *testing.T) {
	payload := RequestBody{
		Model:     ModelClaude3Opus,
		Messages:  []llmstreamer.Message{{Role: llmstreamer.RoleUser, Content: "hello"}},
		MaxTokens: 5,
		Stream:    true,
	}

	apiKey := "test-key"
	client, req, err := prepareRequest(context.Background(), payload, apiKey)
	if err != nil {
		t.Fatalf("prepareRequest returned error: %v", err)
	}
	if client == nil {
		t.Fatalf("expected non-nil client")
	}
	if req == nil {
		t.Fatalf("expected non-nil request")
	}

	if req.Method != http.MethodPost {
		t.Fatalf("expected POST method, got %s", req.Method)
	}
	if req.URL == nil || req.URL.String() != url {
		t.Fatalf("expected URL %s, got %v", url, req.URL)
	}

	if got := req.Header.Get("x-api-key"); got != apiKey {
		t.Fatalf("expected x-api-key %q, got %q", apiKey, got)
	}
	if ct := req.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
	if v := req.Header.Get("anthropic-version"); v == "" {
		t.Fatalf("expected anthropic-version header set")
	}

	if client.Timeout != 0 {
		t.Fatalf("expected client.Timeout 0, got %v", client.Timeout)
	}

	b, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("reading request body failed: %v", err)
	}
	if c, ok := req.Body.(io.Closer); ok {
		c.Close()
	}

	var got RequestBody
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal of request body failed: %v", err)
	}
	if got.Model != payload.Model {
		t.Fatalf("model mismatch: expected %v got %v", payload.Model, got.Model)
	}
	if len(got.Messages) != len(payload.Messages) || got.Messages[0].Content != payload.Messages[0].Content {
		t.Fatalf("messages mismatch: expected %+v got %+v", payload.Messages, got.Messages)
	}
}

func TestProcessStream_DeltaFinish(t *testing.T) {
	body := "" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"Hello\"}}\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\" world\"}}\n" +
		"data: {\"type\":\"message_stop\"}\n"

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	var contents []string
	var final string

	cb := &llmstreamer.StreamCallbacks{
		OnContent: func(s string) {
			contents = append(contents, s)
		},
		OnFinish: func(f string) {
			final = f
		},
		OnError: func(err error) {
			t.Fatalf("OnError called: %v", err)
		},
	}

	processStream(resp, cb)

	if len(contents) != 2 {
		t.Fatalf("expected 2 content chunks, got %d: %v", len(contents), contents)
	}
	if contents[0] != "Hello" || contents[1] != " world" {
		t.Fatalf("unexpected contents: %v", contents)
	}
	if final != "Hello world" {
		t.Fatalf("unexpected final message: %q", final)
	}
}

func TestProcessStream_Non200(t *testing.T) {
	resp := &http.Response{
		StatusCode: 400,
		Body:       io.NopCloser(strings.NewReader("bad request")),
	}

	var gotErr error
	cb := &llmstreamer.StreamCallbacks{
		OnError: func(err error) {
			gotErr = err
		},
	}

	processStream(resp, cb)

	if gotErr == nil {
		t.Fatalf("expected an error for non-200 response")
	}
	if !strings.Contains(gotErr.Error(), "non-200") {
		t.Fatalf("error message did not contain 'non-200': %v", gotErr)
	}
}

func TestProcessStream_Non200ReadError(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Body:       errReadCloser{},
	}

	var gotErr error
	cb := &llmstreamer.StreamCallbacks{
		OnError: func(err error) { gotErr = err },
	}

	processStream(resp, cb)

	if gotErr == nil {
		t.Fatalf("expected OnError to be called when Read fails")
	}
	if !strings.Contains(gotErr.Error(), "read body failed") {
		t.Fatalf("expected error message to mention read body failure, got: %v", gotErr)
	}
}

func TestProcessStream_ReadFailedInLoop(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       errReadCloser{},
	}

	var gotErr error
	cb := &llmstreamer.StreamCallbacks{
		OnError:  func(err error) { gotErr = err },
		OnFinish: func(s string) { t.Fatalf("unexpected finish: %q", s) },
	}

	processStream(resp, cb)

	if gotErr == nil {
		t.Fatalf("expected OnError when reader returns error during streaming")
	}
	if !strings.Contains(gotErr.Error(), "read failed") {
		t.Fatalf("expected error message to contain 'read failed', got: %v", gotErr)
	}
}

func TestProcessStream_EOFTriggersFinish(t *testing.T) {
	body := "" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"Hi\"}}\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\" there\"}}\n"

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	var final string
	cb := &llmstreamer.StreamCallbacks{
		OnContent: func(s string) {},
		OnFinish:  func(f string) { final = f },
		OnError:   func(err error) { t.Fatalf("unexpected error: %v", err) },
	}

	processStream(resp, cb)

	if final != "Hi there" {
		t.Fatalf("expected final 'Hi there', got %q", final)
	}
}

func TestProcessStream_InvalidJSONThenValid(t *testing.T) {
	body := "" +
		"data: not-a-json\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"Ok\"}}\n" +
		"data: {\"type\":\"message_stop\"}\n"

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	var errs []string
	var contents []string
	var final string

	cb := &llmstreamer.StreamCallbacks{
		OnContent: func(s string) { contents = append(contents, s) },
		OnFinish:  func(f string) { final = f },
		OnError:   func(err error) { errs = append(errs, err.Error()) },
	}

	processStream(resp, cb)

	if len(errs) == 0 {
		t.Fatalf("expected parse error to be reported")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "failed to parse JSON") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected an error containing 'failed to parse JSON', got: %v", errs)
	}

	if len(contents) != 1 || contents[0] != "Ok" {
		t.Fatalf("expected one content chunk 'Ok', got: %v", contents)
	}
	if final != "Ok" {
		t.Fatalf("expected final 'Ok', got %q", final)
	}
}

func TestProcessStream_IgnoreEmptyLines(t *testing.T) {
	body := "" +
		"\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"A\"}}\n" +
		"   \n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"B\"}}\n"

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	var contents []string
	var final string

	cb := &llmstreamer.StreamCallbacks{
		OnContent: func(s string) { contents = append(contents, s) },
		OnFinish:  func(f string) { final = f },
		OnError:   func(err error) { t.Fatalf("unexpected error: %v", err) },
	}

	processStream(resp, cb)

	if len(contents) != 2 {
		t.Fatalf("expected 2 content chunks, got %d: %v", len(contents), contents)
	}
	if contents[0] != "A" || contents[1] != "B" {
		t.Fatalf("unexpected contents: %v", contents)
	}
	if final != "AB" {
		t.Fatalf("unexpected final message: %q", final)
	}
}
