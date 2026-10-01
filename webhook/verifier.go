package webhook

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Request and response headers used by FreeStuff webhooks.
const (
	HeaderID                   = "Webhook-Id"
	HeaderTimestamp            = "Webhook-Timestamp"
	HeaderSignature            = "Webhook-Signature"
	HeaderCompatibilityDate    = "X-Compatibility-Date"
	HeaderSetCompatibilityDate = "X-Set-Compatibility-Date"
	HeaderClientLibrary        = "X-Client-Library"
)

// DefaultTolerance is how far a message's timestamp may be from the current
// time before the message is rejected.
const DefaultTolerance = 5 * time.Minute

// signatureVersion is the Webhook-Signature scheme used by FreeStuff:
// Ed25519 over "id.timestamp.body".
const signatureVersion = "v1a"

// timestampEpoch is FreeStuff's custom epoch for the Webhook-Timestamp
// header, which counts seconds since 2025-01-01T00:00:00Z.
var timestampEpoch = time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

// Verification errors. Errors returned by [Verifier.Verify] wrap one of them.
var (
	// ErrMissingHeaders means a Webhook-Id, Webhook-Timestamp or
	// Webhook-Signature header is missing.
	ErrMissingHeaders = errors.New("webhook: missing webhook headers")
	// ErrInvalidTimestamp means the timestamp is malformed, too old or too
	// far in the future.
	ErrInvalidTimestamp = errors.New("webhook: invalid timestamp")
	// ErrUnsupportedSignature means no signature uses a supported scheme.
	ErrUnsupportedSignature = errors.New("webhook: unsupported signature scheme")
	// ErrInvalidSignature means no signature matches the message.
	ErrInvalidSignature = errors.New("webhook: invalid signature")
)

// Verifier checks the signature and timestamp of webhook messages. It is safe
// for concurrent use.
type Verifier struct {
	key       ed25519.PublicKey
	tolerance time.Duration
	now       func() time.Time
}

// VerifierOption configures a [Verifier].
type VerifierOption func(*Verifier)

// WithTolerance sets how far a message's timestamp may be from the current
// time, in either direction. It defaults to [DefaultTolerance].
func WithTolerance(d time.Duration) VerifierOption {
	return func(v *Verifier) { v.tolerance = d }
}

// WithClock replaces the clock used to check timestamps. Useful in tests.
func WithClock(now func() time.Time) VerifierOption {
	return func(v *Verifier) { v.now = now }
}

// NewVerifier returns a verifier for messages signed by the app whose public
// key is given. The key is accepted in the format shown on the FreeStuff
// dashboard (base64 DER), as PEM, or as a base64 raw 32-byte Ed25519 key.
func NewVerifier(publicKey string, opts ...VerifierOption) (*Verifier, error) {
	key, err := ParsePublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	return NewVerifierFromKey(key, opts...)
}

// NewVerifierFromKey is like [NewVerifier] for an already parsed key.
func NewVerifierFromKey(key ed25519.PublicKey, opts ...VerifierOption) (*Verifier, error) {
	if len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("webhook: invalid Ed25519 public key length %d", len(key))
	}
	v := &Verifier{key: key, tolerance: DefaultTolerance, now: time.Now}
	for _, opt := range opts {
		opt(v)
	}
	if v.tolerance <= 0 {
		return nil, errors.New("webhook: tolerance must be positive")
	}
	return v, nil
}

// ParsePublicKey parses an Ed25519 public key given as base64 DER
// (SubjectPublicKeyInfo, the format shown on the FreeStuff dashboard), as PEM,
// or as a base64 raw 32-byte key. Padded, unpadded and URL-safe base64 are
// accepted.
func ParsePublicKey(s string) (ed25519.PublicKey, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("webhook: missing public key")
	}

	var der []byte
	if strings.HasPrefix(s, "-----BEGIN") {
		block, _ := pem.Decode([]byte(s))
		if block == nil {
			return nil, errors.New("webhook: invalid PEM public key")
		}
		der = block.Bytes
	} else {
		var err error
		if der, err = decodeBase64(s); err != nil {
			return nil, errors.New("webhook: public key is not valid base64")
		}
		if len(der) == ed25519.PublicKeySize {
			return ed25519.PublicKey(der), nil
		}
	}

	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("webhook: invalid public key: %w", err)
	}
	key, ok := parsed.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("webhook: public key is %T, not Ed25519", parsed)
	}
	return key, nil
}

