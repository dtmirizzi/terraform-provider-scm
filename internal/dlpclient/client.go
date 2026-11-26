package dlpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	setup "github.com/paloaltonetworks/scm-go"
)

const (
	DefaultDLPHost        = "api.dlp.paloaltonetworks.com"
	DefaultServiceName    = "prisma-access"
	DefaultClientName     = "terraform-provider-scm"
	DefaultMaxRetries     = 3
	DefaultRetryDelay     = 1000 * time.Millisecond
	DefaultMaxRetryAfter  = 60 * time.Second
	DefaultRequestTimeout = 60 * time.Second
)

// APIError represents an error returned by the Enterprise DLP API.
type APIError struct {
	Method       string
	URL          string
	StatusCode   int
	ResponseBody string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("DLP API %s %s failed (%d): %s", e.Method, e.URL, e.StatusCode, e.ResponseBody)
}

// IsNotFound returns true if the error is an HTTP 404 Not Found.
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// MultipartPart describes a part in a multipart/form-data request.
type MultipartPart struct {
	FieldName   string
	FileName    string
	ContentType string
	Data        []byte
}

// Client is the HTTP client for Enterprise DLP public APIs.
type Client struct {
	SetupClient *setup.Client
	HttpClient  *http.Client
	BaseURL     string
	ServiceName string
	ClientName  string
	UserAgent   string
	MaxRetries  int
	RetryDelay  time.Duration
	mu          sync.Mutex
}

// Config configures a DLP Client.
type Config struct {
	SetupClient *setup.Client
	HttpClient  *http.Client
	Protocol    string
	Host        string
	ServiceName string
	ClientName  string
	UserAgent   string
}

// NewClient constructs a new Enterprise DLP Client, applying environment fallbacks if fields are empty.
func NewClient(cfg Config) *Client {
	protocol := strings.TrimSpace(cfg.Protocol)
	if protocol == "" {
		protocol = "https"
	}

	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		if envHost := strings.TrimSpace(os.Getenv("SCM_DLP_HOST")); envHost != "" {
			host = envHost
		} else if envHost := strings.TrimSpace(os.Getenv("DLP_HOST")); envHost != "" {
			host = envHost
		} else if envHost := strings.TrimSpace(os.Getenv("DLP_API_URL")); envHost != "" {
			host = envHost
		} else {
			host = DefaultDLPHost
		}
	}

	var baseURL string
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		baseURL = strings.TrimRight(host, "/")
	} else {
		baseURL = fmt.Sprintf("%s://%s", protocol, strings.TrimRight(host, "/"))
	}

	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		if envSvc := strings.TrimSpace(os.Getenv("SCM_DLP_SERVICE_NAME")); envSvc != "" {
			serviceName = envSvc
		} else if envSvc := strings.TrimSpace(os.Getenv("DLP_SERVICE_NAME")); envSvc != "" {
			serviceName = envSvc
		} else {
			serviceName = DefaultServiceName
		}
	}

	clientName := strings.TrimSpace(cfg.ClientName)
	if clientName == "" {
		clientName = DefaultClientName
	}

	httpClient := cfg.HttpClient
	if httpClient == nil && cfg.SetupClient != nil && cfg.SetupClient.HttpClient != nil {
		httpClient = cfg.SetupClient.HttpClient
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultRequestTimeout}
	}

	userAgent := cfg.UserAgent
	if userAgent == "" && cfg.SetupClient != nil && cfg.SetupClient.Agent != "" {
		userAgent = cfg.SetupClient.Agent
	}
	if userAgent == "" {
		userAgent = "Terraform-Provider-SCM-DLP"
	}

	return &Client{
		SetupClient: cfg.SetupClient,
		HttpClient:  httpClient,
		BaseURL:     baseURL,
		ServiceName: serviceName,
		ClientName:  clientName,
		UserAgent:   userAgent,
		MaxRetries:  DefaultMaxRetries,
		RetryDelay:  DefaultRetryDelay,
	}
}

