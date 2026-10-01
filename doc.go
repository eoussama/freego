// Package freego is a Go client for the FreeStuff API v2
// (https://docs.freestuffbot.xyz/), the API behind the FreeStuff Discord bot
// that announces free games.
//
// FreeStuff delivers its data in two ways, and freego covers both:
//
//   - Webhooks push announcements and product updates to your server as they
//     happen. They are available on every plan, including the free tier. See
//     the [github.com/eoussama/freego/webhook] package.
//   - The REST API, served by [Client], lists and looks up products (paid
//     "full" tier only) and exposes static metadata such as JSON schemas,
//     problem types and event types (all tiers).
//
// # Getting started
//
// Create an application on https://dashboard.freestuffbot.xyz/ to obtain a
// REST API key and your app's Ed25519 public key, then:
//
//	client, err := freego.New(apiKey)
//	if err != nil {
//		log.Fatal(err)
//	}
//	list, err := client.Products(ctx, &freego.ProductsQuery{Resolve: true})
//
// # Errors
//
// API errors are returned as [*Problem] values, decoded from the RFC 7807
// problem documents FreeStuff responds with. Common cases can be matched with
// errors.Is against the predefined sentinels:
//
//	if errors.Is(err, freego.ErrUnavailableForFreeTier) { ... }
//
// # Compatibility dates
//
// FreeStuff versions its data model with compatibility dates. Every request
// made by this library pins [CompatibilityDate], so the data always matches
// the Go types in this package regardless of the setting on your dashboard.
package freego
