package webhook

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eoussama/freego"
)

// recorder collects what the handler passes to the callbacks.
type recorder struct {
	mu            sync.Mutex
	pings         []*Ping
	announcements []*freego.ResolvedAnnouncement
	products      []*freego.Product
	unknown       []*Event
	errs          []error
	errEvents     []*Event
}

func (r *recorder) config() Config {
	return Config{
		OnPing: func(_ context.Context, _ *Event, p *Ping) error {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.pings = append(r.pings, p)
			return nil
		},
		OnAnnouncementCreated: func(_ context.Context, _ *Event, a *freego.ResolvedAnnouncement) error {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.announcements = append(r.announcements, a)
			return nil
		},
		OnProductUpdated: func(_ context.Context, _ *Event, p *freego.Product) error {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.products = append(r.products, p)
			return nil
		},
		OnUnknown: func(_ context.Context, e *Event) error {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.unknown = append(r.unknown, e)
			return nil
		},
		OnError: func(_ context.Context, e *Event, err error) {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.errs = append(r.errs, err)
			r.errEvents = append(r.errEvents, e)
		},
	}
}

type harness struct {
	t       *testing.T
	priv    ed25519.PrivateKey
	handler *Handler
	rec     *recorder
}

func newHarness(t *testing.T, mutate func(*Config)) *harness {
	t.Helper()
	pub, priv := newKeyPair(t)
	rec := &recorder{}
	cfg := rec.config()
	if mutate != nil {
		mutate(&cfg)
	}
	return &harness{t: t, priv: priv, handler: NewHandler(newTestVerifier(t, pub), cfg), rec: rec}
}

// request builds a signed delivery. compat "" omits the compatibility date
// header.
func (h *harness) request(id, compat, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	for k, v := range Sign(h.priv, id, fixedNow, []byte(body)) {
		r.Header[k] = v
	}
	r.Header.Set("Content-Type", "application/json")
	if compat != "" {
		r.Header.Set(HeaderCompatibilityDate, compat)
	}
	return r
}

func (h *harness) serve(r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.handler.ServeHTTP(w, r)
	return w
}

func envelopeJSON(eventType, data string) string {
	return `{"type":"` + eventType + `","timestamp":"2026-10-01T11:59:58.123Z","data":` + data + `}`
}

func productJSON(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../testdata/product_2026-06-08.json")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestHandlerDeliversEvents(t *testing.T) {
	h := newHarness(t, nil)
	product := productJSON(t)

	var gotEvent *Event
	h.handler.cfg.OnPing = func(_ context.Context, e *Event, p *Ping) error {
		gotEvent = e
		h.rec.pings = append(h.rec.pings, p)
		return nil
	}

	for _, req := range []*http.Request{
		h.request("msg_ping", freego.CompatibilityDate, envelopeJSON(EventPing, `{"manual":true}`)),
		h.request("msg_ann", freego.CompatibilityDate, envelopeJSON(EventAnnouncementCreated, `{"id":9,"products":[123456],"resolvedProducts":[`+product+`]}`)),
		h.request("msg_upd", freego.CompatibilityDate, envelopeJSON(EventProductUpdated, product)),
		h.request("msg_new", freego.CompatibilityDate, envelopeJSON("fsb:event:something_new", `{"x":1}`)),
	} {
		w := h.serve(req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("%s: status %d: %s", req.Header.Get(HeaderID), w.Code, w.Body)
		}
		if w.Header().Get(HeaderSetCompatibilityDate) != freego.CompatibilityDate || w.Header().Get(HeaderClientLibrary) != freego.ClientLibrary {
			t.Errorf("missing library headers: %v", w.Header())
		}
	}

	if len(h.rec.pings) != 1 || !h.rec.pings[0].Manual {
		t.Errorf("pings = %+v", h.rec.pings)
	}
	if gotEvent.ID != "msg_ping" || gotEvent.Type != EventPing || gotEvent.CompatibilityDate != freego.CompatibilityDate ||
		!gotEvent.Timestamp.Equal(time.Date(2026, 10, 1, 11, 59, 58, 123e6, time.UTC)) {
		t.Errorf("event = %+v", gotEvent)
	}
	if len(h.rec.announcements) != 1 || h.rec.announcements[0].ID != 9 || h.rec.announcements[0].ResolvedProducts[0].Title != "Example Game" {
		t.Errorf("announcements = %+v", h.rec.announcements)
	}
	if len(h.rec.products) != 1 || h.rec.products[0].ID != 123456 {
		t.Errorf("products = %+v", h.rec.products)
	}
	if len(h.rec.unknown) != 1 || h.rec.unknown[0].Type != "fsb:event:something_new" || string(h.rec.unknown[0].Data) != `{"x":1}` {
		t.Errorf("unknown = %+v", h.rec.unknown)
	}
	if len(h.rec.errs) != 0 {
		t.Errorf("errors = %v", h.rec.errs)
	}
}

