# ADR-0005: Public URL via cloudflared quick tunnel

- **Status**: Accepted
- **Date**: 2026-05-25
- **Deciders**: Sadettin Er

## Context

Track B (local minikube) needs an internet-reachable URL for the submission demo: a single `curl <public-url>/ping → pong`. The laptop sits behind home NAT with no inbound reachability and no fixed IP. We do not want to expose the cluster API or open router ports for a four-day case study.

## Decision

Expose the in-cluster ingress (`pingapp.local` on the minikube node) through a **cloudflared quick tunnel** (`cloudflared tunnel --url ...`). The tunnel is anonymous, ephemeral, and dials *out* from the laptop to Cloudflare's edge — no inbound exposure.

## Alternatives considered

- **ngrok free tier** — works, but the free plan has aggressive rate limits, a random subdomain that changes on restart, and a mandatory interstitial page on first visit for unauthenticated tunnels. cloudflared has none of those for quick tunnels and the rate ceiling is much higher.
- **Cloudflare named tunnel (Tunnel + custom domain)** — strictly better for anything long-lived (stable URL, TLS, access policies), but it needs a Cloudflare account, a domain on Cloudflare, and a token committed somewhere. Overkill for a four-day demo; called out as the "if time" upgrade path.
- **Port forwarding on the router + dynamic DNS** — exposes the home network, requires router config the user may not control, and gives no TLS by default. Hard no.
- **Tailscale Funnel** — also a pull-tunnel, also free, also good. Picked cloudflared because it has zero account requirement for the quick variant.

## Consequences

- **Positive:** No inbound network exposure — cloudflared dials out over HTTPS to Cloudflare's edge. The laptop's IP is never advertised.
- **Positive:** Automatic TLS at the public hostname. The browser shows a valid Cloudflare-issued cert.
- **Positive:** One command (`cloudflared tunnel --url http://localhost:8080`) prints a public URL — fits the brief's "single demo link" submission shape.
- **Trade-off accepted:** Quick tunnels are **ephemeral**. The hostname (`<random>.trycloudflare.com`) is invalidated when the process stops; a fresh run prints a new URL. The submission email must include the URL captured at submission time.
- **Trade-off accepted:** Cloudflare can rate-limit or null-route quick tunnels at any time. For demo traffic this is irrelevant; for production it would be disqualifying.
- **Risk to revisit:** If the case study is extended beyond a few days, upgrade to a named tunnel with a stable hostname — the diff is small (a token, a config file, a custom domain on Cloudflare).

## Notes

- The tunnel target is the **minikube node IP** with a `Host: pingapp.local` header rewrite via `cloudflared`'s `--http-host-header`, so the in-cluster nginx ingress routes the request correctly:
  `cloudflared tunnel --url http://$(minikube ip):80 --http-host-header pingapp.local`
- Logged in `Makefile` target `tunnel`. The public URL is the only thing that needs to be pasted into the submission.
