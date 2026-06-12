# Epics & Stories — Service Engine

**Version:** 1.1  
**Date:** 2026-06-13  
**Status:** Ready for sprint planning

---

## Overview

7 epics covering the full PoC scope. Epics 1–6 build the engine components. Epic 7 wires them into the proven end-to-end slice.

API versioning is threaded through E1 (spec model), E2 (routing), and E6 (dashboard) — not a separate epic.

| Epic | Name | Stories |
|------|------|---------|
| E1 | Engine Foundation | 3 |
| E2 | HTTP Runtime | 3 |
| E3 | backend Integration Layer | 3 |
| E4 | Response Code Mapper | 2 |
| E5 | Request Handler Pipeline | 2 |
| E6 | Dashboard | 2 |
| E7 | PoC End-to-End Validation | 2 |

---

## Epic 1 — Engine Foundation

> Establish the Go project structure, spec YAML model, and loader. All other epics depend on this.

---

### E1-S1: Project scaffold

**As a** developer,  
**I want** a Go project with the defined folder structure, `go.mod`, and a runnable `main.go` skeleton,  
**so that** all subsequent stories have a consistent home and the binary can be compiled from day one.

**Acceptance Criteria:**
- `go.mod` created with module name `engine-poc`
- Directory structure matches architecture doc (`cmd/engine`, `internal/spec`, `internal/server`, `internal/handler`, `internal/transformer`, `internal/cbs`, `internal/codemap`, `internal/dashboard`)
- `cmd/engine/main.go` compiles and exits cleanly with a startup log line
- `chi` added as the only external dependency
- `go build ./...` passes with zero errors

---

### E1-S2: Spec loader

**As a** developer,  
**I want** a `spec.Load(path string) (*Spec, error)` function that parses and validates the YAML spec,  
**so that** all other components receive a typed, validated in-memory `Spec` rather than raw YAML.

**Acceptance Criteria:**
- Parses `version`, `name`, `description`, `systems`, `response_codes`, `endpoints` from YAML
- `version` is a required string field (e.g. `"1.0.0"`) — missing version is a load error
- Returns a descriptive error (not a panic) if required fields are missing or types are wrong
- `Spec` struct is fully typed (no `map[string]interface{}` escape hatches)
- Logs `name`, `version`, and UTC load timestamp on successful load
- Unit tests cover: valid spec loads correctly, missing required field returns error, missing `version` returns error, unknown field is ignored
- Example spec file at `spec/v1/example.yaml` loads without error

---

### E1-S3: Multi-version spec registry

**As a** developer,  
**I want** the engine to load all spec files from a versioned directory structure and register each version independently,  
**so that** old API versions remain accessible when a new spec version is deployed alongside them.

**Acceptance Criteria:**
- Spec files live under `spec/<version>/` directories (e.g. `spec/v1/`, `spec/v2/`)
- Engine scans the directory given by env var `ENGINE_SPEC_DIR` (default: `spec/`) at startup
- Each subdirectory that contains a `*.yaml` file is treated as one spec version
- All discovered specs are loaded; if any one fails validation the engine exits with a clear error naming the failing file
- `SpecRegistry` type holds `map[string]*Spec` keyed by the directory name (e.g. `"v1"`, `"v2"`)
- Startup log lists all loaded versions: `loaded spec versions: [v1 v2]`
- Unit tests: single version loads, two versions load independently, one bad file causes registry load failure

---

## Epic 2 — HTTP Runtime

> Start an HTTP server with routes registered dynamically from the spec. Validate incoming requests.

---

### E2-S1: HTTP server with dynamic route registration

**As a** calling service,  
**I want** the engine to start an HTTP server with routes registered per spec version,  
**so that** adding an endpoint to any spec version is the only step required to expose that versioned route.

**Acceptance Criteria:**
- Server starts on a configurable port (default `8080`, via env var `ENGINE_PORT`)
- Routes registered at startup from all versions in `SpecRegistry`, prefixed by version directory name
  - e.g. spec `v1` endpoint `POST /example/query` → registered as `POST /v1/example/query`
  - e.g. spec `v2` endpoint `POST /example/query` → registered as `POST /v2/example/query`
