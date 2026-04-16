package genspark

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome"

type Client struct {
	baseURL           string
	askCopilotURL     string
	askAgentURL       string
	askAgentEventsURL string
	modelsConfigURL   string
	deleteURLFmt      string
	uploadURL         string
	httpClient        *http.Client
}

func NewClient(baseURL, proxyURL string, timeout time.Duration) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "https://www.genspark.ai"
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream base url: %w", err)
	}
	u.Path = ""
	u.RawQuery = ""
	u.Fragment = ""
	baseURL = strings.TrimRight(u.String(), "/")

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	if p := strings.TrimSpace(proxyURL); p != "" {
		proxy, parseErr := url.Parse(p)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid PROXY_URL: %w", parseErr)
		}
		transport.Proxy = http.ProxyURL(proxy)
	}

	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return &Client{
		baseURL:           baseURL,
		askCopilotURL:     baseURL + "/api/copilot/ask",
		askAgentURL:       baseURL + "/api/agent/ask_proxy",
		askAgentEventsURL: baseURL + "/api/agent/ask_proxy_events",
		modelsConfigURL:   baseURL + "/api/models_config",
		deleteURLFmt:      baseURL + "/api/project/delete?project_id=%s",
		uploadURL:         baseURL + "/api/get_upload_personal_image_url",
		httpClient:        httpClient,
	}, nil
}

// Ask sends a legacy copilot ask request.
func (c *Client) Ask(ctx context.Context, cookie string, payload map[string]any, accept string) (*http.Response, error) {
	if accept == "" {
		accept = "application/json"
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, http.MethodPost, c.askCopilotURL, cookie, accept, "application/json", bytes.NewReader(body), nil)
}

func (c *Client) AskBody(ctx context.Context, cookie string, payload map[string]any, accept string) (string, int, error) {
	resp, err := c.Ask(ctx, cookie, payload, accept)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, err
	}
	return string(raw), resp.StatusCode, nil
}

func (c *Client) AskAgent(ctx context.Context, cookie string, payload map[string]any, accept string, events bool) (*http.Response, error) {
	if accept == "" {
		accept = "application/json"
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	targetURL := c.askAgentURL
	if events {
		targetURL = c.askAgentEventsURL
	}
	return c.do(ctx, http.MethodPost, targetURL, cookie, accept, "application/json", bytes.NewReader(body), nil)
}

func (c *Client) AskAgentBody(ctx context.Context, cookie string, payload map[string]any, accept string) (string, int, error) {
	resp, err := c.AskAgent(ctx, cookie, payload, accept, false)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, err
	}
	return string(raw), resp.StatusCode, nil
}

func (c *Client) DeleteProject(ctx context.Context, cookie, projectID string) error {
	if strings.TrimSpace(projectID) == "" {
		return nil
	}
	url := fmt.Sprintf(c.deleteURLFmt, projectID)
	resp, err := c.do(ctx, http.MethodGet, url, cookie, "application/json", "application/json", nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}

func (c *Client) GetUploadURLs(ctx context.Context, cookie string) (uploadImageURL, privateStorageURL string, err error) {
	resp, err := c.do(ctx, http.MethodGet, c.uploadURL, cookie, "*/*", "application/json", nil, nil)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	var parsed struct {
		Data struct {
			UploadImageURL  string `json:"upload_image_url"`
			PrivateStoreURL string `json:"private_storage_url"`
		} `json:"data"`
	}
	if err = json.Unmarshal(body, &parsed); err != nil {
		return "", "", err
	}
	if parsed.Data.UploadImageURL == "" || parsed.Data.PrivateStoreURL == "" {
		return "", "", fmt.Errorf("upstream upload url response missing fields")
	}
	return parsed.Data.UploadImageURL, parsed.Data.PrivateStoreURL, nil
}

func (c *Client) UploadBytes(ctx context.Context, uploadURL string, data []byte) error {
	resp, err := c.do(ctx, http.MethodPut, uploadURL, "", "*/*", "application/octet-stream", bytes.NewReader(data), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) PollTaskStatus(ctx context.Context, cookie string, isVideo bool, taskIDs []string) (*http.Response, error) {
	endpoint := "/api/ig_tasks_status"
	if isVideo {
		endpoint = "/api/vg_tasks_status"
	}
	body, err := json.Marshal(map[string]any{"task_ids": taskIDs})
	if err != nil {
		return nil, err
	}
	return c.do(ctx, http.MethodPost, c.baseURL+endpoint, cookie, "*/*", "application/json", bytes.NewReader(body), nil)
}

func (c *Client) ModelsConfig(ctx context.Context, cookie string) (map[string]any, int, error) {
	raw, status, err := c.ModelsConfigRaw(ctx, cookie)
	if err != nil {
		return nil, status, err
	}
	var parsed map[string]any
	if err = json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, status, err
	}
	return parsed, status, nil
}

func (c *Client) ModelsConfigRaw(ctx context.Context, cookie string) (string, int, error) {
	resp, err := c.do(ctx, http.MethodGet, c.modelsConfigURL, cookie, "application/json", "application/json", nil, nil)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, err
	}
	return string(body), resp.StatusCode, nil
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) do(ctx context.Context, method, targetURL, cookie, accept, contentType string, body io.Reader, extraHeaders map[string]string) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	req.Header.Set("Origin", c.baseURL)
	req.Header.Set("Referer", c.baseURL+"/")
	req.Header.Set("User-Agent", defaultUserAgent)
	if strings.TrimSpace(cookie) != "" {
		req.Header.Set("Cookie", cookie)
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	return c.httpClient.Do(req)
}

func FetchBytes(ctx context.Context, targetURL string, timeout time.Duration) ([]byte, error) {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("fetch url failed with status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func JoinURL(base string, more ...string) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	parts := []string{u.Path}
	parts = append(parts, more...)
	u.Path = path.Join(parts...)
	return u.String()
}
