package server

import (
	"bigbat/internal/config"
	"bigbat/internal/genspark"
	"bigbat/internal/openai"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (a *App) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req openai.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "Invalid request parameters", "invalid_request_error", "bad_request")
		return
	}

	req.Model = normalizeModelName(req.Model)
	if req.Model == "" {
		writeOpenAIError(w, http.StatusBadRequest, "model is required", "invalid_request_error", "bad_request")
		return
	}
	if len(req.Messages) == 0 {
		writeOpenAIError(w, http.StatusBadRequest, "messages is required", "invalid_request_error", "bad_request")
		return
	}

	if config.IsImageModel(req.Model) {
		a.handleChatViaImageModel(w, r, &req)
		return
	}

	if req.Stream {
		a.handleChatStream(w, r, &req)
		return
	}
	a.handleChatNonStream(w, r, &req)
}

func (a *App) handleChatViaImageModel(w http.ResponseWriter, r *http.Request, req *openai.ChatCompletionRequest) {
	prompt := req.GetLastUserText()
	if strings.TrimSpace(prompt) == "" {
		writeOpenAIError(w, http.StatusBadRequest, "Invalid request parameters", "invalid_request_error", "bad_request")
		return
	}
	imgReq := openai.ImagesGenerationRequest{
		Model:  req.Model,
		Prompt: prompt,
	}
	resp, err := a.generateImages(r.Context(), imgReq)
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, err.Error(), "request_error", "upstream_failed")
		return
	}
	parts := make([]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		if item.URL != "" {
			parts = append(parts, fmt.Sprintf("![Image](%s)", item.URL))
		}
	}
	content := strings.Join(parts, "\n")
	jsonReq, _ := json.Marshal(req.Messages)
	promptTokens := openai.ApproxTokenCount(string(jsonReq))
	completionTokens := openai.ApproxTokenCount(content)

	if req.Stream {
		responseID := openai.NewResponseID()
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		if err = sendOpenAIChunk(w, openai.NewStreamChunk(req.Model, responseID, content, nil, promptTokens, completionTokens)); err != nil {
			return
		}
		finish := "stop"
		_ = sendOpenAIChunk(w, openai.NewStreamChunk(req.Model, responseID, "", &finish, promptTokens, completionTokens))
		_ = sendOpenAIDone(w)
		return
	}

	result := openai.NewChatCompletionResponse(req.Model, content, promptTokens, completionTokens)
	writeJSON(w, http.StatusOK, result)
}