- `chi` middleware stack applied: request ID header injection, structured request log (method, path, status, latency), panic recovery
- `GET /health` always responds `200 OK` regardless of spec content
- Server startup log includes bound address and all registered version prefixes
- Requesting a path not in any spec returns `404`
- Two versions can define the same logical endpoint path with different transforms — each is independent

---

### E2-S2: Request body validation

**As a** calling service,  
**I want** the engine to validate my request body against the spec's field definitions before doing anything else,  
**so that** I get a clear, fast error if my request is malformed or missing required fields.

**Acceptance Criteria:**
- Non-JSON body → `400` with engine code `31000` (input parse error)
- Missing required field → `400` with engine code `32000` (input validation error), body includes which field failed
- Extra fields not in spec are accepted silently (no strict mode in PoC)
- `systemCode` absent from root → `400` with engine code `32000`
- Unit tests cover all error paths above

---

### E2-S3: `systemCode` resolution

**As a** developer,  
**I want** the handler to resolve `systemCode` from the request body to a backend port before proceeding,  
**so that** unknown system codes are rejected early with the correct error code.

**Acceptance Criteria:**
- `systemCode` value found in `spec.Systems` → port number returned
- `systemCode` value not found in `spec.Systems` → `200` with engine code `21002` (route name not found)
- Resolution is a pure function: `ResolvePort(systemCode string, systems map[string]SystemDef) (int, error)`
- Unit tests: known code resolves, unknown code returns error

---

## Epic 3 — backend Integration Layer

> Define the backend client interface, implement fixed-length transformer, and build the mock client.

---

### E3-S1: Fixed-length transformer

**As a** developer,  
**I want** a transformer that serializes HTTP request fields into a fixed-length TCP message and deserializes a fixed-length backend response back into named fields,  
**so that** the transformation rules in the spec are the single source of truth for wire format.

**Acceptance Criteria:**
- `ToTCP(req Request, rules []FieldRule) (CBSMessage, error)`:
  - Evaluates `{{request.body.<field>}}` templates against request body
  - Pads/truncates each field to exact `length` using `align` and `pad` from rule
  - Returns fields in declaration order as a byte slice
  - Returns error if a referenced field is absent from the request body
- `ToHTTP(cbs BackendResponse, rules ResponseRules) (map[string]any, error)`:
  - Parses backend byte slice by cumulative offset using `response.fields`
  - Trims padding from each parsed value
  - Evaluates `{{cbs.<field>}}` templates in `response.mapping` to build HTTP body map
- `ResponseCode` is extracted from backend response before `ToHTTP` is called (responsibility of handler, not transformer)
- Unit tests cover: correct serialization, left/right alignment, pad character, truncation, template substitution, offset-based parsing

---

### E3-S2: backend client interface and mock

**As a** developer,  
**I want** a `cbs.Client` interface and a `MockClient` implementation driven by a fixture file,  
**so that** the PoC can run end-to-end without a real TCP connection, and the mock is controlled independently of the spec.

**Acceptance Criteria:**
- `Client` interface: `Send(port int, msg CBSMessage) (BackendResponse, error)`
- `CBSMessage` = `map[string]string` (field name → string value, pre-serialization)
- `BackendResponse` = `[]byte` (raw fixed-length response bytes)
- `MockClient` loaded from `mock/fixtures.yaml`
- Fixture matching: first fixture where all `match` key-values match the incoming message fields wins
- No matching fixture → returns error (unmatched request is a test setup problem, not a business error)
- `mock/fixtures.yaml` includes at least: success case (`ResponseCode: "OK"`) and one business error case (`BACKEND_ERR_001`)
- Unit tests: match found, no match returns error, first-match-wins ordering

---

### E3-S3: TCP client (stub)

**As a** developer,  
**I want** a `TCPClient` struct that implements `cbs.Client` with real TCP dial and fixed-length read/write,  
**so that** the interface contract is verified against a real implementation even if it is not used in the PoC runtime.

