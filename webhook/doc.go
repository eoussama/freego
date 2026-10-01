// Package webhook receives FreeStuff webhook deliveries.
//
// FreeStuff pushes events (new announcements, product updates and pings) to
// the URL configured on https://dashboard.freestuffbot.xyz/. Deliveries
// follow the Standard Webhooks specification and are signed with your app's
// Ed25519 key. This package verifies them and hands typed events to your
// callbacks:
//
//	verifier, err := webhook.NewVerifier(publicKey) // from the dashboard
//	if err != nil {
//		log.Fatal(err)
//	}
//	handler := webhook.NewHandler(verifier, webhook.Config{
//		OnAnnouncementCreated: func(ctx context.Context, e *webhook.Event, a *freego.ResolvedAnnouncement) error {
//			for _, p := range a.ResolvedProducts {
//				fmt.Println("free:", p.Title)
//			}
//			return nil
//		},
//	})
//	http.Handle("/webhook", handler)
//
// The [Handler] is a plain [net/http.Handler], so it can be mounted on any
// router. [ListenAndServe] runs a dedicated server for it with graceful
// shutdown.
//
// # Delivery semantics
//
// The handler verifies the signature, rejects messages whose timestamp is
// outside the tolerance, and pins the compatibility date of the payload (see
// [Config]). Message ids are remembered for [Config.ReplayWindow] (24 hours
// by default): retries and replays of a received message are acknowledged
// without running the callbacks again.
//
// Callbacks run synchronously; their errors and panics are reported to
// [Config.OnError] and the delivery is still acknowledged, as FreeStuff asks.
// FreeStuff waits for the response, so keep callbacks short and hand long
// work to a queue or goroutine. Ids are recorded before callbacks run, so a
// delivery interrupted by a crash is not processed again; keep callbacks
// idempotent if you persist ids elsewhere.
//
// [Verifier] can also be used on its own with other HTTP frameworks.
package webhook
