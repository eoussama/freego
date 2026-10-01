package webhook

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/eoussama/freego"
)

// Event types delivered by FreeStuff.
const (
	// EventPing is sent when the ping button on the dashboard is pressed,
	// and periodically to check that the webhook works.
	EventPing = "fsb:event:ping"
	// EventAnnouncementCreated is sent when an announcement is published.
	EventAnnouncementCreated = "fsb:event:announcement_created"
	// EventProductUpdated is sent when a published product changes.
	EventProductUpdated = "fsb:event:product_updated"
)

// DefaultMaxBodyBytes is the default limit on the size of a delivery.
const DefaultMaxBodyBytes int64 = 1 << 20

// Event is a verified webhook message.
type Event struct {
	// ID is the message id. Retries of a delivery share the same id.
	ID string
	// Type is the event type, such as [EventAnnouncementCreated].
	Type string
	// Timestamp is when the event happened, taken from the payload. When
	// the payload's timestamp cannot be parsed, it is the delivery time.
	Timestamp time.Time
	// CompatibilityDate is the compatibility date of Data, or "" when the
	// request did not specify one.
	CompatibilityDate string
	// Data is the raw event payload.
	Data json.RawMessage
}

// Ping is the payload of an [EventPing] event.
type Ping struct {
	// Manual is true when the ping was requested from the dashboard.
	Manual bool `json:"manual"`
}

// Config configures a [Handler]. Callbacks left nil are skipped; the
// delivery is still acknowledged.
type Config struct {
	// OnPing is called for ping events.
	OnPing func(ctx context.Context, e *Event, ping *Ping) error
	// OnAnnouncementCreated is called when an announcement is published,
	// with its products resolved.
	OnAnnouncementCreated func(ctx context.Context, e *Event, a *freego.ResolvedAnnouncement) error
	// OnProductUpdated is called with the full, updated product.
	OnProductUpdated func(ctx context.Context, e *Event, p *freego.Product) error
	// OnUnknown is called for event types this library does not know.
	OnUnknown func(ctx context.Context, e *Event) error

	// OnError is called with every error: rejected deliveries (e is nil and
	// err wraps one of the verification errors or a [*RejectionError]), and
	// failed or panicking callbacks (e is the event). Use it for logging.
	OnError func(ctx context.Context, e *Event, err error)

	// MaxBodyBytes limits the size of a delivery. It defaults to
	// [DefaultMaxBodyBytes].
	MaxBodyBytes int64

	// CompatibilityDate is the compatibility date the handler accepts. It
	// defaults to [freego.CompatibilityDate], the date the Go types are
	// built for.
	//
	// Deliveries carrying another date are refused with 400 and the
	// response asks FreeStuff, through the X-Set-Compatibility-Date header,
	// to switch your app to this date. FreeStuff then updates the setting
	// on your dashboard and redelivers the message in the right format, so
	// nothing is lost. This is the behaviour FreeStuff recommends for
	// libraries. Deliveries without an X-Compatibility-Date header are
	// accepted.
	CompatibilityDate string
	// AcceptAnyCompatibilityDate disables the above: every compatibility
	// date is accepted and the dashboard setting is left alone. Payloads of
	// older dates still decode where possible.
	AcceptAnyCompatibilityDate bool

	// ReplayCache detects replayed and retried deliveries. It defaults to a
	// [MemoryReplayCache].
	ReplayCache ReplayCache
	// ReplayWindow is how long message ids are remembered. A delivery whose
	// id was already received within the window is acknowledged without
	// running the callbacks again. FreeStuff retries failed deliveries with
	// the same id over several hours, so keep it long. It defaults to
	// [DefaultReplayWindow] and is never shorter than twice the verifier's
	// tolerance plus a minute, the time a signed message stays valid.
	ReplayWindow time.Duration
}

// DefaultReplayWindow is the default [Config.ReplayWindow].
const DefaultReplayWindow = 24 * time.Hour