**Acceptance Criteria:**
- `TCPClient.Send` dials `localhost:<port>`, writes serialized message bytes, reads response bytes, closes connection
- Compiles and passes `go vet`
- Not wired into `main.go` for PoC (MockClient is used) — but exists and is reachable
- No unit test required for PoC (needs real backend); compile-time correctness is sufficient

---

## Epic 4 — Response Code Mapper

> Map backend `ResponseCode` values to the 5-digit engine code scheme.

---

### E4-S1: Standard code table and mapper

**As a** developer,  
**I want** a `codemap.Mapper` that maps a backend `ResponseCode` string to a 5-digit `EngineCode`,  
**so that** the response code contract is enforced consistently across all endpoints.

**Acceptance Criteria:**
- Standard table hardcoded in `internal/codemap/standard.go` with all codes from the Response Code Specification doc:
  - `AA` → `10000` (HTTP 200)
  - `BACKEND_ERR_001`–`BACKEND_ERR_013` → `21001`–`21013` (HTTP 200)
  - `SVC_ERR_001` → `22001` (HTTP 200)
  - Unmapped → `20000` (HTTP 200, default business error)
- `Mapper.Map(cbsCode string) EngineCode` returns correct code for all entries
- Unknown backend code returns `20000` (not an error)
- Unit tests cover: every standard code, unknown code fallback

---

### E4-S2: Spec-level response code overrides

**As a** developer,  
**I want** the mapper to merge per-spec `response_codes` overrides on top of the standard table at startup,  
**so that** a spec can add new backend code mappings without modifying the standard table.

**Acceptance Criteria:**
- Overrides from `spec.ResponseCodes` are merged into mapper at load time
- Override wins over standard table for the same backend code
- Standard table is unchanged (overrides do not mutate it)
- Unit test: spec override takes precedence over standard entry for same backend code

---

## Epic 5 — Request Handler Pipeline

> Wire all components into the end-to-end request processing pipeline.

---

### E5-S1: Handler pipeline wiring

**As a** calling service,  
**I want** the engine to process a full request through validate → resolve → transform → backend → map codes → respond,  
**so that** a well-formed request produces the correct JSON response in a single HTTP call.

**Acceptance Criteria:**
- Handler is constructed with: `*Spec`, `Transformer`, `cbs.Client`, `*codemap.Mapper`
- Pipeline steps execute in order; any step failure short-circuits to an error response
- Successful response shape:
  ```json
  {
    "code": 10000,
    "message": "Success",
    "data": { ...mapped fields... }
  }
  ```
- Error response shape (all error types):
  ```json
  {
    "code": 32000,
    "message": "Input validation error",
    "data": null
  }
  ```
- Business error (backend returned non-success code) uses HTTP 200 with appropriate engine code
- Internal error (backend client error, transformer error) returns HTTP 500 with engine code `40000`
- Integration test using `MockClient`: happy path, missing field, unknown `systemCode`, backend business error

---

### E5-S2: Startup wiring in `main.go`

**As a** developer,  
**I want** `main.go` to load the spec, construct all components, and start the server,  
**so that** `go run ./cmd/engine` produces a running engine from the example spec.

**Acceptance Criteria:**
- `main.go` loads spec from path given by env var `ENGINE_SPEC` (default: `spec/example.yaml`)
- Constructs: `Loader → Spec → Mapper → Transformer → MockClient → Handler → Server`
- If spec load fails, process exits with code `1` and logs the error
- `go run ./cmd/engine` starts and logs: spec name, version, bound address
- `curl -X POST localhost:8080/example/query -d '{"systemCode":"SERVICE_A","resourceId":"1234567890"}'` returns `{"code":10000,...}`

---

## Epic 6 — Dashboard

> Serve a read-only web UI from embedded templates, driven entirely by the loaded spec.

---

### E6-S1: Dashboard server and layout

**As a** QA engineer or team member,  
**I want** to open `/dashboard` in a browser and see all loaded spec versions with navigation into each,  
**so that** I have a single URL to understand the full API surface — current and historical versions — without reading YAML.

