# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Tiny HTTP service in Go with `/ping`, `/healthz`, and `/version` endpoints.
- Structured JSON logging via `log/slog`.
- Request ID middleware (UUID v4 per request, propagated as `X-Request-ID`).
- Multi-stage `Dockerfile` producing a `gcr.io/distroless/static:nonroot` image (~10MB, non-root UID 65532).
- `docker-compose.yaml` for local development (read-only fs, dropped caps).
- `Makefile` with `run`, `test`, `lint`, `docker-build`, `docker-run` targets.
- ADR-0001: Language and runtime (Go on distroless).
