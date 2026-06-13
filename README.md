# ⚙️ Service Engine

> One YAML spec. One binary. HTTP in, TCP out.

A spec-driven HTTP ↔ TCP bridge engine in Go. Define your API in YAML — the engine handles routing, request validation, fixed-length TCP transformation, response code mapping, and a built-in dashboard. No code changes needed for routine API updates.

---

## 🚀 Quick Start

```bash
# Build
go build ./cmd/engine

# Run (spec/v1/example.yaml + mock/fixtures.yaml loaded automatically)
./engine

# Try it
curl -X POST localhost:8080/v1/example/query \
  -H 'Content-Type: application/json' \
  -d '{"systemCode":"SERVICE_A","resourceId":"1234567890"}'
# → {"code":10000,"message":"Success","data":{"balance":"000000012345.67","currency":"THB"}}
```

---

## 🗂️ Project Structure

```
.
├── cmd/engine/          # Binary entry point
├── internal/
│   ├── spec/            # YAML loader & multi-version registry
│   ├── server/          # chi HTTP server, route registration
│   ├── handler/         # Request pipeline (validate → resolve → respond)
│   ├── transformer/     # Fixed-length TCP ↔ HTTP field mapping
│   ├── cbs/             # Backend client interface, mock, TCP stub
│   ├── codemap/         # Backend → 5-digit engine code mapper
│   └── dashboard/       # Embedded HTML dashboard
├── spec/v1/             # Versioned YAML spec files
└── mock/                # Fixture responses for mock client
```

---

## 🔁 Request Pipeline

```
HTTP Request
    │
    ▼
1. Parse & validate JSON body (required fields, systemCode)
    │
    ▼
2. Resolve systemCode → backend TCP port
    │
    ▼
3. Transform fields into fixed-length TCP message
    │
    ▼
4. Send to backend (MockClient in PoC, TCPClient in production)
    │
    ▼
5. Parse fixed-length response → extract ResponseCode
    │
    ▼
6. Map backend code → 5-digit engine code
    │
    ▼
HTTP JSON Response
```

---

## 📄 Spec Format

All behavior is defined in a single YAML file:

```yaml
version: "1.0.0"
name: "My Domain"

systems:
  SERVICE_A:
    port: 9001

endpoints:
  - path: /example/query
    method: POST
    request:
      fields:
        resourceId: { type: string, required: true }
    transform:
      request:
        - name: RESOURCE_ID
          length: 16
          value: "{{request.body.resourceId}}"
      response:
        fields:
          - { name: ResponseCode, length: 15 }
          - { name: FIELD_A, length: 15 }
        mapping:
          balance: "{{cbs.FIELD_A}}"
```

---

## 📊 Response Codes

| Code | Meaning |
|------|---------|
| `10000` | Success |
| `21001–21013` | Backend errors (`BACKEND_ERR_001–013`) |
| `22001` | No records found (`SVC_ERR_001`) |
| `31000` | Input parse error |
| `32000` | Input validation error |
| `40000` | Internal server error |
| `20000` | Unmapped backend code (fallback) |

---

## 🖥️ Dashboard

Served at `/dashboard` — no external dependencies, pure HTML.

| URL | Shows |
|-----|-------|
| `/dashboard` | All loaded spec versions |
| `/dashboard/v1` | Overview: name, description, nav links |
| `/dashboard/v1/endpoints` | Endpoint list + detail (request fields, transforms) |
| `/dashboard/v1/systems` | systemCode → port table |
| `/dashboard/v1/codes` | Full response code table |

---

## ⚙️ Configuration

| Env Var | Default | Purpose |
|---------|---------|---------|
| `ENGINE_SPEC_DIR` | `spec/` | Directory of versioned spec subdirectories |
| `ENGINE_MOCK_FIXTURES` | `mock/fixtures.yaml` | Mock backend responses |
| `ENGINE_PORT` | `8080` | HTTP listen port |

---

## 🧪 Tests

```bash
go test ./...   # all unit + integration tests
go vet ./...    # static analysis
```

---

## 🔌 Adding a Real Backend

Replace `MockClient` with `TCPClient` in `cmd/engine/main.go`. The `cbs.Client` interface is the only boundary — no other code changes needed.