func (a *App) handleChatStream(w http.ResponseWriter, r *http.Request, req *openai.ChatCompletionRequest) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	responseID := openai.NewResponseID()
	candidates := a.CookiePool.Candidates()
	if len(candidates) == 0 {
		writeOpenAIError(w, http.StatusInternalServerError, "No valid cookies available", "request_error", "no_cookie")
		return
	}

	for idx, cookie := range candidates {
		attemptReq, err := cloneChatRequest(req)
		if err != nil {
			writeOpenAIError(w, http.StatusInternalServerError, "failed to clone request", "request_error", "internal_error")
			return
		}
		body, modelName, searchModel, err := a.buildChatRequestBody(r.Context(), cookie, attemptReq)
		if err != nil {
			writeOpenAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "bad_request")
			return
		}
		if err = a.attachRecaptchaToken(r.Context(), cookie, body); err != nil {
			writeOpenAIError(w, http.StatusBadGateway, err.Error(), "request_error", "recaptcha_failed")
			return
		}

		jsonBody, _ := json.Marshal(body)
		promptTokens := openai.ApproxTokenCount(string(jsonBody))

		resp, askErr := a.Genspark.AskAgent(r.Context(), cookie, body, "text/event-stream", false)
		if askErr != nil {
			a.Logger.Printf("stream ask failed on attempt %d: %v", idx+1, askErr)
			continue
		}

		statusCode := resp.StatusCode
		if statusCode >= 400 {
			raw, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			rr := a.classifyFailure(statusCode, string(raw))
			a.processRetryResult(cookie, rr)
			if rr.Decision == retryWithNextCookie {
				continue
			}
			writeOpenAIError(w, rr.StatusCodeOr(http.StatusBadGateway), rr.Message, "request_error", "upstream_failed")
			return
		}

		var (
			projectID     string
			hitRetryError bool
			done          bool
			sentAnyDelta  bool
		)

		scanErr := genspark.ReadSSELines(resp.Body, func(rawLine string) error {
			if done {
				return nil
			}
			line := strings.TrimSpace(rawLine)
			if line == "" {
				return nil
			}

			rr := a.classifyFailure(http.StatusOK, line)
			if rr.Decision == retryWithNextCookie {
				a.processRetryResult(cookie, rr)
				hitRetryError = true
				return io.EOF
			}
			if rr.Decision == retryFatal {
				return errors.New(rr.Message)
			}

			payload := dataLine(line)
			if !strings.HasPrefix(payload, "{") {
				return nil
			}
			var event genspark.Event
			if err := json.Unmarshal([]byte(payload), &event); err != nil {
				return nil
			}

			switch event.Type {
			case "project_start":
				if event.ID != "" {
					projectID = event.ID
				}
			case "message_field", "message_field_delta":
				a.configMu.RLock()
				hideReasoning := a.Config.ReasoningHide
				a.configMu.RUnlock()
				if !shouldSendField(event.FieldName, hideReasoning) {
					return nil
				}
				delta := event.Delta
				if delta == "" {
					delta = event.FieldVal
				}
				if (modelName == "o1" || modelName == "o3-mini-high") && event.FieldName == "session_state.answer" && event.FieldVal != "" {
					delta = event.FieldVal
				}
				if delta != "" {
					if err := sendOpenAIChunk(w, openai.NewStreamChunk(modelName, responseID, delta, nil, promptTokens, openai.ApproxTokenCount(delta))); err != nil {
						return err
					}
					sentAnyDelta = true
				}
				if !hideReasoning {
					if event.FieldName == "session_state.answerthink_is_started" {
						if err := sendOpenAIChunk(w, openai.NewStreamChunk(modelName, responseID, "<think>\n", nil, promptTokens, 1)); err != nil {
							return err
						}
					}
					if event.FieldName == "session_state.answerthink_is_finished" {
						if err := sendOpenAIChunk(w, openai.NewStreamChunk(modelName, responseID, "\n</think>", nil, promptTokens, 1)); err != nil {
							return err
						}
					}
				}
			case "message_result":
				finalDelta := ""
				if !sentAnyDelta && strings.TrimSpace(event.Content) != "" {
					finalDelta = event.Content
				}
				if modelName == "o1" && searchModel {
					detail, err := extractDetailAnswer(event.Content)
					if err == nil {
						finalDelta = detail
					}
				}
				finish := "stop"
				if err := sendOpenAIChunk(w, openai.NewStreamChunk(modelName, responseID, finalDelta, &finish, promptTokens, openai.ApproxTokenCount(finalDelta))); err != nil {
					return err
				}
				if err := sendOpenAIDone(w); err != nil {
					return err
				}
				a.persistSessionAndAutoDelete(cookie, modelName, projectID)
				done = true
				return io.EOF
			}
			return nil
		})
		resp.Body.Close()

		if done {
			return
		}
		if hitRetryError {
			continue
		}
		if scanErr != nil && scanErr != io.EOF {
			writeOpenAIError(w, http.StatusBadGateway, scanErr.Error(), "request_error", "stream_error")
			return
		}
	}

	writeOpenAIError(w, http.StatusInternalServerError, "All cookies are temporarily unavailable.", "request_error", "no_cookie")
}

