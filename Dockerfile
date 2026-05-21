# syntax=docker/dockerfile:1.7
# Build toolchain is pinned ahead of go.mod's language floor (1.22) so the
# compiled binary picks up Go stdlib security fixes (see ADR-0004). Trivy
# fails the build on HIGH/CRITICAL, which older toolchains trip on.
ARG GO_VERSION=1.26

FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /src

# Cache deps in a separate layer; go.sum may not exist if there are no external deps.
COPY go.mod ./
COPY go.sum* ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/server ./cmd/server

# --- runtime ---
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder /out/server /app/server
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/app/server"]