func TestHandlerRejections(t *testing.T) {
	body := envelopeJSON(EventPing, `{"manual":false}`)

	tests := []struct {
		name    string
		mutate  func(h *harness) *http.Request
		status  int
		wantErr error
	}{
		{
			name: "GET",
			mutate: func(h *harness) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/webhook", nil)
			},
			status: http.StatusMethodNotAllowed,
		},
		{
			name: "missing headers",
			mutate: func(h *harness) *http.Request {
				return httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
			},
			status: http.StatusBadRequest, wantErr: ErrMissingHeaders,
		},
		{
			name: "tampered body",
			mutate: func(h *harness) *http.Request {
				r := h.request("m", freego.CompatibilityDate, body)
				r.Body = httptestBody(strings.Replace(body, "false", "true", 1))
				return r
			},
			status: http.StatusUnauthorized, wantErr: ErrInvalidSignature,
		},
		{
			name: "wrong key",
			mutate: func(h *harness) *http.Request {
				_, other := newKeyPair(h.t)
				r := h.request("m", freego.CompatibilityDate, body)
				r.Header.Set(HeaderSignature, Sign(other, "m", fixedNow, []byte(body)).Get(HeaderSignature))
				return r
			},
			status: http.StatusUnauthorized, wantErr: ErrInvalidSignature,
		},
		{
			name: "unsupported signature",
			mutate: func(h *harness) *http.Request {
				r := h.request("m", freego.CompatibilityDate, body)
				r.Header.Set(HeaderSignature, "v1,abc")
				return r
			},
			status: http.StatusUnauthorized, wantErr: ErrUnsupportedSignature,
		},
		{
			name: "stale timestamp",
			mutate: func(h *harness) *http.Request {
				r := h.request("m", freego.CompatibilityDate, body)
				for k, v := range Sign(h.priv, "m", fixedNow.Add(-time.Hour), []byte(body)) {
					r.Header[k] = v
				}
				return r
			},
			status: http.StatusBadRequest, wantErr: ErrInvalidTimestamp,
		},
		{
			name: "future timestamp",
			mutate: func(h *harness) *http.Request {
				r := h.request("m", freego.CompatibilityDate, body)
				for k, v := range Sign(h.priv, "m", fixedNow.Add(time.Hour), []byte(body)) {
					r.Header[k] = v
				}
				return r
			},
			status: http.StatusBadRequest, wantErr: ErrInvalidTimestamp,
		},
		{
			name: "compatibility date mismatch",
			mutate: func(h *harness) *http.Request {
				return h.request("m", "2025-03-01", body)
			},
			status: http.StatusBadRequest, wantErr: ErrCompatibilityDateMismatch,
		},
		{
			name: "invalid JSON",
			mutate: func(h *harness) *http.Request {
				return h.request("m", freego.CompatibilityDate, `{"type":`)
			},
			status: http.StatusBadRequest,
		},
		{
			name: "missing event type",
			mutate: func(h *harness) *http.Request {
				return h.request("m", freego.CompatibilityDate, `{"data":{}}`)
			},
			status: http.StatusBadRequest,
		},
		{
			name: "oversized body",
			mutate: func(h *harness) *http.Request {
				return h.request("m", freego.CompatibilityDate, envelopeJSON(EventPing, `{"pad":"`+strings.Repeat("x", int(DefaultMaxBodyBytes))+`"}`))
			},
			status: http.StatusRequestEntityTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, nil)
			w := h.serve(tt.mutate(h))

			if w.Code != tt.status {
				t.Fatalf("status = %d, want %d: %s", w.Code, tt.status, w.Body)
			}
			if len(h.rec.pings) != 0 {
				t.Error("callback ran for a rejected delivery")
			}
			if len(h.rec.errs) != 1 || h.rec.errEvents[0] != nil {
				t.Fatalf("OnError calls = %v", h.rec.errs)
			}
			var rej *RejectionError
			if !errors.As(h.rec.errs[0], &rej) || rej.Status != tt.status {
				t.Errorf("OnError err = %v", h.rec.errs[0])
			}
			if tt.wantErr != nil && !errors.Is(h.rec.errs[0], tt.wantErr) {
				t.Errorf("OnError err = %v, want %v", h.rec.errs[0], tt.wantErr)
			}
			// Library headers are sent on every response.
			if w.Header().Get(HeaderSetCompatibilityDate) != freego.CompatibilityDate || w.Header().Get(HeaderClientLibrary) == "" {
				t.Errorf("headers = %v", w.Header())
			}
			if tt.status == http.StatusMethodNotAllowed && w.Header().Get("Allow") != http.MethodPost {
				t.Errorf("Allow = %q", w.Header().Get("Allow"))
			}
		})
	}
}

