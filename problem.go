package freego

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Problem types returned by the FreeStuff API. The full, current list is
// available from [Client.Problems].
const (
	ProblemTypeBadRequest               = "fsb:problem:generic:bad_request"
	ProblemTypeUnauthorized             = "fsb:problem:api:unauthorized"
	ProblemTypeNoAccess                 = "fsb:problem:api:no_access"
	ProblemTypeUnavailableForFreeTier   = "fsb:problem:api:unavailable_for_free_tier"
	ProblemTypeUnknownEndpoint          = "fsb:problem:api:unknown_endpoint"
	ProblemTypeMalformedQueryParameter  = "fsb:problem:api:malformed_query_parameter"
	ProblemTypeInvalidCompatibilityDate = "fsb:problem:api:invalid_compatibility_date"
	ProblemTypeResourceNotFound         = "fsb:problem:api:resource_not_found"
	ProblemTypeRateLimitExceeded        = "fsb:problem:api:rate_limit_exceeded"
	ProblemTypeProductNotFound          = "fsb:problem:data:product_not_found"
	ProblemTypeProductListCacheMiss     = "fsb:problem:data:product_list_cache_miss"
	ProblemTypeAnnouncementNotFound     = "fsb:problem:data:announcement_not_found"
	ProblemTypeFeatureShutdown          = "fsb:problem:generic:feature_shutdown"
	ProblemTypeBadGateway               = "fsb:problem:generic:bad_gateway"
	ProblemTypeUnknown                  = "fsb:problem:generic:unknown"
)

const (
	// problemTypeBlank is the RFC 7807 type used for responses that are not
	// problem documents.
	problemTypeBlank = "about:blank"
	// maxProblemBodyBytes caps how much of an error response is read.
	maxProblemBodyBytes int64 = 64 << 10
)

// Sentinel errors for use with errors.Is. A [*Problem] matches a sentinel
// when its Type is equal to the sentinel's Type, or, for sentinels that only
// carry a Status (such as [ErrNotFound]), when the HTTP status codes match.
var (
	// ErrUnauthorized means the API key is missing, invalid or has been reset.
	ErrUnauthorized = &Problem{Type: ProblemTypeUnauthorized}
	// ErrNoAccess means the API key may not access the requested resource.
	ErrNoAccess = &Problem{Type: ProblemTypeNoAccess}
	// ErrUnavailableForFreeTier means the endpoint requires a paid plan.
	// Content endpoints such as [Client.Products] are only available on the
	// "full" tier.
	ErrUnavailableForFreeTier = &Problem{Type: ProblemTypeUnavailableForFreeTier}
	// ErrRateLimited means too many requests were made. See
	// [Problem.RetryAfter].
	ErrRateLimited = &Problem{Type: ProblemTypeRateLimitExceeded}
	// ErrNotFound matches any problem returned with HTTP status 404.
	ErrNotFound = &Problem{Status: http.StatusNotFound}
)

// ErrNotModified is returned when a request sent with an ETag in
// If-None-Match is answered with 304 Not Modified, meaning the data you
// already have is still current.
var ErrNotModified = errors.New("freego: not modified")

// Problem is an error returned by the FreeStuff API, decoded from an RFC 7807
// problem document. Responses that are not problem documents (for example an
// HTML error page from a proxy) are also reported as a Problem, with Type
// "about:blank" and the HTTP status text as Title.
type Problem struct {
	// Type is a URN identifying the problem, such as
	// "fsb:problem:api:unauthorized".
	Type string `json:"type"`
	// Title is a short, human readable summary of the problem type.
	Title string `json:"title"`
	// Detail is a human readable explanation specific to this occurrence.
	Detail string `json:"detail,omitempty"`
	// Status is the HTTP status code of the response.
	Status int `json:"-"`
	// RetryAfter is the delay requested by the Retry-After response header,
	// or 0 when the header is absent.
	RetryAfter time.Duration `json:"-"`
	// Raw is the response body, truncated to 64 KiB. Problem documents may
	// carry extra members (for example "provided" and "location" on
	// invalid_compatibility_date) that can be read from it.
	Raw []byte `json:"-"`
}

// Error implements the error interface.
func (p *Problem) Error() string {
	var b strings.Builder
	b.WriteString("freestuff: ")
	if p.Title != "" {
		b.WriteString(p.Title)
	} else {
		b.WriteString("request failed")
	}
	b.WriteString(" (")
	if p.Status != 0 {
		b.WriteString(strconv.Itoa(p.Status))
		b.WriteString(", ")
	}
	b.WriteString(p.Type)
	b.WriteString(")")
	if p.Detail != "" {
		b.WriteString(": ")
		b.WriteString(p.Detail)
	}
	return b.String()
}

// Is reports whether p matches target, enabling errors.Is checks against the
// sentinel problems such as [ErrUnauthorized] and [ErrNotFound].
func (p *Problem) Is(target error) bool {
	t, ok := target.(*Problem)
	if !ok || t == nil {
		return false
	}
	if t.Type != "" && t.Type != p.Type {
		return false
	}
	if t.Status != 0 && t.Status != p.Status {
		return false
	}
	return t.Type != "" || t.Status != 0
}

// problemFromResponse builds a Problem from a non-successful response.
func problemFromResponse(resp *http.Response) *Problem {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxProblemBodyBytes))

	p := &Problem{}
	if err := json.Unmarshal(raw, p); err != nil || p.Type == "" {
		p = &Problem{
			Type:   problemTypeBlank,
			Title:  http.StatusText(resp.StatusCode),
			Detail: strings.TrimSpace(string(raw)),
		}
		if len(p.Detail) > 200 {
			p.Detail = p.Detail[:200] + "..."
		}
	}

	p.Status = resp.StatusCode
	p.Raw = raw
	p.RetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
	return p
}

// parseRetryAfter parses a Retry-After header value given either as a number
// of seconds or as an HTTP date.
func parseRetryAfter(v string, now time.Time) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := t.Sub(now); d > 0 {
			return d
		}
	}
	return 0
}
