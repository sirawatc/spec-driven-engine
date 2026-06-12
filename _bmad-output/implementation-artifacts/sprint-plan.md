# Sprint Plan — Service Engine PoC

**Version:** 1.0  
**Date:** 2026-06-13  
**Total Stories:** 17  
**Status:** In Progress

---

## How to Use This Plan

- Stories are ordered by dependency — implement top to bottom
- Mark each story `[x]` when complete before moving to the next
- Stories within the same phase that share no dependency can be implemented in parallel
- Dev agent works one story at a time: `bmad-create-story` → `bmad-dev-story` → `bmad-code-review`

---

## Sprint Progress

| Phase | Stories | Done | Status |
|-------|---------|------|--------|
| 1 — Foundation | E1-S1, E1-S2, E1-S3 | 0/3 | Not Started |
| 2 — Core Components | E2-S1–S3, E3-S1–S3, E4-S1–S2 | 0/8 | Blocked on Phase 1 |
| 3 — Assembly | E5-S1, E5-S2, E6-S1, E6-S2 | 0/4 | Blocked on Phase 2 |
| 4 — PoC Validation | E7-S1, E7-S2 | 0/2 | Blocked on Phase 3 |

---

## Phase 1 — Foundation
> Must complete in order. Everything else depends on this.

- [ ] **E1-S1** — Project scaffold
  - _Depends on:_ nothing
  - _Delivers:_ compilable Go module, folder structure, `main.go` skeleton, chi dependency

- [ ] **E1-S2** — Spec loader
  - _Depends on:_ E1-S1
  - _Delivers:_ `spec.Load()`, typed `Spec` struct, startup audit log

- [ ] **E1-S3** — Multi-version spec registry
  - _Depends on:_ E1-S2
  - _Delivers:_ `SpecRegistry`, directory scan from `ENGINE_SPEC_DIR`, startup version log

---

## Phase 2 — Core Components
> All depend on Phase 1. Stories within this phase are independent of each other — implement in any order.

### HTTP Runtime

- [ ] **E2-S1** — HTTP server with dynamic route registration
  - _Depends on:_ E1-S3
  - _Delivers:_ chi server, versioned routes (`/v1/...`), middleware stack, `/health`

- [ ] **E2-S2** — Request body validation
  - _Depends on:_ E1-S2
  - _Delivers:_ JSON parse check (→ 31000), required field check (→ 32000)

- [ ] **E2-S3** — `systemCode` resolution
  - _Depends on:_ E1-S2
  - _Delivers:_ `ResolvePort()`, missing systemCode → 21002

### backend Integration Layer

- [ ] **E3-S1** — Fixed-length transformer
  - _Depends on:_ E1-S2
  - _Delivers:_ `Transformer` interface, `ToTCP` (fixed-length serialization), `ToHTTP` (offset-based parsing), template engine

- [ ] **E3-S2** — backend client interface and mock
  - _Depends on:_ E1-S1
  - _Delivers:_ `cbs.Client` interface, `MockClient`, fixture loader from `mock/fixtures.yaml`

- [ ] **E3-S3** — TCP client stub
  - _Depends on:_ E3-S2
  - _Delivers:_ `TCPClient` struct compiling against `cbs.Client` interface (not wired to main)

### Response Code Mapper

- [ ] **E4-S1** — Standard code table and mapper
  - _Depends on:_ E1-S1
  - _Delivers:_ `codemap.Mapper`, full standard table (AA, BACKEND_ERR_001–BACKEND_ERR_013, SVC_ERR_001, fallback 20000)

- [ ] **E4-S2** — Spec-level response code overrides
  - _Depends on:_ E4-S1, E1-S2
  - _Delivers:_ override merge at load time, override wins over standard table

---

## Phase 3 — Assembly
> E5 depends on all of Phase 2. E6 depends on E1-S3 only — can run in parallel with E5.

- [ ] **E5-S1** — Handler pipeline wiring
  - _Depends on:_ E2-S2, E2-S3, E3-S1, E3-S2, E4-S1, E4-S2
  - _Delivers:_ full request pipeline (validate → resolve → transform → backend → map → respond), standard success/error response shape

- [ ] **E5-S2** — Startup wiring in `main.go`
  - _Depends on:_ E5-S1, E2-S1
  - _Delivers:_ `main.go` wires all components; `go run ./cmd/engine` produces a running engine

- [ ] **E6-S1** — Dashboard server and layout
  - _Depends on:_ E1-S3
  - _Delivers:_ `GET /dashboard` (version list), `GET /dashboard/<version>` (spec overview), embedded templates

- [ ] **E6-S2** — Endpoint, systems, and response code pages
  - _Depends on:_ E6-S1
  - _Delivers:_ endpoints list + detail, systems table, response codes table, all scoped to `/dashboard/<version>/`

---

## Phase 4 — PoC Validation
> Depends on all of Phase 3.

- [ ] **E7-S1** — Example spec and mock fixtures
  - _Depends on:_ E5-S2
  - _Delivers:_ `spec/v1/example.yaml` (full working spec), `mock/fixtures.yaml` (happy path + 2 error cases)

- [ ] **E7-S2** — End-to-end integration test
  - _Depends on:_ E7-S1, E6-S2
  - _Delivers:_ `go test ./...` passes; 5 HTTP test cases + dashboard smoke test cover all PoC success criteria

---

## Completion Criteria

The PoC is **done** when:
- [ ] `go test ./...` passes with zero failures
- [ ] `go run ./cmd/engine` starts with `spec/` directory and logs all loaded versions
- [ ] `POST /v1/example/query` with a valid body returns `{"code": 10000, ...}`
- [ ] `GET /dashboard` renders all loaded spec versions
- [ ] `GET /dashboard/v1/endpoints` renders the full endpoint spec including transforms

---

## Reference

| Document | Path |
|----------|------|
| Product Brief | `_bmad-output/planning-artifacts/product-brief.md` |
| Architecture | `_bmad-output/planning-artifacts/architecture.md` |
| Epics & Stories | `_bmad-output/planning-artifacts/epics-and-stories.md` |
| Response Code Spec | [internal-docs] |