**Acceptance Criteria:**
- `GET /dashboard` → HTML page listing all loaded spec versions as cards/rows (version key, spec name, spec `version` field, description)
- Each version card links to `/dashboard/<version>` (e.g. `/dashboard/v1`)
- `GET /dashboard/<version>` → overview page for that spec: name, version, description, active-since timestamp, navigation to Endpoints / Systems / Response Codes
- Templates embedded via `//go:embed` — no files read at request time
- Renders correctly with one version and with two versions
- No external CSS/JS CDN dependencies (self-contained HTML)

---

### E6-S2: Endpoint, systems, and response code pages

**As a** QA engineer,  
**I want** to browse all endpoints within a spec version with their full detail — request schema, transformation rules, and expected response codes —  
**so that** I can write tests directly from the dashboard without needing to find the YAML file.

**Acceptance Criteria:**
- `GET /dashboard/<version>/endpoints` → list of all endpoints (path, method, summary)
- `GET /dashboard/<version>/endpoints?path=/example/query&method=POST` → detail page showing:
  - Request fields (name, type, required, description)
  - Request transform rules (backend field, length, value/template, align, pad)
  - Response transform fields (backend field, length) and mapping (backend field → response key)
  - Response codes applicable to this endpoint (merged global + endpoint-specific)
- `GET /dashboard/<version>/systems` → table of systemCode → port → description
- `GET /dashboard/<version>/codes` → full response code table (code, type, HTTP status, description, backend code)
- All pages render data from the in-memory `SpecRegistry` (no file reads at request time)
- Navigating to a non-existent version returns a clear `404` HTML page

---

## Epic 7 — PoC End-to-End Validation

> Produce a working example spec and verify the complete flow manually and with an integration test.

---

### E7-S1: Example spec and mock fixtures

**As a** developer,  
**I want** a complete `spec/example.yaml` with one endpoint and a matching `mock/fixtures.yaml`,  
**so that** anyone can clone the repo, run the engine, and immediately see it working.

**Acceptance Criteria:**
- `spec/v1/example.yaml` defines:
  - `version`, `name`, `description`
  - One system: `SERVICE_A` → port `9001`
  - One endpoint: `POST /example/query` with `systemCode` + `resourceId` request fields
  - Full request transform (at least 2 backend fields)
  - Full response transform (parse `ResponseCode`, `FIELD_A`, `FIELD_B`; map to `balance`, `currency`)
- Engine registers route as `POST /v1/example/query`
- `mock/fixtures.yaml` defines:
  - Happy path: `systemCode=SERVICE_A` + any `RESOURCE_ID` → `ResponseCode=OK`, `FIELD_A`, `FIELD_B`
  - Business error: `RESOURCE_ID=ERROR_ACCOUNT` → `ResponseCode=BACKEND_ERR_001`
  - Timeout simulation: `RESOURCE_ID=TIMEOUT_ACCOUNT` → `ResponseCode=BACKEND_ERR_003`
- `spec.Load("spec/example.yaml")` succeeds with no errors

---

### E7-S2: End-to-end integration test

**As a** developer,  
**I want** an integration test that starts the full engine (with MockClient) and exercises the complete request pipeline via HTTP,  
**so that** the PoC success criteria are provably met in code, not just manually.

**Acceptance Criteria:**
- Test starts engine with `spec/v1/example.yaml` (via `ENGINE_SPEC_DIR=spec/`) and `mock/fixtures.yaml` on a random port
- Test cases (each verified by HTTP response code + JSON body):
  1. Happy path: `POST /v1/example/query` valid request → `{"code": 10000, "data": {"balance": "...", "currency": "THB"}}`
  2. Missing `resourceId` → `{"code": 32000}`, HTTP 400
  3. Unknown `systemCode` → `{"code": 21002}`, HTTP 200
  4. backend business error (`RESOURCE_ID=ERROR_ACCOUNT`) → `{"code": 21001}`, HTTP 200
  5. backend timeout error (`RESOURCE_ID=TIMEOUT_ACCOUNT`) → `{"code": 21003}`, HTTP 200
- Dashboard smoke test: `GET /dashboard` returns HTTP 200 with engine name in body
- All tests pass with `go test ./...`
