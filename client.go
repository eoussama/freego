package freego

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultTimeout is the request timeout of the HTTP client used when none is
// provided with [WithHTTPClient].
const DefaultTimeout = 30 * time.Second

// Client is a client for the FreeStuff REST API v2. It is safe for concurrent
// use by multiple goroutines.
type Client struct {
	apiKey            string
	baseURL           string
	httpClient        *http.Client
	userAgent         string
	compatibilityDate string
}

// Option configures a [Client].
type Option func(*Client)

// WithBaseURL overrides the API base URL. It defaults to [DefaultBaseURL] and
// is mostly useful for testing.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = baseURL }
}

// WithHTTPClient sets the HTTP client used to make requests. The default
// client uses a [DefaultTimeout] timeout.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.httpClient = httpClient }
}

// WithUserAgent prepends product to the User-Agent header sent with every
// request, for example "my-bot/1.2 (https://example.com)". The library's own
// identifier is always kept.
func WithUserAgent(product string) Option {
	return func(c *Client) {
		if product = strings.TrimSpace(product); product != "" {
			c.userAgent = product + " " + ClientLibrary
		}
	}
}

// WithCompatibilityDate overrides the compatibility date (YYYY-MM-DD) sent
// with every request. It defaults to [CompatibilityDate], the date the types
// in this package are built for; only change it if you know the data model of
// the date you pick decodes into these types.
func WithCompatibilityDate(date string) Option {
	return func(c *Client) { c.compatibilityDate = date }
}

// New returns a client authenticated with the given REST API key, which you
// can find on https://dashboard.freestuffbot.xyz/ under "my application".
func New(apiKey string, opts ...Option) (*Client, error) {
	c := &Client{
		apiKey:            strings.TrimSpace(apiKey),
		baseURL:           DefaultBaseURL,
		userAgent:         ClientLibrary,
		compatibilityDate: CompatibilityDate,
	}
	for _, opt := range opts {
		opt(c)
	}

	if c.apiKey == "" {
		return nil, errors.New("freego: missing API key")
	}

	u, err := url.Parse(c.baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("freego: invalid base URL %q", c.baseURL)
	}
	c.baseURL = strings.TrimRight(c.baseURL, "/")

	if _, err := time.Parse("2006-01-02", c.compatibilityDate); err != nil {
		return nil, fmt.Errorf("freego: invalid compatibility date %q, expected YYYY-MM-DD", c.compatibilityDate)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: DefaultTimeout}
	}

	return c, nil
}

// response holds what callers need from a successful response besides the
// decoded body.
type response struct {
	etag string
}

// get performs a GET request on path (relative to the base URL, starting
// with "/") and decodes the JSON response into out, if out is not nil.
func (c *Client) get(ctx context.Context, path string, query url.Values, ifNoneMatch string, out any) (*response, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("freego: building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-Compatibility-Date", c.compatibilityDate)
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("freego: GET %s: %w", path, err)
	}
	defer func() {
		// Drain so the connection can be reused.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		resp.Body.Close()
	}()

	switch {
	case resp.StatusCode == http.StatusNotModified:
		return nil, ErrNotModified
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return nil, problemFromResponse(resp)
	}

	r := &response{etag: resp.Header.Get("ETag")}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return r, nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, fmt.Errorf("freego: decoding response of GET %s: %w", path, err)
	}
	return r, nil
}
