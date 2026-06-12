# Product Brief — Service Engine

**Version:** 0.1-draft  
**Date:** 2026-06-13  
**Author:** Sirawat Ngarmphandisorn  
**Status:** Draft

---

## 1. Concept

A **spec-driven HTTP↔TCP bridge engine** written in Go. The engine reads a single custom YAML spec file, starts an HTTP web server with routes defined in that spec, and for each incoming request validates, transforms, and forwards it as a TCP socket message to the downstream TCP service. backend responses are mapped back to a standardized 5-digit response code scheme and returned as HTTP JSON responses.

The same spec file powers a **built-in dashboard** — a browsable UI served by the engine itself — giving QA, developers, and other teams a single authoritative source for all API definitions, response codes, and transformation rules.

---

## 2. Problem

| Pain | Impact |
|------|--------|
| Every API or config change requires a full SDLC cycle | Slow delivery; days or weeks to expose a new backend operation |
| No documentation of existing services | Teams implement without leaving a record; knowledge is tribal |
| No single source of truth for QA | Test cases written from memory or stale Confluence pages |
| Deployment is the bottleneck for API delivery | A spec change requires code change → review → deploy → done |

---

## 3. Solution

The engine inverts the relationship between code and configuration:

```
Traditional:  Code defines behavior → deploy to change
Engine:       Spec defines behavior → reload to change
```

**Delivery flow:**
1. Engineer edits the YAML spec (adds/modifies an endpoint or transformation rule)
2. Commits to Git (audit trail via version control)
3. Engine restarts / hot-reloads — reads new spec version
4. New API behavior is live immediately

No Go code needs to be written or deployed for routine API changes.

---

## 4. Users & Stakeholders

| Who | How they interact with the engine |
|-----|-----------------------------------|
| **Calling services** | Consume the REST API exposed by the engine |
| **Backend engineers** | Author and version the spec file |
| **QA engineers** | Use the dashboard to read endpoint contracts and expected response codes |
| **Other teams / consumers** | Browse the dashboard to discover available APIs without hunting Confluence |
| **Auditors** | Inspect Git history + engine startup logs (spec version recorded on load) |

---

## 5. Key Concepts

### `systemCode` — Routing Key
Every request body must include a `systemCode` field. The engine looks up the registered TCP port for that code. If not found → returns `21002` (Route name not found).

### Custom Spec Format (not OpenAPI)
A purpose-built YAML format covering:
- **HTTP contract** — path, method, request/response schema
- **Systems registry** — `systemCode` → backend port mapping
- **Request transform** — HTTP fields → TCP message fields
- **Response transform** — TCP response fields → HTTP response body
- **Response code overrides** — per-service extensions to the standard 5-digit table

### Response Code Standard
5-digit codes with structured meaning:

| First digit | Type | HTTP status |
|-------------|------|-------------|
| 1 | Success | 200 |
| 2 | Business error (backend returned error) | 200 |
| 3 | Validation error | 400 |
| 4 | Internal server error | 500 |

backend codes (`BACKEND_ERR_001`, `SVC_ERR_001`, `AA`, etc.) are mapped to engine codes in the spec.

### Built-in Dashboard
The engine serves a web UI at a designated path (e.g. `/dashboard`). Content is derived entirely from the loaded spec. No separate documentation step required.

### Multi-domain Architecture
One engine instance per business domain. Each engine connects to the same TCP downstream but targets different TCP ports per `systemCode`.

### Audit Trail
- Spec versioned in Git (change history, PR reviews, approvals)
- `version` field in spec file identifies the active definition
- Engine logs active spec version and load timestamp on startup

---

## 6. Constraints

| Constraint | Detail |
|------------|--------|
| **Language** | Go |
| **Environment** | Regulated environment (audit trail mandatory) |
| **Downstream protocol** | TCP socket (backend) |
| **Spec format** | Custom YAML — not OpenAPI |
| **Deployment model** | Spec change requires at minimum an engine restart (not zero-downtime hot-swap in PoC) |

---

## 7. PoC Success Criteria

The PoC is successful when:

1. A single endpoint is defined entirely in the YAML spec (no Go code written for that endpoint)
2. The engine reads the spec and starts an HTTP server exposing that route
3. A request is sent → engine validates it → transforms it into a TCP message (correct field mapping)
4. A **service mock** receives the TCP message and returns a realistic response
5. Engine maps the backend response code → returns a clean JSON HTTP response
6. The dashboard renders the endpoint definition, request/response schema, and response code table

All 5 layers exercised. TCP connection is mocked — the transformation correctness and end-to-end usability are what is being validated, not real backend integration.

---

## 8. Out of Scope (PoC)

- Real TCP connection (replaced by a mock that validates transformation)
- Hot-reload without restart
- Authentication / mTLS on HTTP layer
- Multiple concurrent endpoints
- Dashboard write operations (read-only in PoC)
- Full backend response code coverage (only codes exercised by the PoC endpoint)
