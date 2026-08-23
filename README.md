# PulseGrid

PulseGrid is a standalone event-driven operations platform written in Go.
It models organizations, contacts, audiences, campaigns, templates,
automation workflows, message deliveries, and event analytics.

The project intentionally uses only the Go standard library. Data is kept in
memory so the service can run as a compact development fixture without a
database or external broker.

## Run

```bash
go run ./cmd/pulsegridd
```

The server listens on `:8091`.

## API examples

```bash
curl http://localhost:8091/healthz

curl -X POST http://localhost:8091/v1/organizations \
  -H 'content-type: application/json' \
  -d '{"name":"Acme","owner":"alice"}'

curl -X POST http://localhost:8091/v1/contacts \
  -H 'content-type: application/json' \
  -d '{"organization_id":"<org-id>","email":"person@example.com","name":"Person"}'
```

## Packages

- `internal/domain`: business entities and lifecycle rules
- `internal/store`: concurrency-safe in-memory repository
- `internal/events`: typed event bus
- `internal/jobs`: bounded asynchronous workers
- `internal/analytics`: event aggregation and reports
- `internal/service`: application orchestration
- `internal/httpapi`: JSON HTTP transport
