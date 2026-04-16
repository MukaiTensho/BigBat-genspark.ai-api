package anthropic

type MessagesRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens,omitempty"`
	Stream    bool             `json:"stream,omitempty"`
	System    any              `json:"system,omitempty"`
	Messages  []RequestMessage `json:"messages"`
}

type RequestMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type MessageResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Model        string         `json:"model"`
	Content      []ContentBlock `json:"content"`
	StopReason   string         `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence"`
	Usage        ResponseUsage  `json:"usage"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
