package dlpclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	setup "github.com/paloaltonetworks/scm-go"
)

func TestClientHeadersAndMethods(t *testing.T) {
	var seenServiceName, seenClientName, seenAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenServiceName = r.Header.Get("service-name")
		seenClientName = r.Header.Get("client-name")
		seenAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","name":"test"}`))
	}))
	defer srv.Close()

	sc := &setup.Client{
		Jwt:          "test-jwt-token",
		JwtExpiresAt: time.Now().Add(10 * time.Minute),
	}
	client := NewClient(Config{
		SetupClient: sc,
		HttpClient:  srv.Client(),
		Host:        srv.URL,
	})

	var out map[string]any
	err := client.Get(context.Background(), "/v2/api/data-patterns/123", nil, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seenServiceName != DefaultServiceName {
		t.Errorf("expected service-name %q, got %q", DefaultServiceName, seenServiceName)
	}
	if seenClientName != DefaultClientName {
		t.Errorf("expected client-name %q, got %q", DefaultClientName, seenClientName)
	}
	if seenAuth != "Bearer test-jwt-token" {
		t.Errorf("expected Authorization Bearer test-jwt-token, got %q", seenAuth)
	}
}

func TestClient429Retry(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt32(&calls, 1)
		if c < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limited"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := NewClient(Config{
		HttpClient: srv.Client(),
		Host:       srv.URL,
	})
	client.RetryDelay = 5 * time.Millisecond

	var out map[string]bool
	if err := client.Get(context.Background(), "/v2/api/data-profiles", nil, &out); err != nil {
		t.Fatalf("expected retry to succeed, got: %v", err)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestGetAllZeroAndOneBasedPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v2/api/data-patterns" {
			page := r.URL.Query().Get("page")
			if page == "0" {
				_, _ = w.Write([]byte(`{"content":[{"id":"1","name":"p1","type":"custom"},{"id":"2","name":"p2","type":"custom"}],"totalPages":2,"size":2,"last":false}`))
				return
			}
			_, _ = w.Write([]byte(`{"content":[{"id":"3","name":"p3","type":"custom"}],"totalPages":2,"size":2,"last":true}`))
			return
		}
		if r.URL.Path == "/v1/api/forward-profile" {
			page := r.URL.Query().Get("page_number")
			if page == "1" {
				_, _ = w.Write([]byte(`{"items":[{"id":"10","name":"f1","enabled":true}],"total_pages":2,"size":1}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[{"id":"11","name":"f2","enabled":false}],"total_pages":2,"size":1}`))
			return
		}
	}))
	defer srv.Close()

	client := NewClient(Config{HttpClient: srv.Client(), Host: srv.URL})

	patterns, err := GetAll[DataPatternDTO](context.Background(), client, "/v2/api/data-patterns", nil, PaginationZeroBased, 2, 10)
	if err != nil {
		t.Fatalf("unexpected GetAll zero-based error: %v", err)
	}
	if len(patterns) != 3 {
		t.Fatalf("expected 3 patterns, got %d", len(patterns))
	}

	forwards, err := GetAll[ForwardProfileDTO](context.Background(), client, "/v1/api/forward-profile", nil, PaginationOneBased, 1, 10)
	if err != nil {
		t.Fatalf("unexpected GetAll one-based error: %v", err)
	}
	if len(forwards) != 2 {
		t.Fatalf("expected 2 forward profiles, got %d", len(forwards))
	}
}