// decodeBase64 decodes standard or URL-safe base64, padded or not.
func decodeBase64(s string) ([]byte, error) {
	s = strings.TrimRight(s, "=")
	if strings.ContainsAny(s, "-_") {
		return base64.RawURLEncoding.DecodeString(s)
	}
	return base64.RawStdEncoding.DecodeString(s)
}

// Tolerance returns the timestamp tolerance of the verifier.
func (v *Verifier) Tolerance() time.Duration { return v.tolerance }

// Verify checks a message given the values of its Webhook-Id,
// Webhook-Timestamp and Webhook-Signature headers and its raw, unparsed body.
// It returns nil when the message is authentic and recent, and otherwise an
// error wrapping one of the verification errors such as
// [ErrInvalidSignature].
//
// Verify does not detect replayed messages; [Handler] does, or see
// [ReplayCache] to do it yourself.
func (v *Verifier) Verify(id, timestamp, signature string, body []byte) error {
	if len(v.key) != ed25519.PublicKeySize || v.tolerance <= 0 || v.now == nil {
		return errors.New("webhook: Verifier not initialized, create it with NewVerifier")
	}
	if id == "" || timestamp == "" || signature == "" {
		return ErrMissingHeaders
	}

	sent, err := ParseTimestamp(timestamp)
	if err != nil {
		return err
	}
	if age := v.now().Sub(sent); age > v.tolerance || age < -v.tolerance {
		return fmt.Errorf("%w: sent %s, outside the %s tolerance", ErrInvalidTimestamp, sent.Format(time.RFC3339), v.tolerance)
	}

	signed := make([]byte, 0, len(id)+len(timestamp)+len(body)+2)
	signed = append(signed, id...)
	signed = append(signed, '.')
	signed = append(signed, timestamp...)
	signed = append(signed, '.')
	signed = append(signed, body...)

	// The header may list several space separated "version,signature"
	// entries, e.g. during key rotation; one valid entry is enough.
	supported := false
	for _, entry := range strings.Fields(signature) {
		version, sig, ok := strings.Cut(entry, ",")
		if !ok || version != signatureVersion {
			continue
		}
		supported = true
		raw, err := decodeBase64(sig)
		if err != nil || len(raw) != ed25519.SignatureSize {
			continue
		}
		if ed25519.Verify(v.key, signed, raw) {
			return nil
		}
	}
	if !supported {
		return ErrUnsupportedSignature
	}
	return ErrInvalidSignature
}

// VerifyRequest is like [Verifier.Verify], reading the headers from r. body
// must be the raw request body.
func (v *Verifier) VerifyRequest(r *http.Request, body []byte) error {
	return v.Verify(r.Header.Get(HeaderID), r.Header.Get(HeaderTimestamp), r.Header.Get(HeaderSignature), body)
}

// ParseTimestamp parses a Webhook-Timestamp header value, which FreeStuff
// sends as seconds since its custom epoch of 2025-01-01T00:00:00Z.
func ParseTimestamp(s string) (time.Time, error) {
	secs, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || secs < 0 {
		return time.Time{}, fmt.Errorf("%w: %q is not a valid timestamp", ErrInvalidTimestamp, s)
	}
	return timestampEpoch.Add(time.Duration(secs) * time.Second), nil
}

// FormatTimestamp formats t as a Webhook-Timestamp header value.
func FormatTimestamp(t time.Time) string {
	return strconv.FormatInt(int64(t.Sub(timestampEpoch)/time.Second), 10)
}

// Sign signs a message the way FreeStuff does and returns the Webhook-Id,
// Webhook-Timestamp and Webhook-Signature headers to send with it. It is
// meant for testing your own handlers.
func Sign(key ed25519.PrivateKey, id string, at time.Time, body []byte) http.Header {
	ts := FormatTimestamp(at)
	signed := []byte(id + "." + ts + "." + string(body))
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(key, signed))

	h := http.Header{}
	h.Set(HeaderID, id)
	h.Set(HeaderTimestamp, ts)
	h.Set(HeaderSignature, signatureVersion+","+sig)
	return h
}
