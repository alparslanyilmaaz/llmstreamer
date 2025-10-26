package main

import (
	"context"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/alparslanyilmaaz/llmstreamer/anthropic"
	"github.com/alparslanyilmaaz/llmstreamer"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	openKey := os.Getenv("openai")

	streamer := openai.New(openKey, openai.ModelGPT4o)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					cancel()
					conn.Close()
					break
				}
			}
		}()

		messages := []llmstreamer.Message{}

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}

			userMessage := llmstreamer.Message{
				Role:    "user",
				Content: string(msg),
			}
			messages = append(messages, userMessage)

			cb := &llmstreamer.StreamCallbacks{
				OnContent: func(content string) {
					conn.WriteMessage(websocket.TextMessage, []byte(content))
				},
				OnFinish: func(finalMessage string) {
					assistantMessage := llmstreamer.Message{
						Role:    "assistant",
						Content: finalMessage,
					}
					messages = append(messages, assistantMessage)

					conn.WriteMessage(websocket.TextMessage, []byte("[DONE]"))
				},
				OnError: func(err error) {
					conn.WriteMessage(websocket.TextMessage, []byte("[ERROR] "+err.Error()))
				},
			}

			go streamer.StreamChat(ctx, messages, cb)
		}
	})

	http.ListenAndServe(":8080", nil)
}