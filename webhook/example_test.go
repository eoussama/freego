package webhook_test

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/eoussama/freego"
	"github.com/eoussama/freego/webhook"
)

func ExampleNewHandler() {
	verifier, err := webhook.NewVerifier(os.Getenv("FREEGO_FREESTUFF_PUBLIC_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	handler := webhook.NewHandler(verifier, webhook.Config{
		OnAnnouncementCreated: func(ctx context.Context, e *webhook.Event, a *freego.ResolvedAnnouncement) error {
			for _, p := range a.ResolvedProducts {
				if !p.ShouldIgnore() {
					fmt.Println("new free product:", p.Title)
				}
			}
			return nil
		},
		OnError: func(ctx context.Context, e *webhook.Event, err error) {
			log.Print(err)
		},
	})

	// Mount it on any router.
	mux := http.NewServeMux()
	mux.Handle("/webhooks/freestuff", handler)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func ExampleListenAndServe() {
	verifier, err := webhook.NewVerifier(os.Getenv("FREEGO_FREESTUFF_PUBLIC_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	handler := webhook.NewHandler(verifier, webhook.Config{
		OnPing: func(ctx context.Context, e *webhook.Event, p *webhook.Ping) error {
			log.Printf("ping (manual: %v)", p.Manual)
			return nil
		},
	})

	// Stop gracefully on Ctrl+C.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := webhook.ListenAndServe(ctx, ":8080", "/webhook", handler); err != nil {
		log.Fatal(err)
	}
}

// Sign lets you test your own handler with deliveries signed like
// FreeStuff's.
func ExampleSign() {
	pub, priv, _ := ed25519.GenerateKey(nil)
	der, _ := x509.MarshalPKIXPublicKey(pub)
	verifier, _ := webhook.NewVerifier(base64.StdEncoding.EncodeToString(der))

	handler := webhook.NewHandler(verifier, webhook.Config{
		OnPing: func(ctx context.Context, e *webhook.Event, p *webhook.Ping) error {
			fmt.Println("received", e.Type, "manual:", p.Manual)
			return nil
		},
	})

	body := `{"type":"fsb:event:ping","timestamp":"2026-10-01T12:00:00Z","data":{"manual":true}}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	for k, v := range webhook.Sign(priv, "msg_123", time.Now(), []byte(body)) {
		req.Header[k] = v
	}
	req.Header.Set(webhook.HeaderCompatibilityDate, freego.CompatibilityDate)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	fmt.Println(rec.Code)
	// Output:
	// received fsb:event:ping manual: true
	// 204
}

func ExampleVerifier_VerifyRequest() {
	verifier, err := webhook.NewVerifier(os.Getenv("FREEGO_FREESTUFF_PUBLIC_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	// With another framework, verify the raw body yourself.
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, webhook.DefaultMaxBodyBytes))
		if err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if err := verifier.VerifyRequest(r, body); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		// Decode and process body, remembering its Webhook-Id to skip
		// redeliveries.
		w.WriteHeader(http.StatusNoContent)
	})
}
