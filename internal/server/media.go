package server

import (
	"bigbat/internal/config"
	"bigbat/internal/genspark"
	"bigbat/internal/openai"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (a *App) handleImagesGenerations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req openai.ImagesGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "Invalid request parameters", "invalid_request_error", "bad_request")
		return
	}
	req.Model = normalizeModelName(req.Model)
	if req.Model == "" || !config.IsImageModel(req.Model) {
		writeOpenAIError(w, http.StatusBadRequest, "Invalid model", "invalid_request_error", "bad_model")
		return
	}

	resp, err := a.generateImages(r.Context(), req)
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, err.Error(), "request_error", "upstream_failed")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) handleVideosGenerations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req openai.VideosGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "Invalid request parameters", "invalid_request_error", "bad_request")
		return
	}
	req.Model = normalizeModelName(req.Model)
	if req.Model == "" || !config.IsVideoModel(req.Model) {
		writeOpenAIError(w, http.StatusBadRequest, "Invalid model", "invalid_request_error", "bad_model")
		return
	}

	resp, err := a.generateVideos(r.Context(), req)
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, err.Error(), "request_error", "upstream_failed")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) generateImages(ctx context.Context, req openai.ImagesGenerationRequest) (*openai.ImagesGenerationResponse, error) {
	attempts := a.imageAttempts()
	if len(attempts) == 0 {
		return nil, fmt.Errorf("No valid cookies available")
	}

	for _, attempt := range attempts {
		body, err := a.buildImageRequestBody(ctx, req, attempt.Cookie, attempt.ChatID)
		if err != nil {
			return nil, err
		}
		if err = a.attachRecaptchaToken(ctx, attempt.Cookie, body); err != nil {
			return nil, err
		}

		rawBody, statusCode, err := a.Genspark.AskAgentBody(ctx, attempt.Cookie, body, "*/*")
		if err != nil {
			continue
		}
		if genspark.IsRetiredCopilot(rawBody) {
			return nil, fmt.Errorf("upstream deprecated old copilot flow for media, requires ai chat compatible media path")
		}
		rr := a.classifyFailure(statusCode, rawBody)
		if rr.Decision == retryWithNextCookie {
			a.processRetryResult(attempt.Cookie, rr)
			continue
		}
		if rr.Decision == retryFatal {
			return nil, errors.New(rr.Message)
		}

		projectID, taskIDs := genspark.ExtractTaskIDs(rawBody, false)
		if len(taskIDs) == 0 {
			continue
		}
		urls, pollErr := genspark.PollTaskResult(a.Genspark, attempt.Cookie, taskIDs, false, 2*time.Minute)
		if pollErr != nil || len(urls) == 0 {
			continue
		}

		data := make([]openai.ImageData, 0, len(urls))
		for _, u := range urls {
			item := openai.ImageData{URL: u, RevisedPrompt: req.Prompt}
			if req.ResponseFormat == "b64_json" {
				b64, b64Err := genspark.Base64ByURL(u, 60*time.Second)
				if b64Err == nil {
					item.B64JSON = "data:image/webp;base64," + b64
				}
			}
			data = append(data, item)
		}
		if len(data) == 0 {
			continue
		}

		a.configMu.RLock()
		autoDelete := a.Config.AutoDeleteChat
		a.configMu.RUnlock()
		if autoDelete {
			a.deleteProjectAsync(attempt.Cookie, projectID)
		}
		return &openai.ImagesGenerationResponse{
			Created: time.Now().Unix(),
			Data:    data,
		}, nil
	}

	return nil, fmt.Errorf("all cookies are temporarily unavailable")
}

func (a *App) generateVideos(ctx context.Context, req openai.VideosGenerationRequest) (*openai.VideosGenerationResponse, error) {
	attempts := a.imageAttempts()
	if len(attempts) == 0 {
		return nil, fmt.Errorf("No valid cookies available")
	}

	for _, attempt := range attempts {
		body, err := a.buildVideoRequestBody(ctx, req, attempt.Cookie, attempt.ChatID)
		if err != nil {
			return nil, err
		}
		if err = a.attachRecaptchaToken(ctx, attempt.Cookie, body); err != nil {
			return nil, err
		}

		rawBody, statusCode, err := a.Genspark.AskAgentBody(ctx, attempt.Cookie, body, "*/*")
		if err != nil {
			continue
		}
		if genspark.IsRetiredCopilot(rawBody) {
			return nil, fmt.Errorf("upstream deprecated old copilot flow for media, requires ai chat compatible media path")
		}
		rr := a.classifyFailure(statusCode, rawBody)
		if rr.Decision == retryWithNextCookie {
			a.processRetryResult(attempt.Cookie, rr)
			continue
		}
		if rr.Decision == retryFatal {
			return nil, errors.New(rr.Message)
		}

		projectID, taskIDs := genspark.ExtractTaskIDs(rawBody, true)
		if len(taskIDs) == 0 {
			continue
		}
		urls, pollErr := genspark.PollTaskResult(a.Genspark, attempt.Cookie, taskIDs, true, 2*time.Minute)
		if pollErr != nil || len(urls) == 0 {
			continue
		}

		data := make([]openai.VideoData, 0, len(urls))
		for _, u := range urls {
			item := openai.VideoData{URL: u, RevisedPrompt: req.Prompt}
			if req.ResponseFormat == "b64_json" {
				b64, b64Err := genspark.Base64ByURL(u, 60*time.Second)
				if b64Err == nil {
					item.B64JSON = b64
				}
			}
			data = append(data, item)
		}
		if len(data) == 0 {
			continue
		}

		a.configMu.RLock()
		autoDelete := a.Config.AutoDeleteChat
		a.configMu.RUnlock()
		if autoDelete {
			a.deleteProjectAsync(attempt.Cookie, projectID)
		}
		return &openai.VideosGenerationResponse{
			Created: time.Now().Unix(),
			Data:    data,
		}, nil
	}

	return nil, fmt.Errorf("all cookies are temporarily unavailable")
}

