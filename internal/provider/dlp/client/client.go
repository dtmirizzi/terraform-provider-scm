package dlpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	setup "github.com/paloaltonetworks/scm-go"
)

const (
	DefaultDLPHost         = "api.dlp.paloaltonetworks.com"
	DefaultServiceName     = "prisma-access"
	DefaultClientName      = "terraform-provider-scm"
	DefaultMaxRetries      = 3
	DefaultRetryDelay      = 1000 * time.Millisecond
	DefaultMaxRetryAfter   = 60 * time.Second
	DefaultRequestTimeout  = 60 * time.Second
	MaxErrorResponseLength = 4096
	// MaxResponseSize caps how much of a DLP API response body is read into
	// memory, so a misbehaving or hostile endpoint cannot exhaust memory.
	MaxResponseSize = 32 * 1024 * 1024
)

var validResourceIDRe = regexp.MustCompile(`^[A-Za-z0-9._~:-]{1,128}$`)

// EscapeID URL-escapes a resource identifier for safe interpolation into a
// request path, so IDs containing '/', '?', '#' or '..' cannot change the target
// endpoint (DLP-003). Use ValidateResourceID to reject such IDs outright.
func EscapeID(id string) string {
	return url.PathEscape(strings.TrimSpace(id))
}

// ValidateResourceID checks that a resource ID does not contain path traversal or query/fragment characters.
func ValidateResourceID(id string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return errors.New("resource ID must not be empty")
	}
	if trimmed == "." || trimmed == ".." || strings.ContainsAny(trimmed, "/\\?#%") || !validResourceIDRe.MatchString(trimmed) {
		return fmt.Errorf("invalid resource ID %q", id)
	}
	return nil
}

// APIError represents an error returned by the Enterprise DLP API.
type APIError struct {
	Method       string
	URL          string
	StatusCode   int
	ResponseBody string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("DLP API %s %s failed (%d): %s", e.Method, e.URL, e.StatusCode, safeBody(e.ResponseBody))
}

// safeBody redacts secret-like fields and truncates a response body so it can
// be included in an error or log message.
func safeBody(body string) string {
	body = RedactBody(body)
	if len(body) > MaxErrorResponseLength {
		body = body[:MaxErrorResponseLength] + "...(truncated)"
	}
	return body
}

// IsNotFound returns true if the error is an HTTP 404 Not Found.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
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

	// Clone the http.Client so we can attach redirect protection and a default
	// timeout without mutating the caller's (or SetupClient's) client.
	var cloned http.Client
	if cfg.HttpClient != nil {
		cloned = *cfg.HttpClient
	} else if cfg.SetupClient != nil && cfg.SetupClient.HttpClient != nil {
		cloned = *cfg.SetupClient.HttpClient
	}
	if cloned.Timeout == 0 {
		cloned.Timeout = DefaultRequestTimeout
	}
	baseParsed, _ := url.Parse(baseURL)
	cloned.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		// Refuse redirects to any other origin (DLP-002). Following them would
		// forward custom provider headers and, for 307/308, replay the request
		// body (which may contain uploaded files) to a host we did not choose.
		if baseParsed == nil || !sameOrigin(req.URL, baseParsed) {
			return fmt.Errorf("refusing cross-origin redirect from %s to %s://%s", via[0].URL.Host, req.URL.Scheme, req.URL.Host)
		}
		return nil
	}
	httpClient := &cloned

	userAgent := cfg.UserAgent
	if userAgent == "" && cfg.SetupClient != nil && cfg.SetupClient.Agent != "" {
		userAgent = cfg.SetupClient.Agent
	}
	if userAgent == "" {
		userAgent = "Terraform-Provider-SCM-DLP"
	}

	// Give the DLP client its own copy of the setup client. scm-go reads and
	// writes Jwt/JwtExpiresAt without a lock, so sharing one *setup.Client
	// with the SCM API clients races whenever both refresh the token. The copy
	// keeps the same credentials, headers and HTTP client but owns its token
	// state, which getToken guards with c.mu. This must run during provider
	// configuration, before any API client can be refreshing the token.
	var setupClient *setup.Client
	if cfg.SetupClient != nil {
		sc := *cfg.SetupClient
		setupClient = &sc
	}

	return &Client{
		SetupClient: setupClient,
		HttpClient:  httpClient,
		BaseURL:     baseURL,
		ServiceName: serviceName,
		ClientName:  clientName,
		UserAgent:   userAgent,
		MaxRetries:  DefaultMaxRetries,
		RetryDelay:  DefaultRetryDelay,
	}
}

