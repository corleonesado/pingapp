#!/usr/bin/env bash
# Track B IaC substitute: verify the local toolchain is sane.
# Idempotent — safe to re-run. Prints install hints for anything missing.
#
# Usage: ./scripts/bootstrap.sh

set -euo pipefail

# --- helpers ---------------------------------------------------------------

CYAN=$'\033[1;36m'; RED=$'\033[1;31m'; YELLOW=$'\033[1;33m'; GREEN=$'\033[1;32m'; RESET=$'\033[0m'

say()   { printf "%s==>%s %s\n"     "$CYAN" "$RESET" "$*"; }
ok()    { printf "%s    OK%s   %s\n"  "$GREEN" "$RESET" "$*"; }
warn()  { printf "%s    WARN%s %s\n"  "$YELLOW" "$RESET" "$*"; }
fail()  { printf "%s    MISS%s %s\n"  "$RED"    "$RESET" "$*"; }

missing=0

tool_version() {
  case "$1" in
    kubectl)   kubectl version --client 2>/dev/null | head -n1 ;;
    helm)      helm version --short 2>/dev/null ;;
    minikube)  minikube version --short 2>/dev/null || minikube version 2>/dev/null | head -n1 ;;
    go)        go version 2>/dev/null ;;
    *)         "$1" --version 2>&1 | head -n1 ;;
  esac
}

check() {
  local cmd="$1" hint="$2"
  if command -v "$cmd" >/dev/null 2>&1; then
    ok "$cmd  ($(tool_version "$cmd"))"
  else
    fail "$cmd  — install: $hint"
    missing=$((missing+1))
  fi
}

# --- required tools --------------------------------------------------------

say "Checking required tools"
check docker       "OrbStack (brew install --cask orbstack) or Docker Desktop"
check kubectl      "brew install kubectl"
check helm         "brew install helm"
check minikube     "brew install minikube"
check go           "brew install go"
check git          "Xcode CLT or brew install git"
check gh           "brew install gh"
check cloudflared  "brew install cloudflared"

say "Checking CI / supply-chain tools (recommended)"
check golangci-lint "brew install golangci-lint"
check trivy         "brew install trivy"
check gitleaks      "brew install gitleaks"

# --- runtime checks --------------------------------------------------------

say "Runtime checks"

if docker info >/dev/null 2>&1; then
  ok "docker daemon reachable"
else
  warn "docker daemon is not reachable (start OrbStack/Docker Desktop)"
fi

if minikube status 2>/dev/null | grep -q "host: Running"; then
  ok "minikube is running"
else
  warn "minikube not running — run: make minikube-up"
fi

if helm repo list 2>/dev/null | grep -q '^prometheus-community'; then
  ok "helm repo 'prometheus-community' registered"
else
  warn "helm repo 'prometheus-community' missing — run: helm repo add prometheus-community https://prometheus-community.github.io/helm-charts && helm repo update"
fi

# --- summary ---------------------------------------------------------------

if [ "$missing" -eq 0 ]; then
  say "${GREEN}All required tools present.${RESET}"
  printf "\nNext steps:\n  make minikube-up\n  make deploy-prod          # or use ArgoCD: make argocd-install && make argocd-app TAG=v0.1.0\n  make tunnel               # public URL\n"
  exit 0
else
  say "${RED}$missing tool(s) missing.${RESET} Install them, then re-run."
  exit 1
fi