func TestDoMultipartAndValidators(t *testing.T) {
	var gotContentType, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"dict-1","name":"my-dict","category":"Confidential","original_file_name":"words.csv","region_name":"us"}`))
	}))
	defer srv.Close()

	client := NewClient(Config{HttpClient: srv.Client(), Host: srv.URL})
	dict := &DictionaryDTO{
		Name:             "my-dict",
		Category:         "Confidential",
		OriginalFileName: "words.csv",
		RegionName:       "us",
	}
	parts, errs, err := BuildDictionaryUpdateParts(dict, []string{"ProjectAlpha", "ProjectBeta"})
	if err != nil || len(errs) > 0 {
		t.Fatalf("unexpected BuildDictionaryUpdateParts error: %v (%v)", err, errs)
	}

	var resp DictionaryDTO
	if err := client.DoMultipart(context.Background(), http.MethodPut, "/v2/api/dictionaries/dict-1", nil, parts, &resp); err != nil {
		t.Fatalf("unexpected DoMultipart error: %v", err)
	}
	if !strings.HasPrefix(gotContentType, "multipart/form-data; boundary=") {
		t.Errorf("expected multipart/form-data content-type, got %q", gotContentType)
	}
	if !strings.Contains(gotBody, "keyword\nProjectAlpha\nProjectBeta\n") {
		t.Errorf("expected CSV header + keywords in body, got %q", gotBody)
	}
}

func TestNormalizeAndValidateDataProfile(t *testing.T) {
	op := "or"
	cnt := int64(5)
	profile := &DataProfileDTO{
		Name: "PCI-Test",
		DetectionRules: []DetectionRuleDTO{
			{
				RuleType: "expression_tree",
				ExpressionTree: &ExpressionTreeNodeDTO{
					OperatorType: &op,
					SubExpressions: []ExpressionTreeNodeDTO{
						{
							RuleItem: &RuleItemDTO{
								ID:                     "pattern-1",
								DetectionTechnique:     "regex",
								OccurrenceOperatorType: "more_than_equal_to",
								OccurrenceCount:        &cnt,
							},
						},
					},
				},
			},
		},
	}

	errs, err := NormalizeAndValidateDataProfile(profile)
	if err != nil || len(errs) > 0 {
		t.Fatalf("expected valid profile, got err=%v errs=%v", err, errs)
	}

	leaf := profile.DetectionRules[0].ExpressionTree.SubExpressions[0].RuleItem
	if leaf.MatchType == nil || *leaf.MatchType != "include" {
		t.Errorf("expected default match_type='include', got %v", leaf.MatchType)
	}
	if leaf.ConfidenceLevel == nil || *leaf.ConfidenceLevel != "high" {
		t.Errorf("expected default confidence_level='high', got %v", leaf.ConfidenceLevel)
	}
	if leaf.ByUniqueCount == nil || *leaf.ByUniqueCount != false {
		t.Errorf("expected default by_unique_count=false, got %v", leaf.ByUniqueCount)
	}

	// Verify JSON marshaling preserves normalized defaults
	b, _ := json.Marshal(profile)
	if !strings.Contains(string(b), `"match_type":"include"`) {
		t.Errorf("expected marshaled JSON to include match_type: %s", string(b))
	}
}

func TestSecurityHardening(t *testing.T) {
	// 1. ValidateResourceID & EscapeID (DLP-003 path traversal hardening)
	if err := ValidateResourceID("../admin"); err == nil {
		t.Errorf("expected traversal ID '../admin' to fail validation")
	}
	if err := ValidateResourceID("123/456"); err == nil {
		t.Errorf("expected slash ID '123/456' to fail validation")
	}
	if err := ValidateResourceID("valid-id_123"); err != nil {
		t.Errorf("expected 'valid-id_123' to be valid, got: %v", err)
	}
	if got := EscapeID("a/b?c=1"); got != "a%2Fb%3Fc=1" {
		t.Errorf("expected URL-escaped ID, got %q", got)
	}

	// 2. Reject external origin URLs in buildURL (DLP-002 token exfiltration hardening)
	client := NewClient(Config{Host: "https://api.dlp.paloaltonetworks.com"})
	if _, err := client.buildURL("https://evil.example.com/steal", nil); err == nil {
		t.Errorf("expected external origin URL to be rejected by buildURL")
	}
	if u, err := client.buildURL("https://api.dlp.paloaltonetworks.com/v2/api/ocr", nil); err != nil || u == "" {
		t.Errorf("expected same-origin URL to be allowed, got u=%q err=%v", u, err)
	}

	// 3. Cross-origin redirect strips Authorization header
	var redirectSeenAuth string
	targetSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectSeenAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer targetSrv.Close()

	originSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, targetSrv.URL+"/landed", http.StatusFound)
	}))
	defer originSrv.Close()

	setupCli := &setup.Client{
		Jwt:          "secret-bearer-token",
		JwtExpiresAt: time.Now().Add(1 * time.Hour),
	}
	redirClient := NewClient(Config{
		SetupClient: setupCli,
		HttpClient:  originSrv.Client(),
		Host:        originSrv.URL,
	})
	var out map[string]any
	if err := redirClient.Get(context.Background(), "/v1/api/redirect-me", nil, &out); err != nil {
		t.Fatalf("unexpected redirect error: %v", err)
	}
	if redirectSeenAuth != "" {
		t.Errorf("expected Authorization header to be stripped on cross-origin redirect, got %q", redirectSeenAuth)
	}
}
