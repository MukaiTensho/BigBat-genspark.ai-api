package openai

import (
	"encoding/json"
	"strings"
	"time"
)

type ChatCompletionRequest struct {
	Model    string        `json:"model"`
	Stream   bool          `json:"stream"`
	Messages []ChatMessage `json:"messages"`
	ChatExtraRequest
}

type ChatExtraRequest struct {
	ChannelID *string `json:"channelId,omitempty"`
}

type SessionState struct {
	Models           []string `json:"models,omitempty"`
	Layers           int      `json:"layers,omitempty"`
	Answer           string   `json:"answer,omitempty"`
	AnswerIsFinished bool     `json:"answer_is_finished,omitempty"`
}

type ChatMessage struct {
	Role         string        `json:"role"`
	Content      any           `json:"content"`
	IsPrompt     bool          `json:"is_prompt,omitempty"`
	SessionState *SessionState `json:"session_state,omitempty"`
}

func (r *ChatCompletionRequest) PrependMessagesFromJSON(jsonStr string) error {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return nil
	}
	var msgs []ChatMessage
	if err := json.Unmarshal([]byte(jsonStr), &msgs); err != nil {
		return err
	}
	insertAt := 0
	for i := len(r.Messages) - 1; i >= 0; i-- {
		if r.Messages[i].Role == "system" {
			insertAt = i + 1
			break
		}
	}
	out := make([]ChatMessage, 0, len(r.Messages)+len(msgs))
	out = append(out, r.Messages[:insertAt]...)
	out = append(out, msgs...)
	out = append(out, r.Messages[insertAt:]...)
	r.Messages = out
	return nil
}

func (r *ChatCompletionRequest) NormalizeSystemMessagesForDeepSeek(model string) {
	if model != "deep-seek-r1" {
		return
	}
	for i := range r.Messages {
		if r.Messages[i].Role == "system" {
			r.Messages[i].Role = "user"
		}
		if r.Messages[i].Role == "assistant" {
			r.Messages[i].IsPrompt = false
			r.Messages[i].SessionState = &SessionState{Models: []string{model}}
		}
	}
}

func (r *ChatCompletionRequest) FilterToLastUserTurn() {
	for i := len(r.Messages) - 1; i >= 0; i-- {
		if r.Messages[i].Role == "user" {
			r.Messages = r.Messages[i:]
			return
		}
	}
}

func (r *ChatCompletionRequest) GetLastUserText() string {
	for i := len(r.Messages) - 1; i >= 0; i-- {
		if r.Messages[i].Role != "user" {
			continue
		}
		switch v := r.Messages[i].Content.(type) {
		case string:
			return v
		}
	}
	return ""
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code"`
}

type ChatCompletionResponse struct {
	ID                string        `json:"id"`
	Object            string        `json:"object"`
	Created           int64         `json:"created"`
	Model             string        `json:"model"`
	Choices           []Choice      `json:"choices"`
	Usage             Usage         `json:"usage"`
	SystemFingerprint *string       `json:"system_fingerprint,omitempty"`
	Suggestions       []string      `json:"suggestions,omitempty"`
	Meta              *ResponseMeta `json:"meta,omitempty"`
}

type ResponseMeta struct {
	RequestID string `json:"request_id,omitempty"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message,omitempty"`
	LogProbs     *string `json:"logprobs,omitempty"`
	FinishReason *string `json:"finish_reason,omitempty"`
	Delta        Delta   `json:"delta,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Delta struct {
	Content string `json:"content,omitempty"`
	Role    string `json:"role,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ImagesGenerationRequest struct {
	ChatExtraRequest
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	ResponseFormat string `json:"response_format,omitempty"`
	Image          string `json:"image,omitempty"`
}

type VideosGenerationRequest struct {
	ResponseFormat string `json:"response_format,omitempty"`
	Model          string `json:"model"`
	AspectRatio    string `json:"aspect_ratio"`
	Duration       int    `json:"duration"`
	Prompt         string `json:"prompt"`
	AutoPrompt     bool   `json:"auto_prompt"`
	Image          string `json:"image,omitempty"`
}

type ImagesGenerationResponse struct {
	Created     int64         `json:"created"`
	DailyLimit  bool          `json:"dailyLimit,omitempty"`
	Data        []ImageData   `json:"data"`
	Suggestions []string      `json:"suggestions,omitempty"`
	Meta        *ResponseMeta `json:"meta,omitempty"`
}

type ImageData struct {
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
}

type VideosGenerationResponse struct {
	Created int64         `json:"created"`
	Data    []VideoData   `json:"data"`
	Meta    *ResponseMeta `json:"meta,omitempty"`
}

type VideoData struct {
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
}

type ModelObject struct {
	ID     string `json:"id"`
	Object string `json:"object"`
}

type ModelListResponse struct {
	Object string        `json:"object"`
	Data   []ModelObject `json:"data"`
}

func NewChatCompletionResponse(modelName, content string, promptTokens, completionTokens int) ChatCompletionResponse {
	finish := "stop"
	return ChatCompletionResponse{
		ID:      NewResponseID(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   modelName,
		Choices: []Choice{{
			Index: 0,
			Message: Message{
				Role:    "assistant",
				Content: content,
			},
			FinishReason: &finish,
		}},
		Usage: Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}
}

func NewStreamChunk(modelName, responseID, deltaContent string, finishReason *string, promptTokens, completionTokens int) ChatCompletionResponse {
	return ChatCompletionResponse{
		ID:      responseID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   modelName,
		Choices: []Choice{{
			Index: 0,
			Delta: Delta{
				Role:    "assistant",
				Content: deltaContent,
			},
			FinishReason: finishReason,
		}},
		Usage: Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}
}
