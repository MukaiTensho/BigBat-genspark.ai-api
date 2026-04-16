package recaptcha

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string, timeout time.Duration) *Client {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return &Client{}
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	if strings.HasPrefix(baseURL, "http://") {
		transport.TLSClientConfig = nil
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != "" && c.httpClient != nil
}

func (c *Client) GetToken(ctx context.Context, cookie string) (string, error) {
	if !c.Enabled() {
		return "", nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/genspark", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("recaptcha proxy status %d", resp.StatusCode)
	}
	var payload struct {
		Code    int    `json:"code"`
		Token   string `json:"token"`
		Message string `json:"message"`
	}
	if err = json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("invalid recaptcha proxy json: %w", err)
	}
	if payload.Code != 200 || strings.TrimSpace(payload.Token) == "" {
		if payload.Message == "" {
			payload.Message = "recaptcha proxy returned empty token"
		}
		return "", errors.New(payload.Message)
	}
	return payload.Token, nil
}