func httptestBody(s string) *readCloser { return &readCloser{strings.NewReader(s)} }

type readCloser struct{ *strings.Reader }

func (readCloser) Close() error { return nil }

func TestHandlerReplay(t *testing.T) {
	h := newHarness(t, nil)
	body := envelopeJSON(EventPing, `{"manual":true}`)

	for i := 0; i < 3; i++ {
		if w := h.serve(h.request("msg_same", freego.CompatibilityDate, body)); w.Code != http.StatusNoContent {
			t.Fatalf("delivery %d: status %d", i, w.Code)
		}
	}
	if len(h.rec.pings) != 1 {
		t.Errorf("callback ran %d times, want 1", len(h.rec.pings))
	}
}

func TestHandlerRedeliveryAfterRejection(t *testing.T) {
	h := newHarness(t, nil)
	body := envelopeJSON(EventPing, `{"manual":true}`)

	// FreeStuff sends the old format, the handler asks for its date...
	w := h.serve(h.request("msg_1", "2025-03-01", body))
	if w.Code != http.StatusBadRequest || w.Header().Get(HeaderSetCompatibilityDate) != freego.CompatibilityDate {
		t.Fatalf("first delivery: status %d, headers %v", w.Code, w.Header())
	}
	// ...and redelivers the same message in that format, which must be
	// accepted although its id was already received.
	if w := h.serve(h.request("msg_1", freego.CompatibilityDate, body)); w.Code != http.StatusNoContent {
		t.Fatalf("redelivery: status %d", w.Code)
	}
	if len(h.rec.pings) != 1 {
		t.Errorf("callback ran %d times, want 1", len(h.rec.pings))
	}
}

func TestHandlerCompatibilityDateOptions(t *testing.T) {
	body := envelopeJSON(EventProductUpdated, `{"id":1,"description":"old style","until":0}`)

	t.Run("accept any", func(t *testing.T) {
		h := newHarness(t, func(c *Config) { c.AcceptAnyCompatibilityDate = true })
		w := h.serve(h.request("m", "2025-03-01", body))
		if w.Code != http.StatusNoContent {
			t.Fatalf("status %d: %s", w.Code, w.Body)
		}
		if w.Header().Get(HeaderSetCompatibilityDate) != "" {
			t.Error("X-Set-Compatibility-Date sent although every date is accepted")
		}
		if len(h.rec.products) != 1 || h.rec.products[0].Description.Get("en") != "old style" {
			t.Errorf("products = %+v", h.rec.products)
		}
	})

	t.Run("custom date", func(t *testing.T) {
		h := newHarness(t, func(c *Config) { c.CompatibilityDate = "2026-04-01" })
		if w := h.serve(h.request("m1", "2026-04-01", body)); w.Code != http.StatusNoContent || w.Header().Get(HeaderSetCompatibilityDate) != "2026-04-01" {
			t.Errorf("status %d, headers %v", w.Code, w.Header())
		}
		if w := h.serve(h.request("m2", freego.CompatibilityDate, body)); w.Code != http.StatusBadRequest {
			t.Errorf("status %d, want 400", w.Code)
		}
	})

	t.Run("header absent", func(t *testing.T) {
		h := newHarness(t, nil)
		if w := h.serve(h.request("m", "", body)); w.Code != http.StatusNoContent {
			t.Errorf("status %d", w.Code)
		}
	})
}

