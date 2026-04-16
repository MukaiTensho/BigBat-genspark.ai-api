package genspark

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func ExtractTaskIDs(responseBody string, video bool) (string, []string) {
	var projectID string
	taskIDs := make([]string, 0)

	lines := strings.Split(responseBody, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "data: ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		}
		if !strings.HasPrefix(line, "{") {
			continue
		}

		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if ev.Type == "project_start" && ev.ID != "" {
			projectID = ev.ID
		}
		if !strings.Contains(line, "task_id") {
			continue
		}
		if ev.Content == "" {
			continue
		}
		if video {
			var parsed struct {
				GeneratedVideos []struct {
					TaskID string `json:"task_id"`
				} `json:"generated_videos"`
			}
			if err := json.Unmarshal([]byte(ev.Content), &parsed); err != nil {
				continue
			}
			for _, item := range parsed.GeneratedVideos {
				if item.TaskID != "" {
					taskIDs = append(taskIDs, item.TaskID)
				}
			}
			continue
		}
		var parsed struct {
			GeneratedImages []struct {
				TaskID string `json:"task_id"`
			} `json:"generated_images"`
		}
		if err := json.Unmarshal([]byte(ev.Content), &parsed); err != nil {
			continue
		}
		for _, item := range parsed.GeneratedImages {
			if item.TaskID != "" {
				taskIDs = append(taskIDs, item.TaskID)
			}
		}
	}

	return projectID, taskIDs
}

func ExtractFinalTaskURLs(raw string, taskIDs []string, video bool) []string {
	urls := make([]string, 0)
	if strings.TrimSpace(raw) == "" {
		return urls
	}
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "data: ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		}
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event["type"] != "TASKS_STATUS_COMPLETE" {
			continue
		}
		finalStatus, ok := event["final_status"].(map[string]any)
		if !ok {
			continue
		}
		for _, taskID := range taskIDs {
			task, ok := finalStatus[taskID].(map[string]any)
			if !ok {
				continue
			}
			status, _ := task["status"].(string)
			if status != "SUCCESS" {
				continue
			}
			if video {
				if u := firstString(task["video_urls"]); u != "" {
					urls = append(urls, u)
				}
				continue
			}
			if u := firstString(task["image_urls"]); u != "" {
				urls = append(urls, u)
			}
		}
	}
	return urls
}

func firstString(v any) string {
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return ""
	}
	s, _ := arr[0].(string)
	return s
}

func PollTaskResult(c *Client, cookie string, taskIDs []string, video bool, timeout time.Duration) ([]string, error) {
	if len(taskIDs) == 0 {
		return nil, fmt.Errorf("empty task IDs")
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := c.PollTaskStatus(nil, cookie, video, taskIDs)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		raw, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		urls := ExtractFinalTaskURLs(string(raw), taskIDs, video)
		if len(urls) > 0 {
			return urls, nil
		}
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("task polling timed out")
}

func Base64ByURL(url string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
	}
	return base64.StdEncoding.EncodeToString(body), nil
}