// RejectionError describes why a delivery was refused. It is passed to
// [Config.OnError].
type RejectionError struct {
	// Status is the HTTP status the delivery was answered with.
	Status int
	// Err is the underlying cause.
	Err error
}

func (e *RejectionError) Error() string {
	return fmt.Sprintf("webhook: delivery rejected with %d: %v", e.Status, e.Err)
}

func (e *RejectionError) Unwrap() error { return e.Err }

// PanicError is passed to [Config.OnError] when a callback panics.
type PanicError struct {
	// Value is the value passed to panic.
	Value any
	// Stack is the stack trace of the panicking goroutine.
	Stack []byte
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("webhook: callback panicked: %v", e.Value)
}

// ErrCompatibilityDateMismatch is wrapped by the [RejectionError] of a
// delivery refused because of its compatibility date.
var ErrCompatibilityDateMismatch = errors.New("webhook: compatibility date mismatch")

// Handler is an [http.Handler] receiving FreeStuff webhook deliveries.
// Create it with [NewHandler].
type Handler struct {
	verifier *Verifier
	cfg      Config
}

// NewHandler returns a handler that verifies deliveries with v and passes
// them to the callbacks in cfg. It panics if v was not created with
// [NewVerifier] or [NewVerifierFromKey], or if cfg.CompatibilityDate is not a
// YYYY-MM-DD date: answering FreeStuff with an invalid date would make it
// drop every delivery.
func NewHandler(v *Verifier, cfg Config) *Handler {
	if v == nil || len(v.key) != ed25519.PublicKeySize {
		panic("webhook: NewHandler needs a Verifier created with NewVerifier")
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = DefaultMaxBodyBytes
	}
	if cfg.CompatibilityDate == "" {
		cfg.CompatibilityDate = freego.CompatibilityDate
	}
	if _, err := time.Parse("2006-01-02", cfg.CompatibilityDate); err != nil {
		panic(fmt.Sprintf("webhook: invalid compatibility date %q, expected YYYY-MM-DD", cfg.CompatibilityDate))
	}
	if cfg.ReplayCache == nil {
		cfg.ReplayCache = NewMemoryReplayCache()
	}
	if cfg.ReplayWindow <= 0 {
		cfg.ReplayWindow = DefaultReplayWindow
	}
	if minWindow := 2*v.Tolerance() + time.Minute; cfg.ReplayWindow < minWindow {
		cfg.ReplayWindow = minWindow
	}
	return &Handler{verifier: v, cfg: cfg}
}

