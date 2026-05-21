# ADR-0004: Image scanning gate — Trivy on HIGH/CRITICAL

- **Status**: Accepted
- **Date**: 2026-05-21
- **Deciders**: Sadettin Er

## Context

The case study requires image vulnerability scanning that fails the pipeline on serious findings, and explicitly flags "turning off or removing the image scan step" as a thing to avoid. We need a threshold that is strict enough to be meaningful but not so strict that unfixable noise blocks every build. The scanner runs on every PR and every push to main, plus on each release.

## Decision

Trivy scans the built image and fails the job (`exit-code: 1`) on `HIGH` and `CRITICAL` severities, with `--ignore-unfixed` so we only block on vulnerabilities that actually have a fix available. The same gate runs in `ci.yml` (per PR/push) and `release.yml` (per tag).

## Alternatives considered

- **Fail on any severity (LOW and up)** — too noisy; distroless + a static Go binary still surface informational CVEs with no fix, which would train us to ignore the gate.
- **Scan but never fail (report-only)** — defeats the purpose; the brief wants the pipeline to actually break on serious findings.
- **Block including unfixed CVEs** — would fail builds for vulnerabilities we cannot remediate (no upstream patch), forcing either a `.trivyignore` graveyard or disabling the gate. `--ignore-unfixed` keeps the signal actionable.
- **Grype / Snyk instead of Trivy** — fine tools, but the brief names Trivy and it scans both the OS layer and the Go binary's embedded module versions in one pass.

## Consequences

- **Positive:** A real, enforced gate — proven during Day 3 setup, when Trivy caught HIGH/CRITICAL Go **stdlib** CVEs compiled into the binary by the `golang:1.22-alpine` builder.
- **Positive:** The remediation was a fix, not a bypass: bumping the build toolchain to `golang:1.26-alpine` (Go ≥ 1.26.3) cleared all findings. The go.mod language floor stays at 1.22; only the build toolchain moved (documented in the Dockerfile).
- **Positive:** `--ignore-unfixed` means a red build always corresponds to an action we can take (bump base image, bump a dependency, pin a fixed version).
- **Trade-off accepted:** A newly disclosed, fixed CVE in the base or stdlib can turn `main` red without any code change. That is intended — it is the signal to rebuild on a patched base.
- **Trade-off accepted:** `--ignore-unfixed` means an unfixed CRITICAL won't block us. We accept this as a known gap and rely on the distroless base's small surface to keep it rare.
- **Risk to revisit:** If we ever need to ship despite an unavoidable finding, add a reviewed, time-boxed `.trivyignore` entry with a justification comment — never disable the step.

## Notes

- Trivy invocation: `--severity HIGH,CRITICAL --ignore-unfixed --exit-code 1`.
- The distroless `static:nonroot` base reports as Debian; at time of writing it scans with 0 OS findings.
