package webhook

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"strings"
	"testing"
	"time"
)

// fixedNow is the clock used by the tests.
var fixedNow = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func newKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv
}

// dashboardKey encodes pub like the FreeStuff dashboard: base64 DER SPKI.
func dashboardKey(t *testing.T, pub ed25519.PublicKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(der)
}

func newTestVerifier(t *testing.T, pub ed25519.PublicKey) *Verifier {
	t.Helper()
	v, err := NewVerifier(dashboardKey(t, pub), WithClock(func() time.Time { return fixedNow }))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestParsePublicKey(t *testing.T) {
	pub, _ := newKeyPair(t)
	der, _ := x509.MarshalPKIXPublicKey(pub)
	b64 := base64.StdEncoding.EncodeToString(der)

	valid := map[string]string{
		"dashboard (base64 DER)": b64,
		"unpadded":               strings.TrimRight(b64, "="),
		"url-safe":               base64.RawURLEncoding.EncodeToString(der),
		"surrounding whitespace": "  " + b64 + "\n",
		"PEM":                    string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})),
		"raw 32 bytes":           base64.StdEncoding.EncodeToString(pub),
	}
	for name, in := range valid {
		t.Run(name, func(t *testing.T) {
			got, err := ParsePublicKey(in)
			if err != nil {
				t.Fatal(err)
			}
			if !got.Equal(pub) {
				t.Error("parsed a different key")
			}
		})
	}

	// The prefix FreeStuff keys start with must parse (it is the standard
	// Ed25519 SPKI header).
	if !strings.HasPrefix(b64, "MCowBQYDK2VwAyEA") {
		t.Errorf("unexpected SPKI prefix in %s", b64)
	}

	// Any 32 bytes form a raw key, so garbage must have another length.
	garbage := base64.StdEncoding.EncodeToString([]byte("this is not a public key at all!!"))
	for name, in := range map[string]string{
		"empty":       "",
		"not base64":  "not base64!!",
		"garbage DER": garbage,
		"bad PEM":     "-----BEGIN PUBLIC KEY-----\nnope\n-----END PUBLIC KEY-----",
	} {
		t.Run("invalid "+name, func(t *testing.T) {
			if _, err := ParsePublicKey(in); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestNewVerifierOptions(t *testing.T) {
	pub, _ := newKeyPair(t)
	if _, err := NewVerifierFromKey(pub[:10]); err == nil {
		t.Error("expected an error for a short key")
	}
	if _, err := NewVerifierFromKey(pub, WithTolerance(0)); err == nil {
		t.Error("expected an error for a zero tolerance")
	}
	v, err := NewVerifierFromKey(pub, WithTolerance(time.Minute))
	if err != nil || v.Tolerance() != time.Minute {
		t.Errorf("Tolerance = %v, %v", v.Tolerance(), err)
	}
}

func TestTimestamps(t *testing.T) {
	// 2026-10-01T12:00:00Z is 638 days and 12 hours after 2025-01-01.
	const want = "55166400"
	if got := FormatTimestamp(fixedNow); got != want {
		t.Errorf("FormatTimestamp = %s, want %s", got, want)
	}
	got, err := ParseTimestamp(want)
	if err != nil || !got.Equal(fixedNow) {
		t.Errorf("ParseTimestamp = %v, %v", got, err)
	}
	// The docs' formula: unix ms = value*1000 + 1735689600000.
	if got.UnixMilli() != 55166400*1000+1735689600000 {
		t.Errorf("custom epoch mismatch: %d", got.UnixMilli())
	}
	for _, bad := range []string{"", "-1", "12.5", "abc"} {
		if _, err := ParseTimestamp(bad); !errors.Is(err, ErrInvalidTimestamp) {
			t.Errorf("ParseTimestamp(%q) err = %v", bad, err)
		}
	}
}

func TestVerify(t *testing.T) {
	pub, priv := newKeyPair(t)
	_, otherPriv := newKeyPair(t)
	v := newTestVerifier(t, pub)
	body := []byte(`{"type":"fsb:event:ping","timestamp":"2026-10-01T12:00:00Z","data":{"manual":true}}`)

	signed := Sign(priv, "msg_1", fixedNow, body)
	id, ts, sig := signed.Get(HeaderID), signed.Get(HeaderTimestamp), signed.Get(HeaderSignature)
	if !strings.HasPrefix(sig, "v1a,") {
		t.Fatalf("signature header %q", sig)
	}

	if err := v.Verify(id, ts, sig, body); err != nil {
		t.Fatalf("valid message rejected: %v", err)
	}

	other := Sign(otherPriv, "msg_1", fixedNow, body).Get(HeaderSignature)
	sigB64 := strings.TrimPrefix(sig, "v1a,")

	tests := []struct {
		name          string
		id, ts, sig   string
		body          []byte
		want          error
		wantSucceeded bool
	}{
		{name: "missing id", ts: ts, sig: sig, body: body, want: ErrMissingHeaders},
		{name: "missing timestamp", id: id, sig: sig, body: body, want: ErrMissingHeaders},
		{name: "missing signature", id: id, ts: ts, body: body, want: ErrMissingHeaders},
		{name: "tampered body", id: id, ts: ts, sig: sig, body: []byte(strings.Replace(string(body), "true", "false", 1)), want: ErrInvalidSignature},
		{name: "tampered id", id: "msg_2", ts: ts, sig: sig, body: body, want: ErrInvalidSignature},
		{name: "wrong key", id: id, ts: ts, sig: other, body: body, want: ErrInvalidSignature},
		{name: "unsupported version", id: id, ts: ts, sig: "v1," + sigB64, body: body, want: ErrUnsupportedSignature},
		{name: "no comma", id: id, ts: ts, sig: "v1a" + sigB64, body: body, want: ErrUnsupportedSignature},
		{name: "bad base64", id: id, ts: ts, sig: "v1a,!!!", body: body, want: ErrInvalidSignature},
		{name: "truncated signature", id: id, ts: ts, sig: "v1a," + sigB64[:20], body: body, want: ErrInvalidSignature},
		{name: "several signatures, one valid", id: id, ts: ts, sig: other + " v1," + sigB64 + " " + sig, body: body, wantSucceeded: true},
		{name: "unpadded signature", id: id, ts: ts, sig: strings.TrimRight(sig, "="), body: body, wantSucceeded: true},
		{name: "malformed timestamp", id: id, ts: "yesterday", sig: sig, body: body, want: ErrInvalidTimestamp},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Verify(tt.id, tt.ts, tt.sig, tt.body)
			if tt.wantSucceeded {
				if err != nil {
					t.Errorf("err = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestVerifyTimestampTolerance(t *testing.T) {
	pub, priv := newKeyPair(t)
	v := newTestVerifier(t, pub)
	body := []byte(`{}`)

	for _, tt := range []struct {
		name  string
		at    time.Time
		valid bool
	}{
		{"now", fixedNow, true},
		{"4 minutes old", fixedNow.Add(-4 * time.Minute), true},
		{"6 minutes old", fixedNow.Add(-6 * time.Minute), false},
		{"4 minutes ahead", fixedNow.Add(4 * time.Minute), true},
		{"6 minutes ahead", fixedNow.Add(6 * time.Minute), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := Sign(priv, "msg", tt.at, body)
			err := v.Verify(h.Get(HeaderID), h.Get(HeaderTimestamp), h.Get(HeaderSignature), body)
			if tt.valid && err != nil {
				t.Errorf("err = %v", err)
			}
			if !tt.valid && !errors.Is(err, ErrInvalidTimestamp) {
				t.Errorf("err = %v, want ErrInvalidTimestamp", err)
			}
		})
	}
}
