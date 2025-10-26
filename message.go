package llmstreamer

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "assistant"
)

type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}
