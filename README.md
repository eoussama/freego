<p align="center">
  <img width="100" src="./assets/logo.png">
</p>

<h1 align="center">Freego</h1>
<p align="center">Go client for the FreeStuff API v2.</p>

<p align="center">
    <a href="https://pkg.go.dev/github.com/eoussama/freego"><img src="https://pkg.go.dev/badge/github.com/eoussama/freego.svg" alt="Go Reference"></a>
    <a href="https://github.com/eoussama/freego/actions/workflows/ci.yml"><img src="https://github.com/eoussama/freego/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
    <img src="https://img.shields.io/github/v/tag/eoussama/freego" />
    <img src="https://img.shields.io/github/license/eoussama/freego" />
</p>

## Description

Freego is a Go library for the [FreeStuff API](https://docs.freestuffbot.xyz/), the API behind the [FreeStuff](https://freestuffbot.xyz/) Discord bot. It gives your services the same data and events the bot uses to announce free games:

- **Webhooks** (package [`webhook`](./webhook)): receive new announcements and product updates as they happen. Deliveries are verified (Ed25519 signature, timestamp, replay protection) and handed to typed callbacks.
- **REST API** (package `freego`): list and look up products, and read the API's schemas, problem types and event types.

It has no dependencies outside the Go standard library.

| | Free tier | Full tier |
|---|---|---|
| Webhooks | ✓ | ✓ |
| `Ping`, `Schemas`, `Schema`, `Problems`, `Events` | ✓ | ✓ |
| `Products`, `ProductsIter`, `Product` | ✗ (`ErrUnavailableForFreeTier`) | ✓ |

## Usage

### Prerequisites

- Go `1.22` or later.
- A FreeStuff application. Create one on the [FreeStuff dashboard](https://dashboard.freestuffbot.xyz/), sign up for a plan under *Plans & Billing*, and copy your **REST API key** and **public key** from *my application*.

### Installation

```sh
go get github.com/eoussama/freego
```

### Receiving webhooks

Set your webhook URL on the dashboard, then mount a handler on it:

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/eoussama/freego"
	"github.com/eoussama/freego/webhook"
)

func main() {
	verifier, err := webhook.NewVerifier(os.Getenv("FREEGO_FREESTUFF_PUBLIC_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	handler := webhook.NewHandler(verifier, webhook.Config{
		OnAnnouncementCreated: func(ctx context.Context, e *webhook.Event, a *freego.ResolvedAnnouncement) error {
			for _, p := range a.ResolvedProducts {
				if p.ShouldIgnore() {
					continue
				}
				url, _ := p.BestURL(freego.URLFlagOpensInBrowser)
				log.Printf("%s is free on %s: %s", p.Title, p.Store, url.URL)
			}
			return nil
		},
		OnProductUpdated: func(ctx context.Context, e *webhook.Event, p *freego.Product) error {
			log.Printf("%s was updated", p.Title)
			return nil
		},
		OnError: func(ctx context.Context, e *webhook.Event, err error) {
			log.Print(err)
		},
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := webhook.ListenAndServe(ctx, ":8080", "/webhook", handler); err != nil {
		log.Fatal(err)
	}
}
```

`webhook.Handler` is a regular `http.Handler`, so you can also mount it on your own server or router (`mux.Handle("/webhook", handler)`). The handler:

- verifies the `Webhook-Signature` against your public key and rejects timestamps more than 5 minutes off (`webhook.WithTolerance` changes this);
- remembers message ids for 24 hours (`Config.ReplayWindow`), so FreeStuff's retries and replayed requests are acknowledged without running your callbacks twice. Ids are kept in memory by default; implement `webhook.ReplayCache` to share them across instances or keep them across restarts;
- runs callbacks synchronously, recovers panics, reports errors to `OnError` and answers `204` regardless, as FreeStuff requires;
- pins the payload format to the library's compatibility date (see [Compatibility dates](#compatibility-dates)).

On another framework, use `verifier.Verify(id, timestamp, signature, rawBody)` directly. To test your own handlers, `webhook.Sign` produces deliveries signed like FreeStuff's.

### Using the REST API

```go
client, err := freego.New(os.Getenv("FREEGO_FREESTUFF_API_KEY"))
if err != nil {
	log.Fatal(err)
}

ctx := context.Background()

// Check the API and the key. Works on every plan.
if err := client.Ping(ctx); err != nil {
	log.Fatal(err)
}

// List products (full tier). Without Resolve, only partial products are
// returned: ID, Kind, Until, Type, Flags and Store.
list, err := client.Products(ctx, &freego.ProductsQuery{
	Type:    freego.ChannelKeep,
	Resolve: true,
})
if errors.Is(err, freego.ErrUnavailableForFreeTier) {
	log.Fatal("listing products requires the full tier")
}

// Walk every page.
it := client.ProductsIter(&freego.ProductsQuery{Resolve: true})
for it.Next(ctx) {
	fmt.Println(it.Product().Title)
}
if err := it.Err(); err != nil {
	log.Fatal(err)
}

// Look up one product.
product, err := client.Product(ctx, 123456)
```

To poll cheaply, send back the `ETag` of the previous result; unchanged results return `freego.ErrNotModified`:

```go
list, err := client.Products(ctx, &freego.ProductsQuery{IfNoneMatch: previous.ETag})
if errors.Is(err, freego.ErrNotModified) {
	// nothing changed
}
```

Options: `freego.WithHTTPClient` (default: 30s timeout), `freego.WithUserAgent`, `freego.WithBaseURL`, `freego.WithCompatibilityDate`. All methods take a `context.Context`.

### Working with products

```go
p.Description.Get("de")              // localized description, falls back to English
p.Price("USD")                       // prices are in minor units: 499 = $4.99
p.BestImage(freego.ImageFlagWide)    // highest priority image with the given flags
p.BestURL(freego.URLFlagOpensInClient)
p.MetaValue(freego.MetaSteamSubIDs)  // metadata (full tier)
p.Flags.Has(freego.ProductFlagTrash) // low quality offer
p.Until.IsZero()                     // no known end date
p.ShouldIgnore()                     // FreeStuff asks not to show this product
```

Enumerations (`Channel`, `ProductKind`, `Store`, `Platform`) and bitfields are open-ended: values unknown to the library are kept as-is.

### Errors

API errors are `*freego.Problem` values decoded from FreeStuff's [RFC 7807](https://datatracker.ietf.org/doc/html/rfc7807) problem documents:

```go
var problem *freego.Problem
if errors.As(err, &problem) {
	log.Println(problem.Status, problem.Type, problem.Detail)
}

errors.Is(err, freego.ErrUnauthorized)           // missing, invalid or reset API key
errors.Is(err, freego.ErrUnavailableForFreeTier) // content endpoint on the free tier
errors.Is(err, freego.ErrRateLimited)            // see problem.RetryAfter
errors.Is(err, freego.ErrNotFound)               // any 404
```

`client.Problems(ctx)` lists every problem type the API can return.

### Compatibility dates

FreeStuff versions its data model with [compatibility dates](https://docs.freestuffbot.xyz/api-v2/concepts#compatibility-dates). This version of freego is built for **`2026-06-08`** (`freego.CompatibilityDate`), and pins it everywhere:

- **REST**: every request sends `X-Compatibility-Date`.
- **Webhooks**: each response carries `X-Set-Compatibility-Date`. A delivery in another format is answered with `400`, which makes FreeStuff switch your app to the library's date and redeliver the message in that format, so no event is lost. **This changes the compatibility date configured on your dashboard.** An app is created with the date of its creation day, so a newly created app will usually differ from the library's date. Behind a proxy that hides the response from FreeStuff (such as smee), the switch cannot happen. To keep managing the date yourself, set `webhook.Config.AcceptAnyCompatibilityDate`; deliveries in other formats are then accepted as they are, and those in the older `2025-03-01` format still decode.

## Examples

Runnable programs live in [`examples/`](./examples). They read the variables from [`.env.example`](./.env.example):

```sh
cp .env.example .env   # then fill in your keys
set -a && . ./.env && set +a

go run ./examples/products   # REST: ping, schemas, current free games
go run ./examples/webhook    # webhook server logging every event
```

`examples/webhook` also reads `FREEGO_WEBHOOK_PORT`, `FREEGO_WEBHOOK_ROUTE` and `FREEGO_WEBHOOK_ACCEPT_ANY_DATE`. See [Receiving webhooks locally](#receiving-webhooks-locally) for a step-by-step test with the dashboard's ping button.

Remove `FREEGO_FREESTUFF_API_URL` from your `.env` unless you want to override the base URL (it must point at `…/v2`).

## Testing

```sh
go test -race ./...
```

The tests run offline against local servers and locally generated keys. Live tests against the real API are opt-in:

```sh
FREEGO_FREESTUFF_API_KEY=... FREEGO_FREESTUFF_PUBLIC_KEY=... go test -tags live -run Live ./...
```

They also check that the live `Product` schema still matches the one the Go types are tested against (`testdata/`).

### Receiving webhooks locally

FreeStuff delivers webhooks from the internet, so your local server needs a public URL. There are two ways to get one. Either way, the receiver is [`examples/webhook`](./examples/webhook) and the test is the **ping** button on your app's page on the [FreeStuff dashboard](https://dashboard.freestuffbot.xyz/).

You need your app's **public key** in `.env` (`FREEGO_FREESTUFF_PUBLIC_KEY`, see [`.env.example`](./.env.example)).

#### Option A: smee in the Docker dev container

[smee](https://smee.io/) is a free relay: it gives you a public URL and forwards what it receives to your machine. The project ships a Docker image with smee preinstalled, so nothing needs installing besides Docker.

1. **Create a channel.** Open https://smee.io/new. It shows a URL like `https://smee.io/AbC123xYz`.

2. **Start the dev container** from the project root. The prompt changes to the container's:

   ```sh
   scripts/start.sh
   ```

3. **Start smee in the container**, with your channel URL (not a placeholder: `<channel>` makes the shell fail with `No such file or directory`). It runs in the background:

   ```sh
   scripts/smee.sh https://smee.io/AbC123xYz 8080
   ```

   It prints `Connected to https://smee.io/…` and `Forwarding … to http://127.0.0.1:8080/webhook`.

4. **Open a second terminal** on your machine and join the same container:

   ```sh
   docker exec -it freego-dev bash
   ```

5. **Start the receiver there.** `docker exec` does not load `.env`, so do that first:

   ```sh
   cd /go/src/github.com/eoussama/freego
   set -a && . ./.env && set +a
   FREEGO_WEBHOOK_ACCEPT_ANY_DATE=true go run ./examples/webhook
   ```

   Wait for `listening on :8080/webhook`. The receiver must run **inside the container**: smee forwards to `127.0.0.1:8080` in there, and a server on your own machine is not reachable that way (smee reports `ECONNREFUSED`).

6. **Point FreeStuff at smee.** On the dashboard, set your app's webhook URL to your smee channel URL.

7. **Press ping** on the dashboard. The receiver prints:

   ```text
   ping msg_… (manual: true)
   ```

   That confirms the whole chain: delivery, signature verification against your public key, and your callback. The channel page (`https://smee.io/AbC123xYz`) lists every event smee received and lets you redeliver them.

**Why `FREEGO_WEBHOOK_ACCEPT_ANY_DATE=true`?** By default the handler answers a delivery in another data format with `400` plus an `X-Set-Compatibility-Date` header, which makes FreeStuff switch your app to the library's [compatibility date](#compatibility-dates) and redeliver. smee does not pass the response back to FreeStuff, so that never happens, and an app whose dashboard date differs from the library's (apps get the date they were created on, for example `2026-10-01`) would see every ping rejected with `compatibility date mismatch`. This switch accepts the dashboard's date as it is, which is safe for testing a ping.

#### Option B: a tunnel on your machine

Tunnels such as [cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/do-more-with-tunnels/trycloudflare/) or [ngrok](https://ngrok.com/) forward requests with their bodies untouched and relay your responses, so the default behaviour works and no Docker is needed:

```sh
set -a && . ./.env && set +a
go run ./examples/webhook                           # listens on :8080/webhook
cloudflared tunnel --url http://localhost:8080      # in another terminal, prints a public URL
```

Set the dashboard's webhook URL to the printed address followed by `/webhook`, then press ping. If port 8080 is busy (for example because the dev container publishes it), choose another one with `FREEGO_WEBHOOK_PORT=9090` for the receiver and the tunnel.

#### Troubleshooting

| Symptom | Cause and fix |
|---|---|
| `No such file or directory` from `scripts/smee.sh` | The channel URL was a placeholder. Use the real URL from https://smee.io/new. |
| smee prints `ECONNREFUSED 127.0.0.1:8080` | Nothing is listening in the container. Start the receiver in a shell inside it (step 5). |
| `bind: address already in use` | Another process holds the port. `ss -ltnp \| grep 8080` shows which. The dev container publishes its port on your machine, so run the receiver inside it or pick another port. |
| `rejected delivery … compatibility date mismatch` | The dashboard's date differs from the library's. Set `FREEGO_WEBHOOK_ACCEPT_ANY_DATE=true` (smee) or use a tunnel (option B). This line also proves the signature was valid, as signatures are checked first. |
| `rejected delivery … invalid signature` | The body was altered on the way (a relay that re-encodes JSON), or the public key does not belong to this app. Check the key on the dashboard; try a tunnel. |
| `rejected delivery … 400 … invalid timestamp` | The clock of your machine (or container) is more than 5 minutes off. |
| Nothing arrives | Check that the webhook URL on the dashboard is your channel (or tunnel) URL, and look at the smee channel page to see whether FreeStuff delivered. |

## Upgrading from v0.0.x

v0.0.x was written for FreeStuff API v1, which FreeStuff has deprecated. v0.1.0 is a rewrite for API v2, and nothing carries over unchanged:

| v0.0.x | v0.1.0 |
|---|---|
| `freego.Config(&models.Options{...})` + `freego.Init(config)` | `freego.New(apiKey, opts...)` |
| `src/enums`, `src/models`, `src/types` packages | everything in `freego` and `freego/webhook` |
| `client.Ping()` | `client.Ping(ctx)` |
| `client.GetGames(filter)` (ids) | `client.Products(ctx, query)` / `client.ProductsIter(query)` |
| `client.GetGameInfo(ids, languages)` | `client.Product(ctx, id)`, `Products` with `Resolve: true`; localized descriptions in `Product.Description` |
| `client.GetGameAnalytics(...)` | removed: FreeStuff dropped partner endpoints |
| `client.On(enums.EventFreeGames, cb)` (blocking, shared secret) | `webhook.NewHandler(verifier, webhook.Config{OnAnnouncementCreated: ...})` (signature verified) |
| `models.GameInfo` | `freego.Product` |
| `FREEGO_WEBHOOK_SECRET` | `FREEGO_FREESTUFF_PUBLIC_KEY` (Ed25519 public key) |
| configuration read from the environment | explicit options; only the examples read the environment |

See the [changelog](./CHANGELOG.md) for details.

## License

[Apache 2.0](./LICENSE)