func (c *Client) getToken(ctx context.Context) (string, error) {
	if c.SetupClient == nil {
		return "", nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.SetupClient.Jwt == "" || (!c.SetupClient.JwtExpiresAt.IsZero() && time.Now().After(c.SetupClient.JwtExpiresAt)) {
		if err := c.SetupClient.RefreshJwt(ctx); err != nil {
			return "", fmt.Errorf("failed to refresh SASE JWT: %w", err)
		}
	}
	return c.SetupClient.Jwt, nil
}

func (c *Client) buildURL(path string, params map[string]string) (string, error) {
	var rawURL string
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		rawURL = path
	} else {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		rawURL = c.BaseURL + path
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if len(params) > 0 {
		q := parsed.Query()
		for k, v := range params {
			if v != "" {
				q.Set(k, v)
			}
		}
		parsed.RawQuery = q.Encode()
	}
	return parsed.String(), nil
}

// DoJSON executes an HTTP request with an optional JSON request body and unmarshals the response into out.
func (c *Client) DoJSON(ctx context.Context, method, path string, params map[string]string, in any, out any) error {
	var bodyBytes []byte
	var err error
	if in != nil {
		bodyBytes, err = json.Marshal(in)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}
	return c.doWithRetry(ctx, method, path, params, "application/json", bodyBytes, out, 0)
}

// Get executes a GET request and unmarshals the JSON response into out.
func (c *Client) Get(ctx context.Context, path string, params map[string]string, out any) error {
	return c.DoJSON(ctx, http.MethodGet, path, params, nil, out)
}

// Post executes a POST request with a JSON body and unmarshals the JSON response into out.
func (c *Client) Post(ctx context.Context, path string, params map[string]string, in any, out any) error {
	return c.DoJSON(ctx, http.MethodPost, path, params, in, out)
}

// Put executes a PUT request with a JSON body and unmarshals the JSON response into out.
func (c *Client) Put(ctx context.Context, path string, params map[string]string, in any, out any) error {
	return c.DoJSON(ctx, http.MethodPut, path, params, in, out)
}

// Patch executes a PATCH request with a JSON body and unmarshals the JSON response into out.
func (c *Client) Patch(ctx context.Context, path string, params map[string]string, in any, out any) error {
	return c.DoJSON(ctx, http.MethodPatch, path, params, in, out)
}

// Delete executes a DELETE request.
func (c *Client) Delete(ctx context.Context, path string, params map[string]string) error {
	return c.DoJSON(ctx, http.MethodDelete, path, params, nil, nil)
}

// DoMultipart executes a POST or PUT multipart/form-data request.
func (c *Client) DoMultipart(ctx context.Context, method, path string, params map[string]string, parts []MultipartPart, out any) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	for _, part := range parts {
		h := make(textproto.MIMEHeader)
		if part.FileName != "" {
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, part.FieldName, part.FileName))
		} else {
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"`, part.FieldName))
		}
		if part.ContentType != "" {
			h.Set("Content-Type", part.ContentType)
		}
		pw, err := mw.CreatePart(h)
		if err != nil {
			return fmt.Errorf("failed to create multipart part %s: %w", part.FieldName, err)
		}
		if _, err := pw.Write(part.Data); err != nil {
			return fmt.Errorf("failed to write multipart part %s: %w", part.FieldName, err)
		}
	}

	if err := mw.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return c.doWithRetry(ctx, method, path, params, mw.FormDataContentType(), buf.Bytes(), out, 0)
}