func (a *App) handleChatNonStream(w http.ResponseWriter, r *http.Request, req *openai.ChatCompletionRequest) {
	candidates := a.CookiePool.Candidates()
	if len(candidates) == 0 {
		writeOpenAIError(w, http.StatusInternalServerError, "No valid cookies available", "request_error", "no_cookie")
		return
	}

	for _, cookie := range candidates {
		attemptReq, err := cloneChatRequest(req)
		if err != nil {
			writeOpenAIError(w, http.StatusInternalServerError, "failed to clone request", "request_error", "internal_error")
			return
		}
		body, modelName, searchModel, err := a.buildChatRequestBody(r.Context(), cookie, attemptReq)
		if err != nil {
			writeOpenAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "bad_request")
			return
		}
		if err = a.attachRecaptchaToken(r.Context(), cookie, body); err != nil {
			writeOpenAIError(w, http.StatusBadGateway, err.Error(), "request_error", "recaptcha_failed")
			return
		}

		rawBody, statusCode, askErr := a.Genspark.AskAgentBody(r.Context(), cookie, body, "application/json")
		if askErr != nil {
			a.Logger.Printf("non-stream ask failed: %v", askErr)
			continue
		}

		rr := a.classifyFailure(statusCode, rawBody)
		if rr.Decision == retryWithNextCookie {
			a.processRetryResult(cookie, rr)
			continue
		}
		if rr.Decision == retryFatal {
			writeOpenAIError(w, rr.StatusCodeOr(http.StatusBadGateway), rr.Message, "request_error", "upstream_failed")
			return
		}

		events, parseErr := genspark.ParseBodyAsEvents(rawBody)
		if parseErr != nil {
			continue
		}

		var (
			projectID      string
			answerThink    strings.Builder
			answerFallback strings.Builder
			finalContent   string
		)

		for _, ev := range events {
			switch ev.Type {
			case "project_start":
				if ev.ID != "" {
					projectID = ev.ID
				}
			case "message_field":
				a.configMu.RLock()
				hideReasoning := a.Config.ReasoningHide
				a.configMu.RUnlock()
				if hideReasoning {
					if ev.FieldName == "content" && ev.FieldVal != "" {
						answerFallback.WriteString(ev.FieldVal)
					}
					continue
				}
				if ev.FieldName == "content" && ev.FieldVal != "" {
					answerFallback.WriteString(ev.FieldVal)
				}
				if ev.FieldName == "session_state.answerthink_is_started" {
					answerThink.WriteString("<think>\n")
				}
				if ev.FieldName == "session_state.answerthink_is_finished" {
					answerThink.WriteString("\n</think>")
				}
			case "message_field_delta":
				a.configMu.RLock()
				hideReasoning := a.Config.ReasoningHide
				a.configMu.RUnlock()
				if shouldSendField(ev.FieldName, hideReasoning) {
					if ev.Delta != "" {
						answerFallback.WriteString(ev.Delta)
					} else if ev.FieldVal != "" {
						answerFallback.WriteString(ev.FieldVal)
					}
				}
				if !hideReasoning && ev.FieldName == "session_state.answerthink" {
					answerThink.WriteString(ev.Delta)
				}
			case "message_result":
				content := ev.Content
				if modelName == "o1" && searchModel {
					detail, err := extractDetailAnswer(content)
					if err == nil {
						content = detail
					}
				}
				finalContent = strings.TrimSpace(answerThink.String() + content)
			}
		}

		if strings.TrimSpace(finalContent) == "" {
			finalContent = strings.TrimSpace(answerThink.String() + answerFallback.String())
		}
		if finalContent == "" {
			continue
		}

		requestJSON, _ := json.Marshal(body)
		promptTokens := openai.ApproxTokenCount(string(requestJSON))
		completionTokens := openai.ApproxTokenCount(finalContent)
		result := openai.NewChatCompletionResponse(modelName, finalContent, promptTokens, completionTokens)
		writeJSON(w, http.StatusOK, result)
		a.persistSessionAndAutoDelete(cookie, modelName, projectID)
		return
	}

	writeOpenAIError(w, http.StatusInternalServerError, "All cookies are temporarily unavailable.", "request_error", "no_cookie")
}

func (a *App) buildChatRequestBody(ctx context.Context, cookie string, req *openai.ChatCompletionRequest) (map[string]any, string, bool, error) {
	modelName, searchModel := removeSearchSuffix(req.Model)
	req.Model = modelName
	req.NormalizeSystemMessagesForDeepSeek(modelName)
	a.configMu.RLock()
	preMessages := a.Config.PreMessagesJSON
	modelChatMap := make(map[string]string, len(a.Config.ModelChatMap))
	for k, v := range a.Config.ModelChatMap {
		modelChatMap[k] = v
	}
	a.configMu.RUnlock()
	if err := req.PrependMessagesFromJSON(preMessages); err != nil {
		return nil, "", false, fmt.Errorf("PrependMessagesFromJSON err: %w", err)
	}
	if err := a.processChatMessages(ctx, cookie, req.Messages); err != nil {
		return nil, "", false, fmt.Errorf("processMessages err: %w", err)
	}

	projectID := ""
	if chatID, ok := modelChatMap[modelName]; ok {
		projectID = chatID
	} else if chatID, ok := a.SessionPool.Get(cookie, modelName); ok {
		projectID = chatID
	} else {
		req.FilterToLastUserTurn()
	}

	models := []string{modelName}
	if !config.IsTextModel(modelName) {
		models = slicesClone(config.MixtureModelList)
	}

	userInput := req.GetLastUserText()
	if strings.TrimSpace(userInput) == "" {
		userInput = fallbackLastUserInput(req.Messages)
	}

	body := map[string]any{
		"type":         chatTypeAgent,
		"messages":     req.Messages,
		"user_s_input": userInput,
		"action_params": map[string]any{
			"model":                 modelName,
			"request_web_knowledge": searchModel,
		},
		"extra_data": map[string]any{
			"models":                 models,
			"run_with_another_model": false,
			"writingContent":         nil,
			"request_web_knowledge":  searchModel,
		},
	}
	if strings.TrimSpace(projectID) != "" {
		body["project_id"] = projectID
	}

	return body, modelName, searchModel, nil
}

