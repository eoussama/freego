# Contributing to Freego

Thank you for your interest in contributing to **Freego**! We appreciate your help in making the project better.

Before you start contributing, please take a moment to review this guide, which outlines the process for contributing and the community guidelines we follow.

## Code of Conduct

We have a [Code of Conduct](CODE_OF_CONDUCT.md) that we expect all contributors to adhere to. Please read it and make sure you understand and follow it in all your interactions within our project.

## Getting Started

- If you are new to our project, please review our [README](README.md) for an overview of the project and how to get started.

- Check our [Issues](https://github.com/eoussama/freego/issues) to find tasks that need assistance or to report any bugs or feature requests.

## How to Contribute

1. Fork the repository to your GitHub account.

2. Create a new branch for your contribution, named after its issue:

   ```bash
   git checkout -b feat/<issue>-your-feature-name   # or fix/<issue>-...
   ```

3. Make your changes, with tests. Before opening a pull request, make sure these pass (CI runs them on Go 1.22 and the latest release):

   ```bash
   gofmt -l .          # must print nothing
   go vet ./...
   go test -race ./...
   ```

4. If you have a FreeStuff API key, also run the live tests against the real API:

   ```bash
   FREEGO_FREESTUFF_API_KEY=... FREEGO_FREESTUFF_PUBLIC_KEY=... go test -tags live -run Live ./...
   ```

5. Commit using the project's prefixes (`feat:`, `fix:`, `proj:`) and reference the issue, e.g. `fix: decode empty descriptions, fixes #50`. Add an entry to [CHANGELOG.md](CHANGELOG.md).

6. Open a pull request against `develop`.

## Project Layout

- `*.go` (root): the `freego` package: REST client, data models and errors.
- `webhook/`: webhook verification and HTTP handler.
- `examples/`: runnable programs.
- `testdata/`: fixtures, including the live `Product` JSON schema for the compatibility date in `version.go`.
- `docker/`, `scripts/`: development container with smee for receiving webhooks locally.

## Updating the Compatibility Date

When FreeStuff publishes a newer data model (`go test -tags live` logs a note when `Schemas` reports a newer version):

1. Save the new schema: `GET /static/schemas/fsb:schema:apiv2:product` with the `X-Compatibility-Date` header, as `testdata/schema_product_<date>.json`.
2. Update `CompatibilityDate` in `version.go`.
3. Adjust the types in `models.go` until `go test ./...` passes (`TestProductMatchesLiveSchema` reports missing fields), and keep older payloads decoding where possible.
4. Since webhook payload formats change with the date, release it as a new minor version.