func (c *Client) doWithRetry(
	ctx context.Context,
	method, path string,
	params map[string]string,
	contentType string,
	bodyBytes []byte,
	out any,
	attempt int,
) error {
	fullURL, err := c.buildURL(path, params)
	if err != nil {
		return err
	}

	var bodyReader io.Reader
	if bodyBytes != nil {
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	if c.ServiceName != "" {
		req.Header.Set("service-name", c.ServiceName)
	}
	if c.ClientName != "" {
		req.Header.Set("client-name", c.ClientName)
	}
	if bodyBytes != nil && contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	if c.SetupClient != nil && len(c.SetupClient.Headers) > 0 {
		for k, v := range c.SetupClient.Headers {
			req.Header.Set(k, v)
		}
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests && attempt < c.MaxRetries {
		delay := c.RetryDelay * time.Duration(1<<attempt)
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if secs, parseErr := strconv.Atoi(retryAfter); parseErr == nil && secs > 0 {
				delay = time.Duration(secs) * time.Second
			}
		}
		if delay > DefaultMaxRetryAfter {
			delay = DefaultMaxRetryAfter
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			return c.doWithRetry(ctx, method, path, params, contentType, bodyBytes, out, attempt+1)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized && c.SetupClient != nil {
			c.mu.Lock()
			c.SetupClient.Jwt = ""
			c.mu.Unlock()
		}
		return &APIError{
			Method:       method,
			URL:          fullURL,
			StatusCode:   resp.StatusCode,
			ResponseBody: string(respBytes),
		}
	}

	if out == nil || resp.StatusCode == http.StatusNoContent || len(bytes.TrimSpace(respBytes)) == 0 {
		return nil
	}

	if err := json.Unmarshal(respBytes, out); err != nil {
		return fmt.Errorf("failed to unmarshal response from %s %s: %w (body: %s)", method, fullURL, err, string(respBytes))
	}
	return nil
}

// PaginationStyle specifies 0-based (Spring v2) or 1-based (Spring v1) pagination.
type PaginationStyle string

const (
	PaginationZeroBased PaginationStyle = "zero"
	PaginationOneBased  PaginationStyle = "one"
)

// PaginatedEnvelope captures both 0-based and 1-based paginated responses from DLP APIs.
type PaginatedEnvelope[T any] struct {
	Content          []T   `json:"content"`
	Items            []T   `json:"items"`
	Data             []T   `json:"data"`
	Resources        []T   `json:"resources"`
	TotalPages       *int  `json:"totalPages"`
	TotalPagesSnake  *int  `json:"total_pages"`
	TotalElements    *int  `json:"totalElements"`
	TotalElemSnake   *int  `json:"total_elements"`
	Total            *int  `json:"total"`
	Last             *bool `json:"last"`
	Size             *int  `json:"size"`
	NumberOfElements *int  `json:"numberOfElements"`
}

// GetAll auto-paginates a GET endpoint returning a slice of items T.
func GetAll[T any](
	ctx context.Context,
	client *Client,
	path string,
	baseParams map[string]string,
	style PaginationStyle,
	pageSize int,
	maxPages int,
) ([]T, error) {
	if pageSize <= 0 {
		pageSize = 100
	}
	if maxPages <= 0 {
		maxPages = 50
	}
	if style == "" {
		style = PaginationZeroBased
	}

	var all []T
	page := 0
	if style == PaginationOneBased {
		page = 1
	}

	for i := 0; i < maxPages; i++ {
		params := make(map[string]string, len(baseParams)+2)
		for k, v := range baseParams {
			params[k] = v
		}
		if style == PaginationZeroBased {
			params["page"] = strconv.Itoa(page)
			params["size"] = strconv.Itoa(pageSize)
		} else {
			params["page_number"] = strconv.Itoa(page)
			params["page_size"] = strconv.Itoa(pageSize)
		}

		var raw json.RawMessage
		if err := client.Get(ctx, path, params, &raw); err != nil {
			return nil, err
		}

		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) == 0 {
			break
		}

		// Handle direct JSON array response
		if trimmed[0] == '[' {
			var arr []T
			if err := json.Unmarshal(trimmed, &arr); err != nil {
				return nil, err
			}
			all = append(all, arr...)
			break
		}

		var env PaginatedEnvelope[T]
		if err := json.Unmarshal(trimmed, &env); err != nil {
			return nil, err
		}

		var items []T
		switch {
		case env.Content != nil:
			items = env.Content
		case env.Items != nil:
			items = env.Items
		case env.Data != nil:
			items = env.Data
		case env.Resources != nil:
			items = env.Resources
		}

		all = append(all, items...)

		if env.Last != nil && *env.Last {
			break
		}

		totalPages := env.TotalPages
		if totalPages == nil {
			totalPages = env.TotalPagesSnake
		}
		if totalPages != nil {
			limit := *totalPages
			if style == PaginationZeroBased {
				limit = *totalPages - 1
			}
			if page >= limit {
				break
			}
		}

		totalElements := env.TotalElements
		if totalElements == nil {
			totalElements = env.TotalElemSnake
		}
		if totalElements == nil {
			totalElements = env.Total
		}
		if totalElements != nil && len(all) >= *totalElements {
			break
		}

		actualPageSize := pageSize
		if env.Size != nil && *env.Size > 0 {
			actualPageSize = *env.Size
		} else if env.NumberOfElements != nil && *env.NumberOfElements > 0 {
			actualPageSize = *env.NumberOfElements
		}

		if len(items) == 0 || len(items) < actualPageSize {
			break
		}

		page++
	}

	return all, nil
}
