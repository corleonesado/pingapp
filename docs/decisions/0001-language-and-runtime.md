# ADR-0001: Language and runtime — Go on distroless

- **Status**: Accepted
- **Date**: 2026-05-20
- **Deciders**: Sadettin Er

## Context

The case study asks for a tiny HTTP service shipped end-to-end with a focus on clear DevOps practices: small images, image scanning, non-root containers, fast CI feedback. The choice of language and base image shapes every downstream step (Dockerfile structure, Trivy surface, CI time, K8s probe behavior). The brief explicitly permits Go, Node.js, or Python.

## Decision

Go 1.22+ with `gcr.io/distroless/static:nonroot` as the runtime base image, produced via a multi-stage build.

## Alternatives considered

- **Python + FastAPI on `python:3.12-slim`** — fastest to write, but produces a 100MB+ image with a wider Trivy surface and slower CI builds. Requires a process supervisor or careful uvicorn config for graceful shutdown.
- **Node.js + Express on `node:20-alpine`** — middle ground, but the npm dependency surface tends to attract Trivy findings, and the runtime image still carries libc.
- **Go on `alpine:3.19`** — smaller than slim but still has a shell, package manager, and the musl/glibc footgun. `distroless/static` is strictly smaller and safer for a statically-linked Go binary.

## Consequences

- **Positive:** Final image is ~10MB (single static binary + CA certs + tzdata).
- **Positive:** No shell, no package manager, no userland — minimal Trivy findings, smaller blast radius.
- **Positive:** Built-in non-root user (UID 65532) — satisfies the case study's non-root requirement with zero config.
- **Positive:** Fast CI builds (~30s for a clean Go build vs ~2m for a clean Python image).
- **Trade-off accepted:** No shell means no `HEALTHCHECK` directive that uses `curl` or `wget`. We rely on Kubernetes probes instead, which is the production-grade pattern anyway.
- **Trade-off accepted:** Debugging inside the container requires `kubectl debug` with an ephemeral container. Acceptable for this scope.
- **Risk to revisit:** If we later need CGO (e.g., for a SQLite driver), we must switch to `distroless/base` and rebuild with `CGO_ENABLED=1`.

## Notes

- Build flags: `CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w -X main.version=$SHA"`.
- The `nonroot` variant pins UID/GID to 65532, which K8s `securityContext.runAsNonRoot: true` will validate.