func (c *Client) getToken(ctx context.Context, forceRefresh bool) (string, error) {
	if c.SetupClient == nil {
		return "", nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if forceRefresh || c.SetupClient.Jwt == "" || (!c.SetupClient.JwtExpiresAt.IsZero() && time.Now().After(c.SetupClient.JwtExpiresAt)) {
		if err := c.SetupClient.RefreshJwt(ctx); err != nil {
			return "", fmt.Errorf("failed to refresh SASE JWT: %w", err)
		}
	}
	if c.SetupClient.Jwt == "" {
		return "", errors.New("SASE JWT token is empty after authentication")
	}
	return c.SetupClient.Jwt, nil
}

func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}

// isLoopbackHost reports whether host (without port) is localhost or a
// loopback IP address.
func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// checkTransportSecurity refuses to send the bearer token over plaintext HTTP
// to anything other than a loopback address (local mocks and tests).
func checkTransportSecurity(u *url.URL) error {
	switch strings.ToLower(u.Scheme) {
	case "https":
		return nil
	case "http":
		if isLoopbackHost(u.Hostname()) {
			return nil
		}
		return fmt.Errorf("refusing to send authenticated DLP request over plaintext HTTP to %q; use https (check dlp_host and the provider protocol setting)", u.Host)
	default:
		return fmt.Errorf("unsupported DLP URL scheme %q", u.Scheme)
	}
}

