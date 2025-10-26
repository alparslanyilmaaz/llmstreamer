# LLM Streamer

[![Go Reference](https://pkg.go.dev/badge/github.com/alparslanyilmaaz/llmstreamer.svg)](https://pkg.go.dev/github.com/alparslanyilmaaz/llmstreamer)
[![Go Report Card](https://goreportcard.com/badge/github.com/alparslanyilmaaz/llmstreamer)](https://goreportcard.com/report/github.com/alparslanyilmaaz/llmstreamer)

A Go library for streaming chat completions from LLM APIs. Currently supports **Anthropic Claude** and **OpenAI GPT** models with real time streaming capabilities.

## Features

- **Real-time streaming** - Get responses as they're generated
- **Unified interface** - Same API for different LLM providers
- **Context-aware** - Built-in context cancellation support
- **Lightweight** - Minimal dependencies, clean architecture
- **Flexible callbacks** - Handle content, completion, and errors your way
- **WebSocket ready** - Perfect for real-time web applications

## Supported Providers

### Available Models

#### Anthropic Models
```go
anthropic.ModelClaude35Sonnet  // claude-3-5-sonnet-20241022
anthropic.ModelClaude35Haiku   // claude-3-5-haiku-20241022
anthropic.ModelClaude3Opus     // claude-3-opus-20240229
anthropic.ModelClaude3Sonnet   // claude-3-sonnet-20240229
anthropic.ModelClaude3Haiku    // claude-3-haiku-20240307
anthropic.ModelClaude21        // claude-2.1
anthropic.ModelClaude20        // claude-2.0
```

#### OpenAI Models
```go
openai.ModelGPT4o         // gpt-4o
openai.ModelGPT4oMini     // gpt-4o-mini
openai.ModelGPT4Turbo     // gpt-4-turbo
openai.ModelGPT35Turbo    // gpt-3.5-turbo
```

## Installation

```bash
go get github.com/alparslanyilmaaz/llmstreamer
```

## Quick Start

### Anthropic Example

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/alparslanyilmaaz/llmstreamer"
    "github.com/alparslanyilmaaz/llmstreamer/anthropic"
)

func main() {
    // Initialize the streamer
    streamer := anthropic.New("your-api-key", anthropic.ModelClaude35Sonnet)
    
    // Prepare messages
    messages := []llmstreamer.Message{
        {Role: llmstreamer.RoleUser, Content: "Hello, how are you?"},
    }
    
    // Set up callbacks
    callbacks := &llmstreamer.StreamCallbacks{
        OnContent: func(content string) {
            fmt.Print(content) // Print each chunk as it arrives
        },
        OnFinish: func(finalMessage string) {
            fmt.Println("\n--- Stream completed ---")
        },
        OnError: func(err error) {
            fmt.Printf("Error: %v\n", err)
        },
    }
    
    // Start streaming
    ctx := context.Background()
    streamer.StreamChat(ctx, messages, callbacks)
}
```

### OpenAI Example

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/alparslanyilmaaz/llmstreamer"
    "github.com/alparslanyilmaaz/llmstreamer/openai"
)

func main() {
    // Initialize the streamer
    streamer := openai.New("your-api-key", openai.ModelGPT4o)
    
    // Prepare messages
    messages := []llmstreamer.Message{
        {Role: llmstreamer.RoleUser, Content: "Explain quantum computing"},
    }
    
    // Set up callbacks
    callbacks := &llmstreamer.StreamCallbacks{
        OnContent: func(content string) {
            fmt.Print(content)
        },
        OnFinish: func(finalMessage string) {
            fmt.Println("\n--- Complete response received ---")
        },
        OnError: func(err error) {
            fmt.Printf("Error: %v\n", err)
        },
    }
    
    // Start streaming
    ctx := context.Background()
    streamer.StreamChat(ctx, messages, callbacks)
}
```

## WebSocket Integration

The library works seamlessly with WebSocket connections for real-time web applications. Check out the example implementations:

- [Anthropic WebSocket Example](examples/anthropic-websocket/main.go)
- [OpenAI WebSocket Example](examples/openai-websocket/main.go)

### WebSocket Example Snippet

```go
// WebSocket handler example
http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
    conn, _ := upgrader.Upgrade(w, r, nil)
    defer conn.Close()
    
    streamer := anthropic.New(apiKey, anthropic.ModelClaude35Haiku)
    
    callbacks := &llmstreamer.StreamCallbacks{
        OnContent: func(content string) {
            // Send each chunk to WebSocket client
            conn.WriteMessage(websocket.TextMessage, []byte(content))
        },
        OnFinish: func(finalMessage string) {
            conn.WriteMessage(websocket.TextMessage, []byte("[DONE]"))
        },
        OnError: func(err error) {
            conn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
        },
    }
    
    streamer.StreamChat(context.Background(), messages, callbacks)
})
```

## API Reference

### Core Types

```go
// Message represents a chat message
type Message struct {
    Role    Role   `json:"role"`    // "user" or "assistant" 
    Content string `json:"content"` // Message content
}

// StreamCallbacks defines callback functions for streaming
type StreamCallbacks struct {
    OnContent func(content string)     // Called for each content chunk
    OnFinish  func(finalMessage string) // Called when stream completes
    OnError   func(err error)          // Called on errors
}
```

### Provider Interfaces

Both Anthropic and OpenAI streamers implement the same interface:

```go
type Streamer interface {
    StreamChat(ctx context.Context, messages []Message, cb *StreamCallbacks)
}
```

## Configuration

### Environment Variables

For the WebSocket examples, set your API keys as environment variables:

```bash
export anthropic="your-anthropic-key"
export openai="your-openai-key"
```

### Context Cancellation

The library supports context cancellation for shutdown:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

streamer.StreamChat(ctx, messages, callbacks)
```

## Error Handling

The library provides basic error handling through the `OnError` callback:

```go
callbacks := &llmstreamer.StreamCallbacks{
    OnError: func(err error) {
        switch {
        case strings.Contains(err.Error(), "401"):
            log.Println("Authentication failed - check your API key")
        case strings.Contains(err.Error(), "429"):
            log.Println("Rate limit exceeded - please retry later")
        default:
            log.Printf("Unexpected error: %v", err)
        }
    },
}
```

## Examples

Run the WebSocket examples:

```bash
cd examples/anthropic-websocket
anthropic="your-key" go run main.go

cd examples/openai-websocket
openai="your-key" go run main.go
```


## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

If you encounter any issues or have questions, please [open an issue](https://github.com/alparslanyilmaaz/llmstreamer/issues) on GitHub.