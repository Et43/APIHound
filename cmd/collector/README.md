# APIHound Collector

A standalone CLI tool that ingests OpenAPI/Swagger specification files into BloodHound, with support for organising API collections under Operational/Business Units (OpCos).

## Overview

The APIHound Collector reads a **manifest file** that maps OpenAPI/Swagger specs to OpCos, then uploads them to a BloodHound server via the file-upload API. Each spec file is associated with an OpCo, which creates graph relationships that enable ownership-based analysis queries.

## Build

```bash
go build -o apihound-collector ./cmd/collector
```

## Usage

### Collect (upload specs)

```bash
apihound-collector collect \
  --manifest manifest.yaml \
  --server https://bloodhound:8080 \
  --token <api-bearer-token>
```

### Dry Run (validate without uploading)

```bash
apihound-collector collect \
  --manifest manifest.yaml \
  --dry-run
```

### Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--manifest` | Yes | Path to the manifest file (YAML or JSON) |
| `--server` | Yes* | BloodHound server URL | 
| `--token` | Yes* | BloodHound API bearer token |
| `--dry-run` | No | Parse and validate without uploading |
| `--verbose` | No | Enable verbose output |

\* Not required when using `--dry-run`.

## Manifest Format

The manifest maps API specification files to Operational/Business Units (OpCos). It supports both YAML and JSON formats.

### YAML

```yaml
opcos:
  - name: "Payments"
    description: "Payment processing business unit"
    specs:
      - path: "./specs/payments-api.yaml"
      - path: "./specs/checkout-api.json"
        service_name: "checkout-service"

  - name: "Identity"
    description: "Identity and access management"
    specs:
      - path: "./specs/identity-api.yaml"
```

### JSON

```json
{
  "opcos": [
    {
      "name": "Payments",
      "description": "Payment processing business unit",
      "specs": [
        { "path": "./specs/payments-api.yaml" },
        { "path": "./specs/checkout-api.json", "service_name": "checkout-service" }
      ]
    }
  ]
}
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `opcos[].name` | Yes | Unique name for the operational/business unit |
| `opcos[].description` | No | Human-readable description of the OpCo |
| `opcos[].specs[].path` | Yes | Path to the OpenAPI/Swagger spec file (relative to the manifest) |
| `opcos[].specs[].service_name` | No | Override the service identifier (defaults to filename without extension) |

### Spec file paths

Spec paths are resolved relative to the manifest file's directory. For example, given this directory structure:

```
project/
├── manifest.yaml
└── specs/
    ├── payments-api.yaml
    └── identity-api.json
```

The manifest would reference `./specs/payments-api.yaml`.

## Graph Model

When specs are ingested with OpCo metadata, the collector creates:

- **APIOpCo node** — represents the operational/business unit
- **BelongsToOpCo edge** — links `APIService` → `APIOpCo`
- **opco_name property** — convenience property on the `APIService` node

The same OpCo name always produces the same graph node (idempotent), so multiple specs under the same OpCo share a single `APIOpCo` node.

## Example Queries

After ingesting specs with OpCo assignments, you can run queries like:

```cypher
-- All services owned by the Payments OpCo
MATCH (o:APIOpCo {name: "Payments"})<-[:BelongsToOpCo]-(s:APIService)
RETURN s

-- Unauthenticated endpoints by OpCo
MATCH (o:APIOpCo)<-[:BelongsToOpCo]-(s:APIService)-[:HasEndpoint]->(e:APIEndpoint)
WHERE NOT (e)-[:RequiresSecurity]->(:APISecurityScheme)
RETURN o.name, s.name, e.name, e.method, e.path

-- Services not assigned to any OpCo
MATCH (s:APIService)
WHERE NOT (s)-[:BelongsToOpCo]->(:APIOpCo)
RETURN s
```

## Authentication

The collector uses BloodHound API token authentication. Generate a token from the BloodHound UI under **Settings → API Tokens**, then pass it via the `--token` flag.
