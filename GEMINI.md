# GEMINI.md

## Project Overview
- O-RAN Near-RT RIC, three interfaces: E2 (SCTP+ASN.1 PER), A1 (REST+JWT+RBAC), O1 (NETCONF/YANG + FCAPS).
- Current issues: incomplete protocol stacks, CI broken, security defaults, test coverage <5%.

## Coding Standards
- Go 1.21, modules mode.
- `golangci-lint` for linting, `go fmt`.
- RFC-style comments on every public function.
- Unix LF, UTF-8 encoding.

## Build & Test
- `make build` → compile binaries.
- `make test-coverage` → run tests + enforce ≥80% coverage.
- `helm lint helm/oran-nearrt-ric/` → validate charts.

## Deployment
- Docker multi-stage build, non-root user.
- Use external Kubernetes secrets: `redis-secret`, `postgres-secret`.
- CI workflow file: `.github/workflows/ci.yml`.

## Tools & Commands
- `/stats` to monitor token usage.
- Use `!git diff --stat` after each code change.
- Use `@pkg/e2/asn1.go` to inject file content.

## Do Not
- Do not modify README.md until code is fully fixed.
- Do not introduce new TODOs.
- Do not bypass authentication or leave hardcoded secrets.