func (a *App) processChatMessages(ctx context.Context, cookie string, messages []openai.ChatMessage) error {
	for i := range messages {
		parts, ok := messages[i].Content.([]any)
		if !ok {
			continue
		}
		for j, part := range parts {
			partMap, ok := part.(map[string]any)
			if !ok {
				continue
			}
			typeName, _ := partMap["type"].(string)
			if typeName != "image_url" {
				continue
			}

			imageURLVal, ok := partMap["image_url"]
			if !ok {
				continue
			}
			var rawURL string
			switch iv := imageURLVal.(type) {
			case string:
				rawURL = iv
			case map[string]any:
				rawURL, _ = iv["url"].(string)
			}
			if strings.TrimSpace(rawURL) == "" {
				continue
			}

			bytes, err := bytesFromURLOrBase64(ctx, rawURL)
			if err != nil {
				return err
			}

			contentType := http.DetectContentType(bytes)
			if strings.HasPrefix(contentType, "image/") {
				imageDataURL := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(bytes)
				switch iv := imageURLVal.(type) {
				case string:
					partMap["image_url"] = map[string]any{"url": imageDataURL}
				case map[string]any:
					iv["url"] = imageDataURL
					partMap["image_url"] = iv
				default:
					partMap["image_url"] = map[string]any{"url": imageDataURL}
				}
				parts[j] = partMap
				continue
			}

			uploadURL, privateStorageURL, err := a.Genspark.GetUploadURLs(ctx, cookie)
			if err != nil {
				return err
			}
			if err = a.Genspark.UploadBytes(ctx, uploadURL, bytes); err != nil {
				return err
			}

			parts[j] = map[string]any{
				"type": "private_file",
				"private_file": map[string]any{
					"name":                "file",
					"type":                contentType,
					"size":                len(bytes),
					"ext":                 contentExt(contentType),
					"private_storage_url": privateStorageURL,
				},
			}
		}
		messages[i].Content = parts
	}
	return nil
}

func bytesFromURLOrBase64(ctx context.Context, raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return genspark.FetchBytes(ctx, raw, 60*time.Second)
	}
	base64Part := raw
	if strings.Contains(raw, ";base64,") {
		parts := strings.SplitN(raw, ";base64,", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid base64 input")
		}
		base64Part = parts[1]
	}
	decoded, err := base64.StdEncoding.DecodeString(base64Part)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func (a *App) persistSessionAndAutoDelete(cookie, modelName, projectID string) {
	if strings.TrimSpace(projectID) == "" {
		return
	}
	a.configMu.RLock()
	autoModelChatMap := a.Config.AutoModelChatMap
	autoDelete := a.Config.AutoDeleteChat
	a.configMu.RUnlock()
	if autoModelChatMap {
		a.SessionPool.Add(cookie, modelName, projectID)
		return
	}
	if autoDelete {
		a.deleteProjectAsync(cookie, projectID)
	}
}

func (a *App) deleteProjectAsync(cookie, projectID string) {
	if strings.TrimSpace(projectID) == "" {
		return
	}
	if a.shouldKeepProject(cookie, projectID) {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = a.Genspark.DeleteProject(ctx, cookie, projectID)
	}()
}

func (a *App) shouldKeepProject(cookie, projectID string) bool {
	a.configMu.RLock()
	modelMap := make(map[string]string, len(a.Config.ModelChatMap))
	for k, v := range a.Config.ModelChatMap {
		modelMap[k] = v
	}
	sessionImageMap := make(map[string]string, len(a.Config.SessionImageChatMap))
	for k, v := range a.Config.SessionImageChatMap {
		sessionImageMap[k] = v
	}
	a.configMu.RUnlock()
	for _, id := range modelMap {
		if id == projectID {
			return true
		}
	}
	for _, id := range sessionImageMap {
		if id == projectID {
			return true
		}
	}
	for _, id := range a.SessionPool.ProjectIDsByCookie(cookie) {
		if id == projectID {
			return true
		}
	}
	return false
}

func sendOpenAIChunk(w http.ResponseWriter, chunk openai.ChatCompletionResponse) error {
	b, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	if _, err = fmt.Fprintf(w, "data: %s\n\n", string(b)); err != nil {
		return err
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func sendOpenAIDone(w http.ResponseWriter) error {
	if _, err := io.WriteString(w, "data: [DONE]\n\n"); err != nil {
		return err
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func slicesClone(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func fallbackLastUserInput(messages []openai.ChatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" {
			continue
		}
		switch v := messages[i].Content.(type) {
		case string:
			return v
		case []any:
			for _, it := range v {
				item, ok := it.(map[string]any)
				if !ok {
					continue
				}
				typeName, _ := item["type"].(string)
				if typeName != "text" {
					continue
				}
				text, _ := item["text"].(string)
				if strings.TrimSpace(text) != "" {
					return text
				}
			}
		}
	}
	return ""
}

func (r retryResult) StatusCodeOr(def int) int {
	if r.StatusCode > 0 {
		return r.StatusCode
	}
	return def
}
