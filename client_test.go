package freego

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient starts a server running handler and returns a client for it.
func newTestClient(t *testing.T, handler http.HandlerFunc, opts ...Option) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := New("test-key", append([]Option{WithBaseURL(srv.URL)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func TestNewValidation(t *testing.T) {
	for _, tt := range []struct {
		name string
		key  string
		opts []Option
		want string
	}{
		{"empty key", "  ", nil, "missing API key"},
		{"relative base URL", "k", []Option{WithBaseURL("api/v2")}, "invalid base URL"},
		{"base URL without host", "k", []Option{WithBaseURL("https://")}, "invalid base URL"},
		{"bad scheme", "k", []Option{WithBaseURL("ftp://example.com")}, "invalid base URL"},
		{"bad compatibility date", "k", []Option{WithCompatibilityDate("2026-6-8")}, "invalid compatibility date"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.key, tt.opts...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want %q", err, tt.want)
			}
		})
	}

	c, err := New("k")
	if err != nil {
		t.Fatal(err)
	}
	if c.baseURL != DefaultBaseURL || c.compatibilityDate != CompatibilityDate || c.httpClient.Timeout != DefaultTimeout {
		t.Errorf("unexpected defaults: %+v", c)
	}
}

func TestRequestHeaders(t *testing.T) {
	var got *http.Request
	var body []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		body, _ = io.ReadAll(r.Body)
		writeJSON(w, 200, map[string]any{"list": []any{}})
	}, WithUserAgent("my-bot/1.0"), WithCompatibilityDate("2026-04-01"))

	if _, err := c.Schemas(context.Background()); err != nil {
		t.Fatal(err)
	}

	for header, want := range map[string]string{
		"Authorization":        "Bearer test-key",
		"Accept":               "application/json",
		"User-Agent":           "my-bot/1.0 " + ClientLibrary,
		"X-Compatibility-Date": "2026-04-01",
	} {
		if v := got.Header.Get(header); v != want {
			t.Errorf("%s = %q, want %q", header, v, want)
		}
	}
	if got.Method != http.MethodGet || got.URL.Path != "/static/schemas" || len(body) != 0 {
		t.Errorf("unexpected request: %s %s body=%q", got.Method, got.URL.Path, body)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestWithHTTPClient(t *testing.T) {
	var calls int
	hc := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != DefaultBaseURL+"/static/schemas" {
			t.Errorf("URL = %s", r.URL)
		}
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"list":[]}`)),
		}, nil
	})}

	c, err := New("k", WithHTTPClient(hc))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Ping(context.Background()); err != nil || calls != 1 {
		t.Errorf("err = %v, calls = %d", err, calls)
	}
}

func TestBaseURLTrailingSlash(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		writeJSON(w, 200, map[string]any{"list": []any{}})
	}))
	defer srv.Close()

	c, err := New("k", WithBaseURL(srv.URL+"/v2/"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Problems(context.Background()); err != nil {
		t.Fatal(err)
	}
	if path != "/v2/static/problems" {
		t.Errorf("path = %q", path)
	}
}

func TestPing(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/static/schemas" {
			t.Errorf("Ping requested %s", r.URL.Path)
		}
		writeJSON(w, 200, map[string]any{"type": "fsb:static:apiv2:schema_list", "list": []any{}})
	})
	if err := c.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}

	bad := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 401, map[string]any{"type": ProblemTypeUnauthorized, "title": "Authorization header missing or invalid"})
	})
	if err := bad.Ping(context.Background()); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("err = %v, want ErrUnauthorized", err)
	}
}

func TestStaticEndpoints(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/static/schemas":
			writeJSON(w, 200, map[string]any{"type": "fsb:static:apiv2:schema_list", "list": []any{
				map[string]any{"name": "Product", "urn": SchemaProduct, "active": true, "latestVersion": "2026-06-08"},
			}})
		case "/static/schemas/fsb:schema:apiv2:product":
			writeJSON(w, 200, map[string]any{"type": "fsb:static:apiv2:schema", "schema": map[string]any{"type": "object"}})
		case "/static/problems":
			writeJSON(w, 200, map[string]any{"list": []any{map[string]any{"urn": ProblemTypeProductNotFound, "status": 40002, "title": "x"}}})
		case "/static/events":
			writeJSON(w, 200, map[string]any{"list": []any{map[string]any{"urn": "fsb:event:ping", "description": "d", "payloadDescription": "p", "payloadSchema": map[string]any{"type": "object"}}}})
		default:
			t.Errorf("unexpected path %s", r.URL.EscapedPath())
			w.WriteHeader(404)
		}
	})
	ctx := context.Background()

	schemas, err := c.Schemas(ctx)
	if err != nil || len(schemas) != 1 || schemas[0] != (SchemaInfo{Name: "Product", URN: SchemaProduct, Active: true, LatestVersion: "2026-06-08"}) {
		t.Errorf("Schemas = %+v, %v", schemas, err)
	}
	schema, err := c.Schema(ctx, SchemaProduct)
	if err != nil || strings.TrimSpace(string(schema)) != `{"type":"object"}` {
		t.Errorf("Schema = %s, %v", schema, err)
	}
	problems, err := c.Problems(ctx)
	if err != nil || len(problems) != 1 || problems[0].Status != 40002 {
		t.Errorf("Problems = %+v, %v", problems, err)
	}
	events, err := c.Events(ctx)
	if err != nil || len(events) != 1 || events[0].URN != "fsb:event:ping" || len(events[0].PayloadSchema) == 0 {
		t.Errorf("Events = %+v, %v", events, err)
	}
}

func TestProblemResponses(t *testing.T) {
	t.Run("problem document", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/problem+json")
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(429)
			_, _ = io.WriteString(w, `{"type":"fsb:problem:api:rate_limit_exceeded","title":"Too many requests","detail":"slow down","extra":1}`)
		})
		err := c.Ping(context.Background())

		var p *Problem
		if !errors.As(err, &p) {
			t.Fatalf("err = %T %v, want *Problem", err, err)
		}
		if p.Status != 429 || p.Type != ProblemTypeRateLimitExceeded || p.Title != "Too many requests" || p.Detail != "slow down" || p.RetryAfter != 7*time.Second {
			t.Errorf("unexpected problem: %+v", p)
		}
		if !strings.Contains(string(p.Raw), `"extra":1`) {
			t.Errorf("Raw = %s", p.Raw)
		}
		if !errors.Is(err, ErrRateLimited) || errors.Is(err, ErrUnauthorized) {
			t.Error("errors.Is mismatch")
		}
	})

	t.Run("non-JSON body", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(502)
			_, _ = io.WriteString(w, "<html>"+strings.Repeat("x", 500)+"</html>")
		})
		err := c.Ping(context.Background())

		var p *Problem
		if !errors.As(err, &p) {
			t.Fatalf("err = %T %v, want *Problem", err, err)
		}
		if p.Status != 502 || p.Type != "about:blank" || p.Title != "Bad Gateway" || len(p.Detail) > 210 {
			t.Errorf("unexpected problem: %+v", p)
		}
	})

	t.Run("not found by status", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 404, map[string]any{"type": ProblemTypeProductNotFound, "title": "not found"})
		})
		_, err := c.Product(context.Background(), 1)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestMalformedSuccessBody(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "{not json")
	})
	_, err := c.Schemas(context.Background())
	var p *Problem
	if err == nil || errors.As(err, &p) || !strings.Contains(err.Error(), "decoding response") {
		t.Errorf("err = %v", err)
	}
}

func TestContextCancellation(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := c.Ping(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestProblemIsAndError(t *testing.T) {
	p := &Problem{Type: ProblemTypeUnavailableForFreeTier, Title: "Not on your plan", Detail: "Upgrade.", Status: 403}
	if !errors.Is(p, ErrUnavailableForFreeTier) || errors.Is(p, ErrNotFound) || errors.Is(p, &Problem{}) || errors.Is(p, errors.New("x")) {
		t.Error("Is mismatch")
	}
	if !p.Is(&Problem{Type: ProblemTypeUnavailableForFreeTier, Status: 403}) || p.Is(&Problem{Type: ProblemTypeUnavailableForFreeTier, Status: 401}) {
		t.Error("Is with type and status")
	}
	want := "freestuff: Not on your plan (403, fsb:problem:api:unavailable_for_free_tier): Upgrade."
	if p.Error() != want {
		t.Errorf("Error() = %q, want %q", p.Error(), want)
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	for in, want := range map[string]time.Duration{
		"":                              0,
		"30":                            30 * time.Second,
		"-5":                            0,
		"soon":                          0,
		"Thu, 01 Oct 2026 12:01:00 GMT": time.Minute,
		"Thu, 01 Oct 2026 11:00:00 GMT": 0,
	} {
		if got := parseRetryAfter(in, now); got != want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestProblemIsTypedNil(t *testing.T) {
	var target *Problem
	err := error(&Problem{Type: ProblemTypeUnauthorized, Status: 401})
	if errors.Is(err, target) {
		t.Error("matched a typed nil")
	}
}