func (c *Client) buildURL(path string, params map[string]string) (string, error) {
	var rawURL string
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		baseParsed, baseErr := url.Parse(c.BaseURL)
		targetParsed, targetErr := url.Parse(path)
		if baseErr != nil || targetErr != nil || !strings.EqualFold(baseParsed.Scheme, targetParsed.Scheme) || !strings.EqualFold(baseParsed.Host, targetParsed.Host) {
			return "", fmt.Errorf("refusing to send authenticated DLP request to external origin %q", path)
		}
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
	if err := checkTransportSecurity(parsed); err != nil {
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
		dispParams := map[string]string{"name": part.FieldName}
		if part.FileName != "" {
			dispParams["filename"] = part.FileName
		}
		disp := mime.FormatMediaType("form-data", dispParams)
		if disp == "" {
			return fmt.Errorf("invalid multipart field or file name %q / %q", part.FieldName, part.FileName)
		}
		h.Set("Content-Disposition", disp)
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

	reqCtx := ctx
	var cancel context.CancelFunc
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		reqCtx, cancel = context.WithTimeout(ctx, DefaultRequestTimeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(reqCtx, method, fullURL, bodyReader)
	if err != nil {
		return err
	}

	if c.SetupClient != nil && len(c.SetupClient.Headers) > 0 {
		for k, v := range c.SetupClient.Headers {
			req.Header.Set(k, v)
		}
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

	token, err := c.getToken(reqCtx, false)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize+1))
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if len(respBytes) > MaxResponseSize {
		return fmt.Errorf("DLP API %s %s response exceeds %d MB limit", method, fullURL, MaxResponseSize/(1024*1024))
	}

	if resp.StatusCode == http.StatusUnauthorized && c.SetupClient != nil && attempt == 0 {
		if _, refreshErr := c.getToken(reqCtx, true); refreshErr == nil {
			return c.doWithRetry(ctx, method, path, params, contentType, bodyBytes, out, attempt+1)
		}
	}

	isIdempotent := method == http.MethodGet || method == http.MethodPut || method == http.MethodDelete
	shouldRetry5xx := isIdempotent && (resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusGatewayTimeout)
	if (resp.StatusCode == http.StatusTooManyRequests || shouldRetry5xx) && attempt < c.MaxRetries {
		delay := c.RetryDelay * time.Duration(1<<attempt)
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if secs, parseErr := strconv.Atoi(retryAfter); parseErr == nil && secs > 0 {
				delay = time.Duration(secs) * time.Second
			} else if retryTime, httpErr := http.ParseTime(retryAfter); httpErr == nil {
				if d := time.Until(retryTime); d > 0 {
					delay = d
				}
			}
		}
		if delay > DefaultMaxRetryAfter {
			delay = DefaultMaxRetryAfter
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			return c.doWithRetry(ctx, method, path, params, contentType, bodyBytes, out, attempt+1)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
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
		return fmt.Errorf("failed to unmarshal response from %s %s: %w (body: %s)", method, fullURL, err, safeBody(string(respBytes)))
	}
	return nil
}

// PaginationStyle specifies 0-based (Spring v2) or 1-based (Spring v1) pagination.
type PaginationStyle string

const (
	PaginationZeroBased PaginationStyle = "zero"
	PaginationOneBased  PaginationStyle = "one"
)

type PaginatedPageMeta struct {
	Size          *int `json:"size"`
	Number        *int `json:"number"`
	TotalElements *int `json:"total_elements"`
	TotalPages    *int `json:"total_pages"`
}

// PaginatedEnvelope captures both 0-based and 1-based paginated responses from DLP APIs.
type PaginatedEnvelope[T any] struct {
	Content             []T                `json:"content"`
	Items               []T                `json:"items"`
	Data                []T                `json:"data"`
	Resources           []T                `json:"resources"`
	Page                *PaginatedPageMeta `json:"page"`
	TotalPages          *int               `json:"totalPages"`
	TotalPagesSnake     *int               `json:"total_pages"`
	TotalElements       *int               `json:"totalElements"`
	TotalElemSnake      *int               `json:"total_elements"`
	Total               *int               `json:"total"`
	Last                *bool              `json:"last"`
	Size                *int               `json:"size"`
	NumberOfElements    *int               `json:"numberOfElements"`
	NumberOfElementsSnk *int               `json:"number_of_elements"`
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
		if totalPages == nil && env.Page != nil {
			totalPages = env.Page.TotalPages
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
		if totalElements == nil && env.Page != nil {
			totalElements = env.Page.TotalElements
		}
		if totalElements != nil && len(all) >= *totalElements {
			break
		}

		actualPageSize := pageSize
		if env.Size != nil && *env.Size > 0 {
			actualPageSize = *env.Size
		} else if env.Page != nil && env.Page.Size != nil && *env.Page.Size > 0 {
			actualPageSize = *env.Page.Size
		} else if env.NumberOfElements != nil && *env.NumberOfElements > 0 {
			actualPageSize = *env.NumberOfElements
		} else if env.NumberOfElementsSnk != nil && *env.NumberOfElementsSnk > 0 {
			actualPageSize = *env.NumberOfElementsSnk
		}

		if len(items) == 0 || len(items) < actualPageSize {
			break
		}

		page++
	}

	return all, nil
}

// sensitiveFieldFragments are case-insensitive fragments of JSON field names
// whose values are masked before a response body is surfaced in an error or
// log (CWE-532). Mirrors the SCM provider's HTTP log redaction and adds
// DLP-specific fields such as syslog certificates.
var sensitiveFieldFragments = []string{
	"passphrase", "password", "passwd", "secret", "token", "credential",
	"private", "psk", "pre_shared", "preshared", "authorization", "cookie",
	"api_key", "apikey", "auth_key", "jwt", "cert",
}

func isSensitiveField(name string) bool {
	n := strings.ToLower(name)
	if strings.HasSuffix(n, "key") {
		return true
	}
	for _, frag := range sensitiveFieldFragments {
		if strings.Contains(n, frag) {
			return true
		}
	}
	return false
}

func redactValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if isSensitiveField(k) {
				t[k] = "****"
			} else {
				t[k] = redactValue(val)
			}
		}
	case []any:
		for i, val := range t {
			t[i] = redactValue(val)
		}
	}
	return v
}

// RedactBody masks secret-like fields in a JSON body. Non-JSON bodies are
// returned unchanged (DLP error bodies are typically short plain-text or JSON
// error messages); callers still truncate them.
func RedactBody(body string) string {
	dec := json.NewDecoder(strings.NewReader(body))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return body
	}
	out, err := json.Marshal(redactValue(v))
	if err != nil {
		return body
	}
	return string(out)
}