// envelope is the Standard Webhooks payload.
type envelope struct {
	Type      string          `json:"type"`
	Timestamp json.RawMessage `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// ServeHTTP implements [http.Handler].
//
// Responses:
//   - 204 when the delivery was received, including duplicates and
//     deliveries whose payload could not be decoded or whose callback failed
//     (reported to [Config.OnError]);
//   - 400 for missing headers, bad timestamps, an envelope that is not valid
//     JSON or has no type, and compatibility date mismatches;
//   - 401 for invalid signatures, 405 for methods other than POST, 413 for
//     oversized bodies;
//   - 500 when the [ReplayCache] fails, so that FreeStuff retries.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	w.Header().Set(HeaderClientLibrary, freego.ClientLibrary)
	if !h.cfg.AcceptAnyCompatibilityDate {
		w.Header().Set(HeaderSetCompatibilityDate, h.cfg.CompatibilityDate)
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		h.reject(ctx, w, http.StatusMethodNotAllowed, fmt.Errorf("method %s not allowed", r.Method))
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, h.cfg.MaxBodyBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			h.reject(ctx, w, http.StatusRequestEntityTooLarge, fmt.Errorf("body larger than %d bytes", h.cfg.MaxBodyBytes))
			return
		}
		h.reject(ctx, w, http.StatusBadRequest, fmt.Errorf("reading body: %w", err))
		return
	}

	if err := h.verifier.VerifyRequest(r, body); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrInvalidSignature) || errors.Is(err, ErrUnsupportedSignature) {
			status = http.StatusUnauthorized
		}
		h.reject(ctx, w, status, err)
		return
	}

	compat := r.Header.Get(HeaderCompatibilityDate)
	if !h.cfg.AcceptAnyCompatibilityDate && compat != "" && compat != h.cfg.CompatibilityDate {
		h.reject(ctx, w, http.StatusBadRequest, fmt.Errorf("%w: got %s, want %s", ErrCompatibilityDateMismatch, compat, h.cfg.CompatibilityDate))
		return
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err != nil || env.Type == "" {
		if err == nil {
			err = errors.New("missing event type")
		}
		h.reject(ctx, w, http.StatusBadRequest, fmt.Errorf("malformed payload: %w", err))
		return
	}

	// Only record the id once every reason to refuse the delivery has been
	// ruled out, so that redeliveries of refused messages are accepted.
	id := r.Header.Get(HeaderID)
	seen, err := h.cfg.ReplayCache.Seen(id, h.cfg.ReplayWindow)
	if err != nil {
		h.reject(ctx, w, http.StatusInternalServerError, fmt.Errorf("replay cache: %w", err))
		return
	}
	if seen {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	event := &Event{
		ID:                id,
		Type:              env.Type,
		Timestamp:         eventTime(env.Timestamp, r.Header.Get(HeaderTimestamp)),
		CompatibilityDate: compat,
		Data:              env.Data,
	}
	if err := h.dispatch(ctx, event); err != nil && h.cfg.OnError != nil {
		h.cfg.OnError(ctx, event, err)
	}

	// FreeStuff asks for a 2xx once a message is received, even when
	// processing it failed.
	w.WriteHeader(http.StatusNoContent)
}

// dispatch decodes the event payload and runs the matching callback,
// recovering from panics.
func (h *Handler) dispatch(ctx context.Context, e *Event) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = &PanicError{Value: v, Stack: debug.Stack()}
		}
	}()

	switch e.Type {
	case EventPing:
		if h.cfg.OnPing == nil {
			return nil
		}
		var ping Ping
		if err := decodeData(e, &ping); err != nil {
			return err
		}
		return h.cfg.OnPing(ctx, e, &ping)

	case EventAnnouncementCreated:
		if h.cfg.OnAnnouncementCreated == nil {
			return nil
		}
		var a freego.ResolvedAnnouncement
		if err := decodeData(e, &a); err != nil {
			return err
		}
		return h.cfg.OnAnnouncementCreated(ctx, e, &a)

	case EventProductUpdated:
		if h.cfg.OnProductUpdated == nil {
			return nil
		}
		var p freego.Product
		if err := decodeData(e, &p); err != nil {
			return err
		}
		return h.cfg.OnProductUpdated(ctx, e, &p)

	default:
		if h.cfg.OnUnknown == nil {
			return nil
		}
		return h.cfg.OnUnknown(ctx, e)
	}
}

// eventTime parses the envelope's timestamp. It is informational only, so a
// format this library does not know falls back to the delivery time from the
// already verified Webhook-Timestamp header instead of failing the delivery.
func eventTime(raw json.RawMessage, header string) time.Time {
	var t freego.Time
	if len(raw) > 0 && json.Unmarshal(raw, &t) == nil && !t.IsZero() {
		return t.Time
	}
	sent, _ := ParseTimestamp(header)
	return sent
}

func decodeData(e *Event, v any) error {
	if err := json.Unmarshal(e.Data, v); err != nil {
		return fmt.Errorf("webhook: decoding %s payload: %w", e.Type, err)
	}
	return nil
}

func (h *Handler) reject(ctx context.Context, w http.ResponseWriter, status int, err error) {
	if h.cfg.OnError != nil {
		h.cfg.OnError(ctx, nil, &RejectionError{Status: status, Err: err})
	}
	http.Error(w, err.Error(), status)
}
