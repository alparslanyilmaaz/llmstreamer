package anthropic

type Model string

const (
	ModelClaude35Sonnet Model = "claude-3-5-sonnet-20241022"
	ModelClaude35Haiku  Model = "claude-3-5-haiku-20241022"

	ModelClaude3Opus   Model = "claude-3-opus-20240229"
	ModelClaude3Sonnet Model = "claude-3-sonnet-20240229"
	ModelClaude3Haiku  Model = "claude-3-haiku-20240307"

	ModelClaude21 Model = "claude-2.1"
	ModelClaude20 Model = "claude-2.0"

	ModelClaudeInstant12 Model = "claude-instant-1.2"
	ModelClaudeInstant11 Model = "claude-instant-1.1"
)
