package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultBaseURL = "http://127.0.0.1:3000"

type Client struct {
	BaseURL       string
	Token         string
	SessionCookie string
	TokenProvider TokenProvider
	HTTPClient    *http.Client
}

type TokenProvider interface {
	AccessToken(ctx context.Context, forceRefresh bool) (string, error)
}

type AutoScheduleRequest struct {
	OrderIDs          []string                  `json:"orderIds,omitempty"`
	Persist           bool                      `json:"persist"`
	RunOverrides      *AutoScheduleRunOverrides `json:"runOverrides,omitempty"`
	MaterialReadiness *MaterialReadinessControl `json:"materialReadiness,omitempty"`
}

type MaterialReadinessControl struct {
	Enabled       *bool  `json:"enabled,omitempty"`
	Mode          string `json:"mode,omitempty"`
	Source        string `json:"source,omitempty"`
	MaxAgeMinutes int    `json:"maxAgeMinutes,omitempty"`
}

type AutoScheduleRunOverrides struct {
	CapacityCalculationMode   string   `json:"capacityCalculationMode,omitempty"`
	ResourceIDs               []string `json:"resourceIds,omitempty"`
	PreferSameProductResource *bool    `json:"preferSameProductResource,omitempty"`
	ReplanUnstartedOrders     *bool    `json:"replanUnstartedOrders,omitempty"`
}

type MoveRequest struct {
	SourceType   string `json:"sourceType"`
	OrderID      string `json:"orderId,omitempty"`
	ScheduleID   string `json:"scheduleId,omitempty"`
	TargetTeamID string `json:"targetTeamId"`
	DryRun       bool   `json:"dryRun"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status,omitempty"`
}

type APIError struct {
	Code    string
	Message string
	Status  int
}

func (e *APIError) Error() string {
	if e.Status > 0 {
		return fmt.Sprintf("%s: HTTP %d: %s", e.Code, e.Status, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func NewWithTokenProvider(baseURL string, provider TokenProvider) *Client {
	client := New(baseURL, "")
	client.TokenProvider = provider
	return client
}

func NewWithSessionCookie(baseURL, sessionCookie string) *Client {
	client := New(baseURL, "")
	client.SessionCookie = sessionCookie
	return client
}

func (c *Client) AutoSchedule(ctx context.Context, req AutoScheduleRequest) (map[string]any, error) {
	return c.Post(ctx, "/api/laps/auto-schedule", req)
}

func (c *Client) Move(ctx context.Context, req MoveRequest) (map[string]any, error) {
	return c.Post(ctx, "/api/laps/move", req)
}

func (c *Client) Get(ctx context.Context, path string, query url.Values) (map[string]any, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	fullURL := c.BaseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, &APIError{Code: "CONFIG_ERROR", Message: err.Error()}
	}
	return c.do(httpReq)
}

func (c *Client) Post(ctx context.Context, path string, req any) (map[string]any, error) {
	return c.JSON(ctx, http.MethodPost, path, req)
}

func (c *Client) Patch(ctx context.Context, path string, req any) (map[string]any, error) {
	return c.JSON(ctx, http.MethodPatch, path, req)
}

func (c *Client) Put(ctx context.Context, path string, req any) (map[string]any, error) {
	return c.JSON(ctx, http.MethodPut, path, req)
}

func (c *Client) Delete(ctx context.Context, path string) (map[string]any, error) {
	return c.JSON(ctx, http.MethodDelete, path, nil)
}

func (c *Client) JSON(ctx context.Context, method string, path string, req any) (map[string]any, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	var body []byte
	var err error
	if req != nil {
		body, err = json.Marshal(req)
	}
	if err != nil {
		return nil, &APIError{Code: "CONFIG_ERROR", Message: err.Error()}
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		method,
		c.BaseURL+path,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, &APIError{Code: "CONFIG_ERROR", Message: err.Error()}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	return c.do(httpReq)
}

func (c *Client) Upload(ctx context.Context, path string, filePath string, fields map[string]string) (map[string]any, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, &APIError{Code: "CONFIG_ERROR", Message: err.Error()}
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, &APIError{Code: "CONFIG_ERROR", Message: err.Error()}
	}
	if _, err := part.Write(raw); err != nil {
		return nil, &APIError{Code: "CONFIG_ERROR", Message: err.Error()}
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, &APIError{Code: "CONFIG_ERROR", Message: err.Error()}
		}
	}
	if err := writer.Close(); err != nil {
		return nil, &APIError{Code: "CONFIG_ERROR", Message: err.Error()}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body.Bytes()))
	if err != nil {
		return nil, &APIError{Code: "CONFIG_ERROR", Message: err.Error()}
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	return c.do(httpReq)
}

func (c *Client) Download(ctx context.Context, path string) ([]byte, string, error) {
	if err := c.validate(); err != nil {
		return nil, "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, "", &APIError{Code: "CONFIG_ERROR", Message: err.Error()}
	}
	resp, err := c.doResponse(httpReq)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, "", &APIError{Code: "HTTP_ERROR", Message: readErr.Error(), Status: resp.StatusCode}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", &APIError{Code: "HTTP_ERROR", Message: responseMessage(raw), Status: resp.StatusCode}
	}
	return raw, resp.Header.Get("Content-Type"), nil
}

