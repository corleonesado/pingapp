## What

<!-- One paragraph: what does this PR change? -->

## Why

<!-- The motivation. Link an issue or ADR if relevant. -->

## Screenshots / evidence

<!-- kubectl / helm / grafana output, curl runs, etc. -->

## Checklist

- [ ] Tests pass locally (`make test`)
- [ ] Lint passes (`make lint`)
- [ ] No secrets introduced (`gitleaks detect --no-git` clean)
- [ ] No `:latest` tags in manifests / compose / values
- [ ] `ROADMAP.md` checkboxes updated
- [ ] `CHANGELOG.md` updated under `[Unreleased]` (user-facing changes only)
- [ ] If a non-obvious decision was made, an ADR was added or updated in `docs/decisions/`