type imageAttempt struct {
	Cookie string
	ChatID string
}

func (a *App) imageAttempts() []imageAttempt {
	a.configMu.RLock()
	sessionImageMap := make(map[string]string, len(a.Config.SessionImageChatMap))
	for k, v := range a.Config.SessionImageChatMap {
		sessionImageMap[k] = v
	}
	a.configMu.RUnlock()
	if len(sessionImageMap) > 0 {
		pairs := randomizePairs(sessionImageMap)
		out := make([]imageAttempt, 0, len(pairs))
		for _, p := range pairs {
			out = append(out, imageAttempt{Cookie: p.Key, ChatID: p.Val})
		}
		return out
	}
	cookies := a.CookiePool.Candidates()
	out := make([]imageAttempt, 0, len(cookies))
	for _, cookie := range cookies {
		out = append(out, imageAttempt{Cookie: cookie})
	}
	return out
}

func (a *App) buildImageRequestBody(ctx context.Context, req openai.ImagesGenerationRequest, cookie, chatID string) (map[string]any, error) {
	modelName := req.Model
	if modelName == "dall-e-3" {
		modelName = "dalle-3"
	}
	modelConfigs := []map[string]any{{
		"model":                   modelName,
		"aspect_ratio":            "auto",
		"use_personalized_models": false,
		"fashion_profile_id":      nil,
		"hd":                      false,
		"reflection_enabled":      false,
		"style":                   "auto",
	}}

	messages := []map[string]any{{
		"role":    "user",
		"content": req.Prompt,
	}}
	if strings.TrimSpace(req.Image) != "" {
		normalized, err := normalizeImageInput(ctx, req.Image)
		if err == nil && normalized != "" {
			messages = []map[string]any{{
				"role": "user",
				"content": []map[string]any{
					{"type": "image_url", "image_url": map[string]any{"url": normalized}},
					{"type": "text", "text": req.Prompt},
				},
			}}
		}
	}

	currentQueryString := "type=" + imageType
	if strings.TrimSpace(chatID) != "" {
		currentQueryString = fmt.Sprintf("id=%s&type=%s", chatID, imageType)
	}

	body := map[string]any{
		"type":                 imageType,
		"current_query_string": currentQueryString,
		"messages":             messages,
		"user_s_input":         req.Prompt,
		"action_params":        map[string]any{},
		"extra_data": map[string]any{
			"model_configs":  modelConfigs,
			"llm_model":      "gpt-4o",
			"imageModelMap":  map[string]any{},
			"writingContent": nil,
		},
	}
	return body, nil
}

func (a *App) buildVideoRequestBody(ctx context.Context, req openai.VideosGenerationRequest, cookie, chatID string) (map[string]any, error) {
	modelConfigs := []map[string]any{{
		"model":              req.Model,
		"aspect_ratio":       req.AspectRatio,
		"reflection_enabled": req.AutoPrompt,
		"duration":           req.Duration,
	}}

	messages := []map[string]any{{
		"role":    "user",
		"content": req.Prompt,
	}}
	if strings.TrimSpace(req.Image) != "" {
		normalized, err := normalizeImageInput(ctx, req.Image)
		if err == nil && normalized != "" {
			messages = []map[string]any{{
				"role": "user",
				"content": []map[string]any{
					{"type": "image_url", "image_url": map[string]any{"url": normalized}},
					{"type": "text", "text": req.Prompt},
				},
			}}
		}
	}

	currentQueryString := "type=" + videoType
	if strings.TrimSpace(chatID) != "" {
		currentQueryString = fmt.Sprintf("id=%s&type=%s", chatID, videoType)
	}

	body := map[string]any{
		"type":                 videoType,
		"current_query_string": currentQueryString,
		"messages":             messages,
		"user_s_input":         req.Prompt,
		"action_params":        map[string]any{},
		"extra_data": map[string]any{
			"model_configs": modelConfigs,
			"imageModelMap": map[string]any{},
		},
	}
	return body, nil
}
