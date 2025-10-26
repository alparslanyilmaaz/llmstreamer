package llmstreamer

type StreamCallbacks struct {
	OnContent func(content string)
	OnFinish  func(finalMessage string)
	OnError   func(err error)
}