func (c *Client) validate() error {
	if c.BaseURL == "" {
		return &APIError{Code: "CONFIG_ERROR", Message: "base URL is required"}
	}
	if c.Token == "" && c.TokenProvider == nil && c.SessionCookie == "" {
		return &APIError{Code: "CONFIG_ERROR", Message: "authentication is required"}
	}
	return nil
}

func (c *Client) do(httpReq *http.Request) (map[string]any, error) {
	resp, err := c.doResponse(httpReq)
	if err != nil {
		return nil, err
	}
	return parseResponse(resp)
}

func (c *Client) doResponse(httpReq *http.Request) (*http.Response, error) {
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	for attempt := 0; attempt < 2; attempt++ {
		request := httpReq
		if attempt > 0 {
			request = httpReq.Clone(httpReq.Context())
			if httpReq.Body != nil && httpReq.Body != http.NoBody {
				if httpReq.GetBody == nil {
					break
				}
				body, err := httpReq.GetBody()
				if err != nil {
					return nil, &APIError{Code: "HTTP_ERROR", Message: err.Error()}
				}
				request.Body = body
			}
		}

		if c.SessionCookie != "" {
			request.Header.Del("Authorization")
			request.Header.Set("Cookie", c.SessionCookie)
		} else {
			token := c.Token
			if c.TokenProvider != nil {
				var err error
				token, err = c.TokenProvider.AccessToken(request.Context(), attempt > 0)
				if err != nil {
					return nil, &APIError{Code: "AUTH_ERROR", Message: err.Error()}
				}
			}
			request.Header.Set("Authorization", "Bearer "+token)
		}
		request.Header.Set("Accept", "application/json")

		resp, err := httpClient.Do(request)
		if err != nil {
			return nil, &APIError{Code: "HTTP_ERROR", Message: err.Error()}
		}
		if resp.StatusCode == http.StatusUnauthorized && c.TokenProvider != nil && attempt == 0 {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			continue
		}
		return resp, nil
	}
	return nil, &APIError{Code: "AUTH_ERROR", Message: "OAuth token refresh failed"}
}

func parseResponse(resp *http.Response) (map[string]any, error) {
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &APIError{Code: "HTTP_ERROR", Message: err.Error(), Status: resp.StatusCode}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{Code: "HTTP_ERROR", Message: responseMessage(raw), Status: resp.StatusCode}
	}

	var payload map[string]any
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, &APIError{Code: "PARSE_ERROR", Message: "empty response", Status: resp.StatusCode}
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &APIError{Code: "PARSE_ERROR", Message: err.Error(), Status: resp.StatusCode}
	}

	if success, ok := payload["success"].(bool); ok && !success {
		return nil, &APIError{Code: "API_ERROR", Message: responseMessage(raw), Status: resp.StatusCode}
	}

	return payload, nil
}

func responseMessage(raw []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return strings.TrimSpace(string(raw))
	}

	for _, path := range [][]string{
		{"message"},
		{"error", "message"},
		{"error", "details"},
	} {
		if value, ok := nestedString(payload, path...); ok {
			return value
		}
	}

	text, err := json.Marshal(payload)
	if err != nil {
		return strings.TrimSpace(string(raw))
	}
	return string(text)
}

func nestedString(payload map[string]any, path ...string) (string, bool) {
	var current any = payload
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current, ok = obj[key]
		if !ok {
			return "", false
		}
	}
	switch value := current.(type) {
	case string:
		return value, value != ""
	default:
		return fmt.Sprint(value), current != nil
	}
}

func AsAPIError(err error) *APIError {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return &APIError{Code: "API_ERROR", Message: err.Error()}
}
