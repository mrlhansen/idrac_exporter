# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Release notes
for tagged versions are also generated automatically on the
[GitHub releases page](https://github.com/fjacquet/idrac_exporter/releases).

## [Unreleased]

## [1.3.3] - 2026-09-13

### Security

- `google.golang.org/grpc` 1.83.0 -> 1.83.2, fixing **GHSA-vp52-pcj8-j9qc** and
  **GHSA-2v4p-qf9q-27wj** (HIGH), pulled in with an `otel` 1.45.0 -> 1.46.0 refresh.

## [1.2.0] - 2026-08-01

This fork is being brought into the exporter-standards family. Highlights since the fork from
`mrlhansen/idrac_exporter`:

### Added

- Optional OTLP push: an off-by-default background snapshot loop over the configured hosts
  (`otlp:` / `collection:` blocks), with a per-host `idrac_up` health gauge.
- One-command Docker Compose quickstart: exporter + Prometheus (with starter alert rules) +
  Grafana (datasource and dashboards auto-provisioned).
- Dedicated PDU Grafana dashboard (`grafana/pdu.json`); a fleet-health row on the overview
  dashboard and a management/storage/power detail row on the per-machine dashboard.
- `--once` (collect every host once and print the exposition) and `--trace` (token-safe
  request logging) command-line flags; `--config-watch` hot-reload of the configuration file.
- Supply-chain hardening: semgrep and govulncheck in CI, a CycloneDX SBOM, SHA-pinned GitHub
  Actions, and a non-root container image.
- MkDocs documentation site and Architecture Decision Records.
- `/livez` and `/readyz` probe endpoints, always HTTP 200 and reading no configuration or
  collector state, so a probe can never restart a working exporter because a BMC went away.
- A `HEALTHCHECK` against `/livez` in both Dockerfiles and a matching `healthcheck:` in
  `docker-compose.yml` and `docker-compose.ghcr.yml`.

### Changed

- Unified host identity on a single `system` label across the scrape and OTLP paths; the
  bundled Grafana dashboards key on it.
- Migrated the CLI to cobra, logging to logrus, the collection fan-out to errgroup with
  bounded concurrency, and the Redfish transport to resty.
- Hardened Redfish JSON parsing so unparseable fields yield absent samples instead of zero.
- Renamed the module path to `github.com/fjacquet/idrac_exporter`; container images are
  published on GHCR.
- `/health` now returns a JSON body — `status`, `version`, `revision`, and one entry per
  configured BMC host — instead of an empty 200. The status code is still 200
  unconditionally.
- The Helm chart's liveness and readiness probes moved from `/health` to `/livez` and
  `/readyz`. `/health` is still served and still 200.
- The container base image is `alpine:latest` instead of `alpine:3.23`, matching the rest of
  the exporter family.

### Notes

- This is a hard fork and no longer tracks upstream `mrlhansen/idrac_exporter`.
- The public metric contract is preserved: the `idrac_` prefix (configurable), port `9348`,
  and the on-demand `/metrics?target=` model.

## [1.1.2] - 2026-07-10

### Security

- Bumped the Go directive to 1.26.5 to patch GO-2026-5856 (crypto/tls), which was failing
  govulncheck / `make ci` family-wide.
- Bumped `google.golang.org/grpc` to v1.83.0, closing **GO-2026-6061** (xDS RBAC and HTTP/2
  vulnerabilities). The vulnerability was reachable from the OTLP push path.

### Fixed

- Restored multi-arch GHCR image publishing via a `dockers_v2` block in `.goreleaser.yaml`;
  tagged releases now publish `ghcr.io/fjacquet/idrac_exporter` again instead of only binaries.
- Added `Dockerfile.goreleaser`, which `COPY`s the per-platform `${TARGETPLATFORM}` binary laid
  out by buildx instead of compiling from source.
