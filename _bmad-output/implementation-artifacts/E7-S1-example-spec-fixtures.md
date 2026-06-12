# Story: E7-S1 — Example Spec and Mock Fixtures

**Epic:** E7 — PoC End-to-End Validation  
**Status:** Ready for Development  
**Depends on:** E5-S2  
**Blocks:** E7-S2

---

## Summary

Produce the complete `spec/v1/example.yaml` and `mock/fixtures.yaml` that prove the full engine concept. These are the canonical inputs for the PoC — anyone cloning the repo runs the engine against these and immediately sees it working.

---

## Acceptance Criteria

- [ ] `spec/v1/example.yaml` is a valid spec with all required fields populated
- [ ] Defines `SERVICE_A` → port `9001`
- [ ] Defines `POST /example/query` with `systemCode` + `resourceId` fields
- [ ] Full request transform (at least 2 backend fields with correct fixed-length rules)
- [ ] Full response transform (parses `ResponseCode`, `FIELD_A`, `FIELD_B`; maps to `balance`, `currency`)
- [ ] `mock/fixtures.yaml` covers: happy path, business error (`BACKEND_ERR_001`), timeout (`BACKEND_ERR_003`)
- [ ] `spec.Load("spec/v1/example.yaml")` succeeds with no errors

---

## `spec/v1/example.yaml`

```yaml
version: "1.0.0"
name: "Service Engine - Example Domain"
description: "Resource query operations for the PoC. Demonstrates the engine concept: one YAML spec drives the HTTP server, TCP transformation, and dashboard."

systems:
  SERVICE_A:
    port: 9001
    description: "backend service TCP port"

response_codes:
  22001:
    type: business_error
    http_status: 200
    description: "No records found"
    backend_code: "SVC_ERR_001"

endpoints:
  - path: /example/query
    method: POST
    summary: "Inquire resource value"
    description: "Returns the current balance and currency for a given resource identifier. The request is forwarded to backend via TCP socket on the port registered for the provided systemCode."

    request:
      fields:
        systemCode:
          type: string
          required: true
          description: "Identifies the backend service to call. Must be registered in the systems section."
        resourceId:
          type: string
          required: true
          description: "The resource identifier to query. Must be a valid backend resource identifier."

    transform:
      request:
        - name: MSG_TYPE
          length: 4
          value: "0200"
          align: left
          pad: " "
        - name: RESOURCE_ID
          length: 16
          value: "{{request.body.resourceId}}"
          align: left
          pad: " "

      response:
        fields:
          - name: ResponseCode
            length: 8
          - name: FIELD_A
            length: 15
          - name: FIELD_B
            length: 3

        mapping:
          balance: "{{cbs.FIELD_A}}"
          currency: "{{cbs.FIELD_B}}"

    response_codes:
      - code: 22001
        description: "Resource not found in backend"
```

---

## `mock/fixtures.yaml`

```yaml
# Mock backend responses for the PoC.
# First matching fixture wins — more specific matches must come before general ones.

# Business error: specific account triggers backend internal error
- match:
    RESOURCE_ID: "ERROR_ACCOUNT"
  response:
    ResponseCode: "BACKEND_ERR_001"
    FIELD_A: "000000000000000"
    FIELD_B: "   "

# Business error: specific account simulates timeout
- match:
    RESOURCE_ID: "TIMEOUT_ACCOUNT"
  response:
    ResponseCode: "BACKEND_ERR_003"
    FIELD_A: "000000000000000"
    FIELD_B: "   "

# Business error: no records found (SVC_ERR)
- match:
    RESOURCE_ID: "NOTFOUND_ACCOUNT"
  response:
    ResponseCode: "SVC_ERR_001"
    FIELD_A: "000000000000000"
    FIELD_B: "   "

# Happy path: any other resource identifier returns success
- match:
    MSG_TYPE: "0200"
  response:
    ResponseCode: "OK"
    FIELD_A: "000000012345.67"
    FIELD_B: "THB"
```

---

## Verification

```bash
# Spec loads cleanly
go run ./cmd/engine   # should start without errors and log "loaded spec versions: [v1]"

# Happy path
curl -X POST localhost:8080/v1/example/query \
  -H 'Content-Type: application/json' \
  -d '{"systemCode":"SERVICE_A","resourceId":"1234567890"}'
# → {"code":10000,"message":"Success","data":{"balance":"12345.67","currency":"THB"}}

# backend internal error
curl -X POST localhost:8080/v1/example/query \
  -H 'Content-Type: application/json' \
  -d '{"systemCode":"SERVICE_A","resourceId":"ERROR_ACCOUNT"}'
# → {"code":21001,"message":"Backend internal error","data":null}

# Timeout
curl -X POST localhost:8080/v1/example/query \
  -H 'Content-Type: application/json' \
  -d '{"systemCode":"SERVICE_A","resourceId":"TIMEOUT_ACCOUNT"}'
# → {"code":21003,"message":"Transaction timeout","data":null}
```