func TestHandlerCallbackFailures(t *testing.T) {
	body := envelopeJSON(EventPing, `{"manual":true}`)

	t.Run("error", func(t *testing.T) {
		boom := errors.New("boom")
		h := newHarness(t, func(c *Config) {
			c.OnPing = func(context.Context, *Event, *Ping) error { return boom }
		})
		if w := h.serve(h.request("m", freego.CompatibilityDate, body)); w.Code != http.StatusNoContent {
			t.Fatalf("status %d", w.Code)
		}
		if len(h.rec.errs) != 1 || !errors.Is(h.rec.errs[0], boom) || h.rec.errEvents[0] == nil || h.rec.errEvents[0].ID != "m" {
			t.Errorf("OnError = %v %v", h.rec.errs, h.rec.errEvents)
		}
	})

	t.Run("panic", func(t *testing.T) {
		h := newHarness(t, func(c *Config) {
			c.OnPing = func(context.Context, *Event, *Ping) error { panic("kaboom") }
		})
		if w := h.serve(h.request("m", freego.CompatibilityDate, body)); w.Code != http.StatusNoContent {
			t.Fatalf("status %d", w.Code)
		}
		var pe *PanicError
		if len(h.rec.errs) != 1 || !errors.As(h.rec.errs[0], &pe) || pe.Value != "kaboom" || len(pe.Stack) == 0 {
			t.Errorf("OnError = %v", h.rec.errs)
		}
		// The handler keeps working afterwards.
		if w := h.serve(h.request("m2", freego.CompatibilityDate, body)); w.Code != http.StatusNoContent {
			t.Errorf("status after panic %d", w.Code)
		}
	})

	t.Run("undecodable payload", func(t *testing.T) {
		h := newHarness(t, nil)
		w := h.serve(h.request("m", freego.CompatibilityDate, envelopeJSON(EventProductUpdated, `{"id":"not a number"}`)))
		if w.Code != http.StatusNoContent {
			t.Fatalf("status %d", w.Code)
		}
		if len(h.rec.products) != 0 || len(h.rec.errs) != 1 || !strings.Contains(h.rec.errs[0].Error(), "decoding fsb:event:product_updated") {
			t.Errorf("errs = %v", h.rec.errs)
		}
	})

	t.Run("nil callbacks", func(t *testing.T) {
		pub, priv := newKeyPair(t)
		h := &harness{t: t, priv: priv, handler: NewHandler(newTestVerifier(t, pub), Config{})}
		for i, typ := range []string{EventPing, EventAnnouncementCreated, EventProductUpdated, "fsb:event:other"} {
			w := h.serve(h.request("m"+string(rune('a'+i)), freego.CompatibilityDate, envelopeJSON(typ, `{}`)))
			if w.Code != http.StatusNoContent {
				t.Errorf("%s: status %d", typ, w.Code)
			}
		}
	})
}

func TestNewHandlerNilVerifier(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected a panic")
		}
	}()
	NewHandler(nil, Config{})
}

func TestServe(t *testing.T) {
	h := newHarness(t, nil)
	h.handler.verifier.now = time.Now
	body := envelopeJSON(EventPing, `{"manual":true}`)

	start := func() (string, context.CancelFunc, chan error) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- Serve(ctx, l, "/hooks/freestuff", h.handler) }()
		return "http://" + l.Addr().String(), cancel, done
	}

	// Two servers in one process: the old implementation registered on
	// http.DefaultServeMux and panicked here.
	url1, cancel1, done1 := start()
	url2, cancel2, done2 := start()

	for i, base := range []string{url1, url2} {
		req, _ := http.NewRequest(http.MethodPost, base+"/hooks/freestuff", bytes.NewReader([]byte(body)))
		for k, v := range Sign(h.priv, "live_"+string(rune('a'+i)), time.Now(), []byte(body)) {
			req.Header[k] = v
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("server %d: status %d", i, resp.StatusCode)
		}

		resp, err = http.Get(base + "/other")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("server %d: other path status %d", i, resp.StatusCode)
		}
	}

	cancel1()
	cancel2()
	for _, done := range []chan error{done1, done2} {
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Serve returned %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Serve did not stop after cancellation")
		}
	}
	if len(h.rec.pings) != 2 {
		t.Errorf("pings = %d", len(h.rec.pings))
	}
}

func TestListenAndServeBadAddress(t *testing.T) {
	if err := ListenAndServe(context.Background(), "256.0.0.1:99999", "/", http.NotFoundHandler()); err == nil {
		t.Error("expected an error")
	}
}

