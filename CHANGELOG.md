# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

## [0.1.0] - 2026-10-01

Rewrite for the FreeStuff API v2. FreeStuff deprecated API v1, which v0.0.x
was built on, so this release replaces the whole public API. See
"Upgrading from v0.0.x" in the README.

### Added

- `freego.Client` for the REST API v2, created with `freego.New(apiKey, opts...)`:
  `Ping`, `Products`, `ProductsIter`, `Product`, `Schemas`, `Schema`,
  `Problems`, `Events`. Every method takes a `context.Context`.
- ETag support: `ProductList.ETag`, `ProductsQuery.IfNoneMatch` and
  `ErrNotModified`.
- `freego.Problem` errors decoded from RFC 7807 problem documents, with
  `errors.Is` sentinels (`ErrUnauthorized`, `ErrUnavailableForFreeTier`,
  `ErrRateLimited`, `ErrNotFound`, ...) and `Retry-After` support.
- Data model of compatibility date 2026-06-08: `Product`, `ProductPrice`,
  `ProductImage`, `ProductURL`, `ProductMeta`, `Descriptions`, `Announcement`,
  `ResolvedAnnouncement`, open enums (`Channel`, `ProductKind`, `Store`,
  `Platform`) and bitfields (`ProductFlags`, `ImageFlags`, `URLFlags`).
  Product helpers: `ShouldIgnore`, `MetaValue`, `Price`, `BestImage`,
  `BestURL`, `Descriptions.Get`.
- `webhook` package: Ed25519 signature verification (`Verifier`), replay
  protection over a 24 hour window (`ReplayCache`, `Config.ReplayWindow`), an
  `http.Handler` with typed callbacks, a `ListenAndServe`/`Serve` helper with
  graceful shutdown, and `Sign` for testing.
- Lenient decoding where the API only promises "a number": IDs, timestamps,
  prices and flags in exponent or non-integral notation, and negative 32-bit
  flags, decode instead of failing the whole product. An unparsable envelope
  timestamp falls back to the delivery time instead of rejecting the
  delivery.
- A step-by-step guide to testing webhooks locally with FreeStuff's ping
  button (README, "Receiving webhooks locally"), verified end to end through
  smee. The dev container is now named `freego-dev`, and `examples/webhook`
  reads `FREEGO_WEBHOOK_ACCEPT_ANY_DATE`.
- Tests (offline, plus opt-in live tests with `-tags live`), godoc examples,
  runnable programs in `examples/`, GitHub Actions CI and this changelog.

### Changed

- The library no longer reads environment variables; configuration is passed
  explicitly. The examples read the `FREEGO_*` variables.
- Webhook deliveries are authenticated with FreeStuff's Ed25519 signature
  instead of a shared secret: `FREEGO_WEBHOOK_SECRET` is replaced by
  `FREEGO_FREESTUFF_PUBLIC_KEY`.

### Removed

- The v1 API surface: `Config`, `Init`, `GetGames`, `GetGameInfo`,
  `GetGameAnalytics`, `GetEvent`, `On` and the `src/...` packages.
- Partner analytics, which API v2 no longer offers.
- `FREEGO_FREESTUFF_API_ORIGIN`, which was never used.

### Fixed

These issues of v0.0.x are gone with the rewrite:

- The webhook handler dereferenced a nil event, and panicked, when a request
  could not be decoded or had the wrong secret.
- `On` registered on `http.DefaultServeMux`, so a second call panicked, and it
  blocked forever with no way to shut down.
- The webhook secret was compared in non-constant time.
- HTTP status codes were never checked, requests had no timeout and GET
  requests sent a `null` body.
- `Config(nil)` panicked.
- The `footer` localization field was decoded from the wrong JSON key
  (`"string"`), and the "ad" announcement type was spelled `"add"`.
- The `lang` query string was appended as a path segment (`/info/?lang=`).

Development tooling:

- The Docker image no longer copies `.env`, and the API key in it, into an
  image layer (new `.dockerignore`).
- `scripts/entrypoint.sh` no longer truncates values containing `#` and
  starts without a `.env` file.
- Removed the no-op `RUN source .env` and the `PRJECT_PATH` typo from the
  Dockerfile, and quoted the arguments of the scripts.

### Notes on the FreeStuff API

Findings from the live API (2026-10-01) that differ from its documentation:

- `GET /ping` returns 404, so `Client.Ping` requests `/static/schemas`.
- REST requests select their compatibility date with the
  `X-Compatibility-Date` request header. Without it the API answers in the
  newest format.
- `GET /products/:id` is undocumented and could not be verified, since
  content endpoints need the full tier. `Client.Product` accepts both a bare
  product and one wrapped in a `product` member.

## [0.0.5] - 2025-03-13

### Fixed

- Skip nullish game info ids (#43).

[0.1.0]: https://github.com/eoussama/freego/compare/v0.0.5...v0.1.0
[0.0.5]: https://github.com/eoussama/freego/releases/tag/v0.0.5
