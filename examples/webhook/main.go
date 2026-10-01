// Command webhook receives FreeStuff webhook deliveries and logs them.
//
// Usage:
//
//	FREEGO_FREESTUFF_PUBLIC_KEY=... go run ./examples/webhook
//
// FREEGO_WEBHOOK_PORT (default 8080) and FREEGO_WEBHOOK_ROUTE (default
// /webhook) set where the server listens. Point your app's webhook URL on
// https://dashboard.freestuffbot.xyz/ at it, then press the ping button.
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/eoussama/freego"
	"github.com/eoussama/freego/webhook"
)

func main() {
	verifier, err := webhook.NewVerifier(os.Getenv("FREEGO_FREESTUFF_PUBLIC_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	handler := webhook.NewHandler(verifier, webhook.Config{
		// Set FREEGO_WEBHOOK_ACCEPT_ANY_DATE=true to keep the compatibility
		// date configured on the dashboard instead of asking FreeStuff to
		// switch to the library's. Needed behind proxies such as smee, which
		// hide the response headers from FreeStuff.
		AcceptAnyCompatibilityDate: os.Getenv("FREEGO_WEBHOOK_ACCEPT_ANY_DATE") == "true",

		OnPing: func(ctx context.Context, e *webhook.Event, p *webhook.Ping) error {
			log.Printf("ping %s (manual: %v)", e.ID, p.Manual)
			return nil
		},
		OnAnnouncementCreated: func(ctx context.Context, e *webhook.Event, a *freego.ResolvedAnnouncement) error {
			log.Printf("announcement %d with %d products", a.ID, len(a.ResolvedProducts))
			for _, p := range a.ResolvedProducts {
				if p.ShouldIgnore() {
					continue
				}
				url, _ := p.BestURL(freego.URLFlagOpensInBrowser)
				log.Printf("  %s [%s, %s] %s", p.Title, p.Type, p.Store, url.URL)
			}
			return nil
		},
		OnProductUpdated: func(ctx context.Context, e *webhook.Event, p *freego.Product) error {
			log.Printf("product %d updated: %s", p.ID, p.Title)
			return nil
		},
		OnUnknown: func(ctx context.Context, e *webhook.Event) error {
			log.Printf("unhandled event %s: %s", e.Type, e.Data)
			return nil
		},
		OnError: func(ctx context.Context, e *webhook.Event, err error) {
			var rejected *webhook.RejectionError
			if errors.As(err, &rejected) {
				log.Printf("rejected delivery: %v", err)
				return
			}
			log.Printf("error handling %s: %v", e.Type, err)
		},
	})

	addr := ":" + getenv("FREEGO_WEBHOOK_PORT", "8080")
	route := getenv("FREEGO_WEBHOOK_ROUTE", "/webhook")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("listening on %s%s", addr, route)
	if err := webhook.ListenAndServe(ctx, addr, route, handler); err != nil {
		log.Fatal(err)
	}
	log.Print("stopped")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
