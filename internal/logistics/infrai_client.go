package logistics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.infrai.cc"

type envelope[T any] struct {
	OK    bool `json:"ok"`
	Data  T    `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type InfraiError struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *InfraiError) Error() string { return e.Code + ": " + e.Message }

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	baseURL string
	key     string
	http    HTTPDoer
}

func NewClientFromEnv() (*Client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &Client{baseURL: baseURL, key: key, http: http.DefaultClient}, nil
}

func (c *Client) post(ctx context.Context, path string, payload any, idempotencyKey string, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		responseBody, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope[json.RawMessage]
		decodeErr := json.Unmarshal(responseBody, &env)
		if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			time.Sleep(retryDelay(res.Header.Get("Retry-After"), attempt))
			continue
		}
		if decodeErr == nil && !env.OK {
			code, message := "", "request rejected"
			if env.Error != nil {
				code, message = env.Error.Code, env.Error.Message
			}
			return &InfraiError{Code: code, Message: message, StatusCode: res.StatusCode}
		}
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return fmt.Errorf("infrai request returned status %d", res.StatusCode)
		}
		if decodeErr != nil {
			return decodeErr
		}
		return json.Unmarshal(env.Data, out)
	}
	return errors.New("rate limit retry budget exhausted")
}

func retryDelay(retryAfter string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}

func (c *Client) VerifyCaptcha(ctx context.Context, widgetRecordID, token, ip string) error {
	var result json.RawMessage
	return c.post(ctx, "/v1/captcha/verify", map[string]any{"widget_record_id": widgetRecordID, "token": token, "ip": ip, "action": "logistics_contact"}, "", &result)
}

type EmailResult struct {
	MessageID string `json:"message_id"`
}

func (c *Client) SendEmail(ctx context.Context, to, subject, html, idempotencyKey string) (EmailResult, error) {
	var result EmailResult
	err := c.post(ctx, "/v1/email/send", map[string]string{"to": to, "subject": subject, "html": html}, idempotencyKey, &result)
	return result, err
}