// A resend arrives with the same id but a fresh timestamp, long after the
// tolerance window of the first attempt.
func TestHandlerReplayWindowCoversResends(t *testing.T) {
	pub, priv := newKeyPair(t)
	now := fixedNow
	v, err := NewVerifierFromKey(pub, WithClock(func() time.Time { return now }))
	if err != nil {
		t.Fatal(err)
	}
	cache := NewMemoryReplayCache()
	cache.now = func() time.Time { return now }
	rec := &recorder{}
	cfg := rec.config()
	cfg.ReplayCache = cache
	h := &harness{t: t, priv: priv, handler: NewHandler(v, cfg), rec: rec}

	body := envelopeJSON(EventPing, `{"manual":false}`)
	deliver := func() int {
		r := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
		for k, vals := range Sign(priv, "msg_retry", now, []byte(body)) {
			r.Header[k] = vals
		}
		return h.serve(r).Code
	}

	if code := deliver(); code != http.StatusNoContent {
		t.Fatalf("first delivery: %d", code)
	}
	now = now.Add(5 * time.Hour)
	if code := deliver(); code != http.StatusNoContent {
		t.Fatalf("resend: %d", code)
	}
	if len(rec.pings) != 1 {
		t.Errorf("callback ran %d times, want 1", len(rec.pings))
	}

	now = now.Add(DefaultReplayWindow)
	if deliver(); len(rec.pings) != 2 {
		t.Errorf("delivery after the replay window was not processed")
	}
}

func TestHandlerReplayWindowMinimum(t *testing.T) {
	pub, _ := newKeyPair(t)
	v, _ := NewVerifierFromKey(pub, WithTolerance(time.Hour))
	h := NewHandler(v, Config{ReplayWindow: time.Minute})
	if h.cfg.ReplayWindow != 2*time.Hour+time.Minute {
		t.Errorf("ReplayWindow = %v", h.cfg.ReplayWindow)
	}
}

func TestHandlerEnvelopeTimestamp(t *testing.T) {
	for name, ts := range map[string]string{
		"no zone":      `"2026-10-01T11:59:58"`,
		"object":       `{}`,
		"missing":      ``,
		"milliseconds": `1790856000000`,
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t, nil)
			var got time.Time
			h.handler.cfg.OnPing = func(_ context.Context, e *Event, _ *Ping) error {
				got = e.Timestamp
				return nil
			}
			body := `{"type":"fsb:event:ping","data":{"manual":true}`
			if ts != "" {
				body += `,"timestamp":` + ts
			}
			body += `}`

			if w := h.serve(h.request("m", freego.CompatibilityDate, body)); w.Code != http.StatusNoContent {
				t.Fatalf("status %d: %s", w.Code, w.Body)
			}
			want := fixedNow // falls back to the delivery time
			if name == "milliseconds" {
				want = time.UnixMilli(1790856000000)
			}
			if !got.Equal(want) {
				t.Errorf("Timestamp = %v, want %v", got, want)
			}
		})
	}
}

type failingCache struct{}

func (failingCache) Seen(string, time.Duration) (bool, error) { return false, errors.New("store down") }

func TestHandlerReplayCacheError(t *testing.T) {
	h := newHarness(t, func(c *Config) { c.ReplayCache = failingCache{} })
	w := h.serve(h.request("m", freego.CompatibilityDate, envelopeJSON(EventPing, `{}`)))
	if w.Code != http.StatusInternalServerError || len(h.rec.pings) != 0 {
		t.Errorf("status %d, pings %d", w.Code, len(h.rec.pings))
	}
	if len(h.rec.errs) != 1 || !strings.Contains(h.rec.errs[0].Error(), "store down") {
		t.Errorf("errs = %v", h.rec.errs)
	}
}

func TestNewHandlerValidation(t *testing.T) {
	pub, _ := newKeyPair(t)
	v, _ := NewVerifierFromKey(pub)
	for name, f := range map[string]func(){
		"zero verifier":     func() { NewHandler(&Verifier{}, Config{}) },
		"invalid date":      func() { NewHandler(v, Config{CompatibilityDate: "2026-6-8"}) },
		"not a date at all": func() { NewHandler(v, Config{CompatibilityDate: "latest"}) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("expected a panic")
				}
			}()
			f()
		})
	}
}

func TestZeroVerifier(t *testing.T) {
	var v Verifier
	if err := v.Verify("id", "1", "v1a,AAAA", []byte("{}")); err == nil {
		t.Error("expected an error")
	}
}

func TestServeForcesShutdown(t *testing.T) {
	old := shutdownTimeout
	shutdownTimeout = 100 * time.Millisecond
	defer func() { shutdownTimeout = old }()

	started := make(chan struct{})
	cancelled := make(chan struct{})
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(cancelled)
	})

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, l, "/", slow) }()

	go func() {
		resp, err := http.Get("http://" + l.Addr().String() + "/")
		if err == nil {
			resp.Body.Close()
		}
	}()
	<-started
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Serve returned %v, want a deadline error", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return")
	}
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("the in-flight request was not cancelled")
	}
}

func TestServeInvalidPath(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if err := Serve(context.Background(), l, "webhook", http.NotFoundHandler()); err == nil {
		t.Error("expected an error")
	}
}
