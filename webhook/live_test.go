//go:build live

package webhook

import (
	"os"
	"testing"
)

// TestLivePublicKey checks that the public key from the FreeStuff dashboard
// parses. Run it with:
//
//	FREEGO_FREESTUFF_PUBLIC_KEY=... go test -tags live -run Live ./webhook
func TestLivePublicKey(t *testing.T) {
	key := os.Getenv("FREEGO_FREESTUFF_PUBLIC_KEY")
	if key == "" {
		t.Skip("FREEGO_FREESTUFF_PUBLIC_KEY is not set")
	}
	if _, err := NewVerifier(key); err != nil {
		t.Fatal(err)
	}
}
