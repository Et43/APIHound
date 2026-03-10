# Plan: Refactor BloodHound → APIHound

**TL;DR**: Strip all AD/Azure domain logic (~40% of codebase) and replace it with an API environment mapping domain. Keep the robust infrastructure — auth, Neo4j graph DB via dawgs, ingest pipeline, Sigma.js graph rendering, datapipe daemon, PostgreSQL via GORM. The new domain models 6 node types (API Endpoint, Service, API Gateway, Business Unit, Auth Provider, Environment) and 5 edge types (Calls, OwnedBy, RoutesThrough, ExposesData, GovernedBy). Data enters via OpenAPI spec import, API gateway exports, and manual upload. First working milestone: ingest an OpenAPI spec → see a graph.

Work is split into 6 sequential phases. Each phase produces a testable increment.

---

## Phase 1 — Branding & Scaffolding (Foundation)

1. Rename project references from BloodHound/BHCE to APIHound across:
   - `cmd/api/src/cmd/` (CLI entrypoint, version strings)
   - `cmd/ui/src/App.tsx` and `cmd/ui/public/manifest.json`
   - `dockerfiles/` and docker-compose configs
   - `debian/` packaging
   - `go.mod`, `go.work` module paths
   - Config files: `local-harnesses/build.config.json`, env templates

2. Delete all AD/Azure domain code (bulk removal — don't replace yet):
   - **CUE schemas**: `packages/cue/bh/ad/`, `packages/cue/bh/azure/` — empty the domain definitions
   - **Generated Go schemas**: `packages/go/graphschema/ad/`, `packages/go/graphschema/azure/`
   - **Ingest converters**: `packages/go/ein/ad.go`, `packages/go/ein/azure.go`, `packages/go/ein/incoming_models.go` (AD-specific types)
   - **Analysis**: `packages/go/analysis/ad/`, `packages/go/analysis/azure/`, `packages/go/analysis/hybrid/`, `cmd/api/src/analysis/ad/`, `cmd/api/src/analysis/azure/`
   - **API handlers**: `cmd/api/src/api/v2/ad_entity.go`, `cmd/api/src/api/v2/ad_related_entity.go`, `cmd/api/src/api/v2/azure.go`, AD/Azure routes in `cmd/api/src/api/registration/v2.go`
   - **Data quality models**: `cmd/api/src/model/adquality.go`, `cmd/api/src/model/azurequality.go`
   - **Frontend**: `packages/javascript/bh-shared-ui/src/graphSchema.ts` (all AD/Azure enums), `packages/javascript/bh-shared-ui/src/components/HelpTexts/` (~120 directories), `packages/javascript/bh-shared-ui/src/commonSearchesAGI.ts`, collector download page
   - **Valid edges**: `schemas/valid_edges.json`
   - Remove `github.com/bloodhoundad/azurehound` dependency from `go.mod`

3. Fix all compile errors from deletions — stub out interfaces/functions referenced from reusable code that called into deleted domain code (mainly in `cmd/api/src/daemons/datapipe/analysis.go`, `packages/go/graphschema/schema.go`, `packages/go/graphschema/primarykind.go`, `cmd/api/src/queries/graph.go`)

**Exit criteria**: Project compiles, boots, serves UI with empty graph. Auth works. No AD/Azure references remain.

---

## Phase 2 — API Domain Schema (The Graph Model)

1. Define CUE schema for the new domain in a new `packages/cue/bh/api/api.cue`:

   **Node Kinds**:
   - `APIEndpoint` — method, path, summary, request/response schemas, deprecated flag, version
   - `Service` — name, description, base URL, protocol (REST/GraphQL/gRPC), version
   - `APIGateway` — name, provider (Kong/Apigee/AWS/Azure), region
   - `BusinessUnit` — name, country, division, cost center
   - `AuthProvider` — type (OAuth2/APIKey/JWT/mTLS), issuer URL, scopes
   - `Environment` — name (prod/staging/dev), region, cloud provider
   - `DataClassification` — label (PII/Financial/Public/Internal), sensitivity level
   - `Policy` — type (rate-limit/CORS/versioning/deprecation), rules

   **Relationship Kinds**:
   - `Calls` — source service → target endpoint (with frequency, latency metadata)
   - `OwnedBy` — service/endpoint → business unit
   - `RoutesThrough` — endpoint → gateway
   - `ExposesData` — endpoint → data classification
   - `GovernedBy` — endpoint/service → policy
   - `DeployedIn` — service → environment
   - `AuthenticatesVia` — endpoint → auth provider
   - `DependsOn` — service → service (computed from Calls aggregation)
   - `ConsumesThirdParty` — service → external service (third-party dependency tracking)
   - `Exposes` — service → endpoint (containment)

2. Run `just bh-graphify` (adapting `packages/go/graphify/main.go` for the new `api` prefix) to generate Go types into `packages/go/graphschema/api/api.go`

3. Update `packages/go/graphschema/schema.go`: replace `DefaultGraph()` to combine `api` + `common` schemas with appropriate indexes (on `Name`, `ObjectID`, `BasePath`, `Region`)

4. Create `schemas/valid_edges.json` for API domain — define all valid source→edge→target combinations

5. Create frontend schema: replace `packages/javascript/bh-shared-ui/src/graphSchema.ts` with `APINodeKind`, `APIRelationshipKind`, `APIKindProperties` enums plus display name functions

6. Create node icon mapping in `packages/javascript/bh-shared-ui/src/utils/icons.ts`: API Endpoint → `faPlug`, Service → `faCubes`, Gateway → `faNetworkWired`, Business Unit → `faBuilding`, Auth Provider → `faKey`, Environment → `faCloud`, DataClassification → `faShieldAlt`, Policy → `faGavel`

**Exit criteria**: `just generate` succeeds. Schema compiles. Icons render for each node type.

---

## Phase 3 — OpenAPI Spec Ingest (First Data Source)

1. Add `github.com/getkin/kin-openapi` dependency (or `github.com/pb33f/libopenapi`) for OpenAPI 2.0/3.0/3.1 parsing

2. Create `packages/go/ein/api.go` — converter functions:
   - `ConvertOpenAPISpec(doc) → ([]IngestibleNode, []IngestibleRelationship)` — walks the parsed OpenAPI doc:
     - Each `info` block → `Service` node
     - Each path+method → `APIEndpoint` node (properties: method, path, summary, parameters, request/response schema refs, deprecated)
     - `Service` → `Exposes` → each `APIEndpoint`
     - Security schemes → `AuthProvider` nodes
     - Each endpoint's security requirements → `AuthenticatesVia` edges
     - Schema refs containing known PII patterns (email, SSN, phone, address) → `DataClassification` nodes + `ExposesData` edges

3. Create `packages/go/ein/api_incoming_models.go` — define `APISpecUpload` struct (spec content, metadata: source gateway, business unit, environment tags)

4. Update `cmd/api/src/services/graphify/ingest.go` — add routing for new `DataTypeOpenAPISpec` to the OpenAPI converter

5. Create API endpoint `POST /api/v2/ingest/openapi` in `cmd/api/src/api/registration/v2.go` — accepts OpenAPI YAML/JSON upload, creates ingest job, queues for processing

6. Support batch import: `POST /api/v2/ingest/openapi/bulk` — accepts ZIP of multiple specs with a manifest mapping spec→business unit→environment

7. Update the Upload page in the UI (`cmd/ui/src/views/Administration/`) — replace SharpHound/AzureHound file upload with OpenAPI spec upload (drag-and-drop YAML/JSON/ZIP)

**Exit criteria**: Upload an OpenAPI spec → nodes and edges appear in Neo4j → visible in the graph explorer.

---

## Phase 4 — Visualization & Entity Info (See the Graph)

1. Update `packages/javascript/bh-shared-ui/src/utils/content.ts` — replace `allSections` with API-specific entity panels:
   - **APIEndpoint**: Object Info (method, path, parameters, schemas), Outbound (Calls), Inbound (Called By), Auth (AuthenticatesVia), Data (ExposesData), Policies (GovernedBy)
   - **Service**: Object Info (base URL, protocol, version), Endpoints (Exposes), Dependencies (Calls out), Dependents (Called by), Owner (OwnedBy), Environment (DeployedIn), Third-Party deps
   - **APIGateway**: Routed endpoints (RoutesThrough), Policies applied
   - **BusinessUnit**: Owned services, country/division info
   - **AuthProvider**: Endpoints using this provider, scopes, type
   - **Environment**: Deployed services, region info

2. Create HelpTexts for each relationship type in `packages/javascript/bh-shared-ui/src/components/HelpTexts/` — description of what the relationship means, implications, remediation advice (e.g., `Calls/` → "This service depends on the target endpoint. If the target is unavailable, this service may be impacted.")

3. Update API handlers — create entity-specific endpoints in `cmd/api/src/api/v2/`:
   - `api_endpoint.go` — `GET /api/v2/endpoints/{id}`, related entity queries
   - `api_service.go` — `GET /api/v2/services/{id}`, dependency graph
   - `api_gateway.go` — `GET /api/v2/gateways/{id}`
   - `api_businessunit.go` — `GET /api/v2/business-units/{id}`
   - Register all routes in `cmd/api/src/api/registration/v2.go`

4. Update `cmd/ui/src/views/Explore/utils.ts` — update `initGraph()` to handle new node kinds with appropriate colors/icons. Remove `isTierZero` glyph logic, replace with data sensitivity glyphs (e.g., red shield for PII-exposing endpoints)

5. Create prebuilt searches in a new `commonSearchesAPI.ts`:
   - "All Services" — `MATCH (s:Service) RETURN s`
   - "All External-Facing Endpoints" — `MATCH (e:APIEndpoint)-[:RoutesThrough]->(g:APIGateway) RETURN e, g`
   - "Endpoints Exposing PII" — `MATCH (e:APIEndpoint)-[:ExposesData]->(d:DataClassification {label: 'PII'}) RETURN e, d`
   - "Services Without Owners" — `MATCH (s:Service) WHERE NOT (s)-[:OwnedBy]->() RETURN s`
   - "Cross-Region Dependencies" — `MATCH (s1:Service)-[:DeployedIn]->(e1:Environment), (s1)-[:Calls]->(:APIEndpoint)<-[:Exposes]-(s2:Service)-[:DeployedIn]->(e2:Environment) WHERE e1.region <> e2.region RETURN s1, s2, e1, e2`
   - "Third-Party Dependencies" — `MATCH (s:Service)-[:ConsumesThirdParty]->(tp:Service) RETURN s, tp`
   - "Unauthenticated Endpoints" — `MATCH (e:APIEndpoint) WHERE NOT (e)-[:AuthenticatesVia]->() RETURN e`

6. Update environment selector (`cmd/api/src/model/search.go`) — replace AD domain/Azure tenant concept with API environment/business unit scoping

**Exit criteria**: Upload OpenAPI spec → explore graph visually → click nodes to see entity info → run prebuilt Cypher queries.

---

## Phase 5 — Analysis Engine (Derive Insights)

1. Create `packages/go/analysis/api/` and `cmd/api/src/analysis/api/post.go` — the post-processing pipeline (called from `cmd/api/src/daemons/datapipe/analysis.go`):

   **Computed analyses**:
   - **Dependency graph enrichment**: Aggregate endpoint-level `Calls` edges into service-level `DependsOn` edges
   - **Blast radius computation**: For each service, calculate how many downstream services are affected if it goes down (transitive dependency count). Store as `blastRadius` property
   - **API sprawl detection**: Flag services with >N endpoints, duplicate endpoints across services (same path+method pattern), deprecated-but-still-called endpoints. Create `SprawlFinding` nodes
   - **Compliance gap detection**: Flag services without owners, endpoints without auth, endpoints exposing PII without rate limiting. Create `ComplianceFinding` nodes
   - **Cross-boundary flow analysis**: Detect data flows crossing region or business unit boundaries. Create `CrossBoundaryFlow` transit edges
   - **Third-party risk scoring**: Count and classify third-party dependencies per service, compute risk concentration
   - **Orphan detection**: Find endpoints not called by any service, services with no endpoints (zombie APIs)

2. Create data quality model — replace `ADDataQualityStat`/`AzureDataQualityStat` with `APIDataQualityStat` in a new `cmd/api/src/model/apiquality.go`:
   - Total services, endpoints, gateways, auth providers
   - Completeness metrics: % with owners, % with auth, % with data classification
   - Freshness: last ingest date per environment

3. Create Data Quality UI — replace AD/Azure stats in `cmd/ui/src/views/DataQuality/` with API environment coverage dashboards

4. Update `cmd/api/src/daemons/datapipe/analysis.go` — wire `api.Post()` into the analysis pipeline, replacing `ad.Post()` and `azure.Post()`

5. Add analysis-driven prebuilt searches:
   - "Highest Blast Radius Services" — `MATCH (s:Service) RETURN s ORDER BY s.blastRadius DESC LIMIT 20`
   - "Compliance Findings" — `MATCH (f:ComplianceFinding)-[:AffectsService]->(s:Service) RETURN f, s`
   - "Shadow APIs" — endpoints with no documentation, no owner, but receiving calls

**Exit criteria**: After ingest, analysis runs automatically. Findings appear in graph. Data quality dashboard shows coverage metrics.

---

## Phase 6 — Additional Data Sources & Polish

1. **API Gateway Import** — `POST /api/v2/ingest/gateway`:
   - Kong: parse `kong.yml` declarative config → services, routes, plugins (rate-limit, auth)
   - AWS API Gateway: parse exported Swagger+extensions → stages, authorizers, usage plans
   - Azure APIM: parse ARM export → APIs, operations, policies, subscriptions
   - Each gateway import creates `APIGateway` node + `RoutesThrough` edges + `Policy` nodes from gateway policies

2. **Manual/CSV upload** — `POST /api/v2/ingest/csv`:
   - Template CSV formats for services, endpoints, ownership mappings, environment mappings
   - UI page with template download + upload

3. **Reporting dashboard** — new view at `/reports`:
   - Service dependency heat map
   - API sprawl timeline (endpoints over time)
   - Compliance scorecard per business unit
   - Third-party dependency report
   - Cross-boundary data flow summary

4. **Asset Group adaptation** — repurpose BloodHound's "Tier Zero" asset group concept for "Critical APIs" — tag services/endpoints that are business-critical, show impact paths to them

5. **OpenAPI doc generation** — update `cmd/api/src/api/` OpenAPI specs to document all new APIHound endpoints. Run `just gen-spec`.

6. Update all tests, run `just prepare-for-codereview`

**Exit criteria**: All three data sources work. Full test suite passes. Documentation updated.

---

## Verification Checklist

- **Phase 1**: `go build ./...` succeeds, `docker compose up` boots, login works, empty graph renders
- **Phase 2**: `just generate` succeeds, `go test ./packages/go/graphschema/...` passes
- **Phase 3**: Upload petstore.yaml → nodes appear in Neo4j (`MATCH (n) RETURN n LIMIT 50`)
- **Phase 4**: Click endpoint node → see method/path/auth info. Prebuilt queries return results
- **Phase 5**: After ingest + analysis cycle, `MATCH (s:Service) RETURN s.blastRadius` returns values
- **Phase 6**: `just prepare-for-codereview` passes clean

---

## Key Decisions

- **Full replacement**: AD/Azure code completely removed, not kept as optional module
- **Neo4j retained**: Reuse dawgs library and Cypher query engine as-is
- **CUE pipeline**: Reuse existing CUE → Go generation for new schema (no custom schema format)
- **OpenAPI first**: First milestone is spec import, gateway/CSV import comes later
- **8 node types** (added DataClassification + Policy beyond the original 6 to support data risk and governance edges)
- **10 edge types** (added DeployedIn, AuthenticatesVia, ConsumesThirdParty, DependsOn, Exposes for completeness)

---

## Reuse Summary

### Fully Reusable (~60% of codebase)
- Auth system (users, roles, permissions, SSO, tokens, sessions)
- Ingest pipeline framework (jobs, tasks, file upload, JSON/ZIP decoding, `IngestibleNode`/`IngestibleRelationship` primitives)
- OpenGraph extension system (dynamic schema registration, custom nodes)
- Database layer (PostgreSQL via GORM, migration framework)
- Graph database abstraction (`dawgs` library)
- Datapipe daemon lifecycle (prune→delete→ingest→analyze loop)
- Cypher query execution engine
- API framework (router, middleware, error handling, pagination, filtering)
- Audit logging, saved queries, app config, feature flags
- Asset group concept (tag-based node grouping)
- CUE→Go code generation pipeline
- `UnifiedGraph`/`UnifiedNode`/`UnifiedEdge` rendering model
- Sigma.js graph rendering (SigmaChart, WebGL programs, layouts)
- Entity info panel structure, search UI, graph controls

### Needs Replacement (~40% of codebase)
- Graph schema definitions (all AD/Azure node kinds, relationship kinds, properties)
- Ingest domain types (`ein/ad.go`, `ein/azure.go`, `ein/incoming_models.go`)
- Ingest converters (AD/Azure-specific graphify converters)
- Analysis/post-processing (all of `analysis/ad/`, `analysis/azure/`, `analysis/hybrid/`)
- Entity-specific API endpoints (`/api/v2/computers/*`, `/api/v2/domains/*`, `/api/v2/azure/*`, etc.)
- Data quality models (`ADDataQualityStat`, `AzureDataQualityStat`)
- Ingest data types (`model/ingest/ingest.go` DataType enum)
- Valid edges schema (`schemas/valid_edges.json`)
- Frontend graph schema (`graphSchema.ts`), icons mapping, HelpTexts (~120 directories), prebuilt Cypher queries, content.ts entity display config
- Collector download page, data quality views

---

## Appendix A — Complete Deletion Inventory

> Generated 2026-03-10. Every file and directory that must be deleted or gutted for the AD/Azure removal.

### A1. CUE Schemas (DELETE entire directories)

| Path | Description |
|------|-------------|
| `packages/cue/bh/ad/` | AD CUE schema directory |
| `packages/cue/bh/ad/ad.cue` | AD node/edge/property definitions |
| `packages/cue/bh/azure/` | Azure CUE schema directory |
| `packages/cue/bh/azure/azure.cue` | Azure node/edge/property definitions |

### A2. Generated Go Graph Schemas (DELETE entire directories)

| Path | Description |
|------|-------------|
| `packages/go/graphschema/ad/` | Generated AD Go schema directory |
| `packages/go/graphschema/ad/ad.go` | AD node/edge/property Go types |
| `packages/go/graphschema/ad/composite.go` | AD composite types |
| `packages/go/graphschema/ad/const.go` | AD constants |
| `packages/go/graphschema/azure/` | Generated Azure Go schema directory |
| `packages/go/graphschema/azure/azure.go` | Azure node/edge/property Go types |
| `packages/go/graphschema/azure/composite.go` | Azure composite types |
| `packages/go/graphschema/azure/identifiers.go` | Azure identifiers |
| `packages/go/graphschema/azure/roles.go` | Azure role definitions |

### A3. Ingest Converters (DELETE files)

| Path | Lines | Description |
|------|-------|-------------|
| `packages/go/ein/ad.go` | 1572 | AD ingest conversion (SharpHound→graph nodes). Imports `graphschema/ad`, `graphschema/common` |
| `packages/go/ein/ad_test.go` | — | Tests for AD ingest conversion |
| `packages/go/ein/azure.go` | 2061 | Azure ingest conversion (AzureHound→graph nodes). Imports `graphschema/ad`, `graphschema/azure`, AzureHound models |
| `packages/go/ein/azure_test.go` | — | Tests for Azure ingest conversion |
| `packages/go/ein/incoming_models.go` | 396 | AD-specific incoming model types (IngestBase, Session, etc.). Imports `graphschema/ad` |

### A4. Analysis Packages (DELETE entire directories)

**`packages/go/analysis/ad/` — 23 files:**

| Path |
|------|
| `packages/go/analysis/ad/ad.go` |
| `packages/go/analysis/ad/ad_integration_test.go` |
| `packages/go/analysis/ad/adcs.go` |
| `packages/go/analysis/ad/adcscache.go` |
| `packages/go/analysis/ad/esc1.go` |
| `packages/go/analysis/ad/esc3.go` |
| `packages/go/analysis/ad/esc4.go` |
| `packages/go/analysis/ad/esc6.go` |
| `packages/go/analysis/ad/esc9.go` |
| `packages/go/analysis/ad/esc10.go` |
| `packages/go/analysis/ad/esc13.go` |
| `packages/go/analysis/ad/esc_shared.go` |
| `packages/go/analysis/ad/filters.go` |
| `packages/go/analysis/ad/filters_test.go` |
| `packages/go/analysis/ad/local_groups.go` |
| `packages/go/analysis/ad/membership.go` |
| `packages/go/analysis/ad/ntlm.go` |
| `packages/go/analysis/ad/owns.go` |
| `packages/go/analysis/ad/post.go` |
| `packages/go/analysis/ad/queries.go` |
| `packages/go/analysis/ad/CalculateCrossProductNodeSetsDoc.md` |
| `packages/go/analysis/ad/internal/nodeprops/reader.go` |
| `packages/go/analysis/ad/internal/nodeprops/reader_test.go` |
| `packages/go/analysis/ad/wellknown/prefix.go` |
| `packages/go/analysis/ad/wellknown/prefix_test.go` |
| `packages/go/analysis/ad/wellknown/suffix.go` |
| `packages/go/analysis/ad/wellknown/suffix_test.go` |

**`packages/go/analysis/azure/` — 28 files:**

| Path |
|------|
| `packages/go/analysis/azure/application.go` |
| `packages/go/analysis/azure/automation_account.go` |
| `packages/go/analysis/azure/azure.go` |
| `packages/go/analysis/azure/container_registry.go` |
| `packages/go/analysis/azure/db_ops.go` |
| `packages/go/analysis/azure/device.go` |
| `packages/go/analysis/azure/filters.go` |
| `packages/go/analysis/azure/function_app.go` |
| `packages/go/analysis/azure/group.go` |
| `packages/go/analysis/azure/key_vault.go` |
| `packages/go/analysis/azure/logic_app.go` |
| `packages/go/analysis/azure/managed_cluster.go` |
| `packages/go/analysis/azure/management_group.go` |
| `packages/go/analysis/azure/model.go` |
| `packages/go/analysis/azure/post.go` |
| `packages/go/analysis/azure/post_test.go` |
| `packages/go/analysis/azure/queries.go` |
| `packages/go/analysis/azure/resource_group.go` |
| `packages/go/analysis/azure/role.go` |
| `packages/go/analysis/azure/role_approver.go` |
| `packages/go/analysis/azure/service_principal.go` |
| `packages/go/analysis/azure/subscription.go` |
| `packages/go/analysis/azure/tenant.go` |
| `packages/go/analysis/azure/user.go` |
| `packages/go/analysis/azure/vm.go` |
| `packages/go/analysis/azure/vm_scale_set.go` |
| `packages/go/analysis/azure/web_app.go` |

**`packages/go/analysis/hybrid/` — 1 file:**

| Path |
|------|
| `packages/go/analysis/hybrid/hybrid.go` |

**`cmd/api/src/analysis/ad/` — 5 files:**

| Path |
|------|
| `cmd/api/src/analysis/ad/post.go` |
| `cmd/api/src/analysis/ad/queries.go` |
| `cmd/api/src/analysis/ad/ad_integration_test.go` |
| `cmd/api/src/analysis/ad/adcs_integration_test.go` |
| `cmd/api/src/analysis/ad/ntlm_integration_test.go` |

**`cmd/api/src/analysis/azure/` — 5 files:**

| Path |
|------|
| `cmd/api/src/analysis/azure/base.go` |
| `cmd/api/src/analysis/azure/post.go` |
| `cmd/api/src/analysis/azure/queries.go` |
| `cmd/api/src/analysis/azure/azure_integration_test.go` |
| `cmd/api/src/analysis/azure/pimroles_integration_test.go` |

**`cmd/api/src/analysis/hybrid/` — 1 file:**

| Path |
|------|
| `cmd/api/src/analysis/hybrid/hybrid_integration_test.go` |

### A5. API Handlers (DELETE files)

| Path | Description |
|------|-------------|
| `cmd/api/src/api/v2/ad_entity.go` | AD entity endpoints (`/computers`, `/domains`, `/users`, etc.) |
| `cmd/api/src/api/v2/ad_entity_test.go` | Tests |
| `cmd/api/src/api/v2/ad_related_entity.go` | AD related entity queries |
| `cmd/api/src/api/v2/ad_related_entity_test.go` | Tests |
| `cmd/api/src/api/v2/azure.go` | Azure entity endpoints |
| `cmd/api/src/api/v2/azure_test.go` | Tests |

### A6. Data Quality Models (DELETE files)

| Path | Description |
|------|-------------|
| `cmd/api/src/model/adquality.go` | `ADDataQualityStat` struct |
| `cmd/api/src/model/azurequality.go` | `AzureDataQualityStat` struct |

### A7. Frontend HelpTexts (DELETE entire directory tree — 115+ subdirectories)

**AD-specific HelpText directories** (prefix: ADCS, AD-related concepts):

| Directory |
|-----------|
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ADCSESC1/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ADCSESC10a/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ADCSESC10b/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ADCSESC13/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ADCSESC3/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ADCSESC4/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ADCSESC6a/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ADCSESC6b/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ADCSESC9a/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ADCSESC9b/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AbuseTGTDelegation/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AddAllowedToAct/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AddKeyCredentialLink/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AddMember/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AddSelf/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AdminTo/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AllExtendedRights/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AllowedToAct/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AllowedToDelegate/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/CanPSRemote/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/CanRDP/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ClaimSpecialIdentity/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/CodeController/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/CoerceAndRelayNTLMToADCS/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/CoerceAndRelayNTLMToLDAP/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/CoerceAndRelayNTLMToLDAPS/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/CoerceAndRelayNTLMToSMB/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/CoerceToTGT/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/Contains/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/CrossForestTrust/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/DCFor/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/DCSync/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/DelegatedEnrollmentAgent/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/DumpSMSAPassword/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/Enroll/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/EnrollOnBehalfOf/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/EnterpriseCAFor/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ExecuteDCOM/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ExtendedByPolicy/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ForceChangePassword/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/GPLink/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/GenericAll/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/GenericWrite/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/GetChanges/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/GetChangesAll/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/GoldenCert/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/HasSIDHistory/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/HasSession/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/HasTrustKeys/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/HostsCAService/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/IssuedSignedBy/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ManageCA/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ManageCertificates/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/MemberOf/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/NTAuthStoreFor/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/OIDGroupLink/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/Owns/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/OwnsLimitedRights/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/OwnsRaw/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ProtectAdminGroups/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/PublishedTo/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ReadGMSAPassword/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/ReadLAPSPassword/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/RootCAFor/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/SQLAdmin/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/SameForestTrust/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/SpoofSIDHistory/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/SyncLAPSPassword/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/SyncedToADUser/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/SyncedToEntraUser/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/TrustedForNTAuth/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/WriteAccountRestrictions/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/WriteDacl/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/WriteGPLink/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/WriteOwner/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/WriteOwnerLimitedRights/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/WriteOwnerRaw/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/WritePKIEnrollmentFlag/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/WritePKINameFlag/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/WriteSPN/` |

**Azure-specific HelpText directories** (prefix: AZ):

| Directory |
|-----------|
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZAKSContributor/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZAddMembers/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZAddOwner/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZAddSecret/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZAppAdmin/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZAutomationContributor/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZAvereContributor/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZCloudAppAdmin/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZContains/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZContributor/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZExecuteCommand/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZGetCertificates/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZGetKeys/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZGetSecrets/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZGlobalAdmin/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZHasRole/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZKeyVaultKVContributor/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZLogicAppContributor/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZMGAddMember/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZMGAddOwner/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZMGAddSecret/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZMGAppRoleAssignment_ReadWrite_All/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZMGApplication_ReadWrite_All/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZMGDirectory_ReadWrite_All/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZMGGrantAppRoles/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZMGGrantRole/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZMGGroupMember_ReadWrite_All/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZMGGroup_ReadWrite_All/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZMGRoleManagement_ReadWrite_Directory/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZMGServicePrincipalEndpoint_ReadWrite_All/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZManagedIdentity/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZMemberOf/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZNodeResourceGroup/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZOwner/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZOwns/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZPrivilegedAuthAdmin/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZPrivilegedRoleAdmin/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZResetPassword/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZRoleApprover/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZRoleEligible/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZRunsAs/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZUserAccessAdministrator/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZVMAdminLogin/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZVMContributor/` |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/AZWebsiteContributor/` |

**Also DELETE (umbrella files and shared AD content):**

| Path | Description |
|------|-------------|
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/Default/` | Default (AD fallback) |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/shared/` | Shared AD components (ACLInheritance.tsx) |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/index.tsx` | Master index mapping all AD/Azure edges (REWRITE) |
| `packages/javascript/bh-shared-ui/src/components/HelpTexts/utils.ts` | HelpText utilities |

### A8. Frontend Schemas & Searches (DELETE or REWRITE)

| Path | Action | Description |
|------|--------|-------------|
| `packages/javascript/bh-shared-ui/src/graphSchema.ts` | REWRITE | Contains all AD/Azure enums (`ActiveDirectoryNodeKind`, `AzureNodeKind`, etc.) |
| `packages/javascript/bh-shared-ui/src/commonSearchesAGI.ts` | DELETE | AD/Azure prebuilt Cypher queries (asset group isolation) |
| `packages/javascript/bh-shared-ui/src/commonSearchesAGT.ts` | DELETE | AD/Azure prebuilt Cypher queries (asset group tagging) |
| `packages/javascript/bh-shared-ui/src/commonSearches.test.ts` | DELETE | Tests for above |

### A9. Collector Download Page (DELETE entire directory)

| Path | Description |
|------|-------------|
| `cmd/ui/src/views/DownloadCollectors/` | Collector download view directory |
| `cmd/ui/src/views/DownloadCollectors/DownloadCollectors.tsx` | SharpHound/AzureHound download UI |
| `cmd/ui/src/views/DownloadCollectors/DownloadCollectors.test.tsx` | Tests |
| `cmd/ui/src/views/DownloadCollectors/index.ts` | Re-export |
| `packages/javascript/bh-shared-ui/src/components/CollectorCard/` | Collector card component |
| `packages/javascript/bh-shared-ui/src/components/CollectorCard/CollectorCard.tsx` | Component |
| `packages/javascript/bh-shared-ui/src/components/CollectorCard/index.ts` | Re-export |
| `packages/javascript/bh-shared-ui/src/components/CollectorCardList/` | Collector card list component |
| `packages/javascript/bh-shared-ui/src/components/CollectorCardList/CollectorCardList.tsx` | Component |
| `packages/javascript/bh-shared-ui/src/components/CollectorCardList/index.ts` | Re-export |
| `local-harnesses/collectors/azurehound/` | AzureHound collector harness |
| `local-harnesses/collectors/sharphound/` | SharpHound collector harness |

### A10. Valid Edges Schema (REWRITE)

| Path | Action | Description |
|------|--------|-------------|
| `schemas/valid_edges.json` | REWRITE | Contains all AD/Azure valid source→edge→target combos |

### A11. Additional AD/Azure-Specific Files (DELETE)

| Path | Description |
|------|-------------|
| `cmd/api/src/services/graphify/azure_convertors.go` | Azure-specific graphify converters |
| `cmd/api/src/test/fixtures/fixtures/expected_ingest.go` | AD expected ingest test fixtures |
| `cmd/api/src/test/fixtures/fixtures/expected_ingest_adcs.go` | ADCS expected ingest test fixtures |
| `packages/go/analysis/ad/CalculateCrossProductNodeSetsDoc.md` | AD-specific docs |
| `packages/csharp/graphschema/PropertyNames.cs` | C# property names (AD/Azure-specific) |

---

## Appendix B — Files That Import From Deleted Packages (Need Fixing)

> These files will NOT be deleted — they are reusable infrastructure that references deleted AD/Azure code.
> Each needs its AD/Azure imports removed and replaced with API domain equivalents.

### B1. Files importing `graphschema/ad`

| File | Line | Import |
|------|------|--------|
| [cmd/api/src/cmd/dawgs-harness/tests/tests.go](cmd/api/src/cmd/dawgs-harness/tests/tests.go#L23) | 23 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/daemons/datapipe/agt.go](cmd/api/src/daemons/datapipe/agt.go#L35) | 35 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/daemons/datapipe/pipeline.go](cmd/api/src/daemons/datapipe/pipeline.go#L36) | 36 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/daemons/datapipe/agi.go](cmd/api/src/daemons/datapipe/agi.go#L34) | 34 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/test/integration/graph.go](cmd/api/src/test/integration/graph.go#L27) | 27 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/test/integration/harnesses.go](cmd/api/src/test/integration/harnesses.go#L32) | 32 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/test/lab/fixtures/computer.go](cmd/api/src/test/lab/fixtures/computer.go#L25) | 25 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/test/lab/fixtures/domain.go](cmd/api/src/test/lab/fixtures/domain.go#L23) | 23 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/model/assetgrouptags.go](cmd/api/src/model/assetgrouptags.go#L27) | 27 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/agi_test.go](cmd/api/src/api/agi_test.go#L24) | 24 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/v2/etac.go](cmd/api/src/api/v2/etac.go#L29) | 29 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/v2/assetgrouptags.go](cmd/api/src/api/v2/assetgrouptags.go#L49) | 49 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/v2/auth/etac.go](cmd/api/src/api/v2/auth/etac.go#L29) | 29 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/v2/auth/auth_test.go](cmd/api/src/api/v2/auth/auth_test.go#L61) | 61 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/v2/agi_internal_test.go](cmd/api/src/api/v2/agi_internal_test.go#L25) | 25 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/v2/agi_test.go](cmd/api/src/api/v2/agi_test.go#L42) | 42 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/v2/assetgrouptags_test.go](cmd/api/src/api/v2/assetgrouptags_test.go#L50) | 50 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/v2/search_test.go](cmd/api/src/api/v2/search_test.go#L35) | 35 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/v2/search_internal_test.go](cmd/api/src/api/v2/search_internal_test.go#L24) | 24 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/v2/pathfinding_internal_test.go](cmd/api/src/api/v2/pathfinding_internal_test.go#L22) | 22 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/v2/agi.go](cmd/api/src/api/v2/agi.go#L38) | 38 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/v2/pathfinding.go](cmd/api/src/api/v2/pathfinding.go#L36) | 36 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/v2/search.go](cmd/api/src/api/v2/search.go#L34) | 34 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/v2/pathfinding_test.go](cmd/api/src/api/v2/pathfinding_test.go#L36) | 36 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/api/v2/dataquality.go](cmd/api/src/api/v2/dataquality.go#L29) | 29 | `"github.com/specterops/bloodhound/packages/go/analysis/ad"` |
| [cmd/api/src/api/v2/edge.go](cmd/api/src/api/v2/edge.go#L28) | 28 | `"github.com/specterops/bloodhound/packages/go/analysis/ad"` |
| [cmd/api/src/queries/rewriter.go](cmd/api/src/queries/rewriter.go#L19) | 19 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/queries/graph.go](cmd/api/src/queries/graph.go#L49) | 49 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/queries/graph_test.go](cmd/api/src/queries/graph_test.go#L32) | 32 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/queries/graph_integration_test.go](cmd/api/src/queries/graph_integration_test.go#L38) | 38 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/queries/graph_internal_test.go](cmd/api/src/queries/graph_internal_test.go#L30) | 30 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/analysis/membership_integration_test.go](cmd/api/src/analysis/membership_integration_test.go#L29) | 29 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/analysis/analysis_integration_test.go](cmd/api/src/analysis/analysis_integration_test.go#L30) | 30 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/analysis/post_integration_test.go](cmd/api/src/analysis/post_integration_test.go#L29) | 29 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/services/graphify/convertors.go](cmd/api/src/services/graphify/convertors.go#L26) | 26 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/services/graphify/ingestrelationships.go](cmd/api/src/services/graphify/ingestrelationships.go#L28) | 28 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/services/graphify/ingestrelationships_integration_test.go](cmd/api/src/services/graphify/ingestrelationships_integration_test.go#L27) | 27 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/services/graphify/ingestnodes.go](cmd/api/src/services/graphify/ingestnodes.go#L28) | 28 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/services/graphify/ingestnodes_internal_test.go](cmd/api/src/services/graphify/ingestnodes_internal_test.go#L26) | 26 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/services/graphify/ingest.go](cmd/api/src/services/graphify/ingest.go#L37) | 37 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/migrations/migrations_integration_test.go](cmd/api/src/migrations/migrations_integration_test.go#L29) | 29 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/services/agi/agi.go](cmd/api/src/services/agi/agi.go#L30) | 30 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [cmd/api/src/migrations/manifest.go](cmd/api/src/migrations/manifest.go#L31) | 31 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [packages/go/graphify/graph/graph.go](packages/go/graphify/graph/graph.go#L42) | 42 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [packages/go/graphschema/common/common.go](packages/go/graphschema/common/common.go#L24) | 24 | `ad "github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [packages/go/graphschema/primarykind.go](packages/go/graphschema/primarykind.go#L20) | 20 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [packages/go/graphschema/primarykind_test.go](packages/go/graphschema/primarykind_test.go#L22) | 22 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [packages/go/graphschema/schema.go](packages/go/graphschema/schema.go#L20) | 20 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [packages/go/analysis/analysis.go](packages/go/analysis/analysis.go#L25) | 25 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [packages/go/analysis/analysis_test.go](packages/go/analysis/analysis_test.go#L23) | 23 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [packages/go/analysis/tiering/tiering.go](packages/go/analysis/tiering/tiering.go#L22) | 22 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [packages/go/analysis/tiering/helpers.go](packages/go/analysis/tiering/helpers.go#L20) | 20 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [packages/go/analysis/post_integration_test.go](packages/go/analysis/post_integration_test.go#L29) | 29 | `"github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [packages/go/schemagen/generator/golang.go](packages/go/schemagen/generator/golang.go#L30) | 30 | `ADSchemaPackageName = "github.com/specterops/bloodhound/packages/go/graphschema/ad"` |
| [packages/go/schemagen/main.go](packages/go/schemagen/main.go#L50) | 50 | `GenerateGolangActiveDirectory("ad", ...)` |

### B2. Files importing `graphschema/azure`

| File | Line | Import |
|------|------|--------|
| [cmd/api/src/daemons/datapipe/agt.go](cmd/api/src/daemons/datapipe/agt.go#L36) | 36 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/daemons/datapipe/pipeline.go](cmd/api/src/daemons/datapipe/pipeline.go#L37) | 37 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/daemons/datapipe/agi.go](cmd/api/src/daemons/datapipe/agi.go#L35) | 35 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/test/integration/graph.go](cmd/api/src/test/integration/graph.go#L28) | 28 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/test/integration/harnesses.go](cmd/api/src/test/integration/harnesses.go#L33) | 33 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/model/assetgrouptags.go](cmd/api/src/model/assetgrouptags.go#L28) | 28 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/api/agi_test.go](cmd/api/src/api/agi_test.go#L25) | 25 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/api/v2/etac.go](cmd/api/src/api/v2/etac.go#L30) | 30 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/api/v2/assetgrouptags.go](cmd/api/src/api/v2/assetgrouptags.go#L50) | 50 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/api/v2/auth/etac.go](cmd/api/src/api/v2/auth/etac.go#L30) | 30 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/api/v2/auth/auth_test.go](cmd/api/src/api/v2/auth/auth_test.go#L62) | 62 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/api/v2/agi_internal_test.go](cmd/api/src/api/v2/agi_internal_test.go#L26) | 26 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/api/v2/agi_test.go](cmd/api/src/api/v2/agi_test.go#L43) | 43 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/api/v2/assetgrouptags_test.go](cmd/api/src/api/v2/assetgrouptags_test.go#L51) | 51 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/api/v2/search_test.go](cmd/api/src/api/v2/search_test.go#L36) | 36 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/api/v2/search_internal_test.go](cmd/api/src/api/v2/search_internal_test.go#L25) | 25 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/api/v2/pathfinding_internal_test.go](cmd/api/src/api/v2/pathfinding_internal_test.go#L23) | 23 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/api/v2/pathfinding.go](cmd/api/src/api/v2/pathfinding.go#L37) | 37 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/api/v2/agi.go](cmd/api/src/api/v2/agi.go#L39) | 39 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/api/v2/search.go](cmd/api/src/api/v2/search.go#L35) | 35 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/api/v2/pathfinding_test.go](cmd/api/src/api/v2/pathfinding_test.go#L37) | 37 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/services/graphify/azure_convertors.go](cmd/api/src/services/graphify/azure_convertors.go#L31) | 31 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/services/graphify/ingest.go](cmd/api/src/services/graphify/ingest.go#L38) | 38 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/services/agi/agi.go](cmd/api/src/services/agi/agi.go#L31) | 31 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/queries/rewriter.go](cmd/api/src/queries/rewriter.go#L20) | 20 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/queries/graph.go](cmd/api/src/queries/graph.go#L50) | 50 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/queries/graph_integration_test.go](cmd/api/src/queries/graph_integration_test.go#L39) | 39 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/queries/graph_internal_test.go](cmd/api/src/queries/graph_internal_test.go#L31) | 31 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [cmd/api/src/migrations/manifest.go](cmd/api/src/migrations/manifest.go#L32) | 32 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [packages/go/graphify/graph/graph.go](packages/go/graphify/graph/graph.go#L43) | 43 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [packages/go/graphschema/common/common.go](packages/go/graphschema/common/common.go#L25) | 25 | `azure "github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [packages/go/graphschema/primarykind.go](packages/go/graphschema/primarykind.go#L21) | 21 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [packages/go/graphschema/schema.go](packages/go/graphschema/schema.go#L21) | 21 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [packages/go/analysis/analysis.go](packages/go/analysis/analysis.go#L26) | 26 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [packages/go/analysis/analysis_test.go](packages/go/analysis/analysis_test.go#L24) | 24 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [packages/go/analysis/post_integration_test.go](packages/go/analysis/post_integration_test.go#L30) | 30 | `"github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [packages/go/schemagen/generator/golang.go](packages/go/schemagen/generator/golang.go#L31) | 31 | `AzureSchemaPackageName = "github.com/specterops/bloodhound/packages/go/graphschema/azure"` |
| [packages/go/schemagen/main.go](packages/go/schemagen/main.go#L54) | 54 | `GenerateGolangAzure("azure", ...)` |

### B3. Files importing `analysis/ad` (package-level)

| File | Line | Import |
|------|------|--------|
| [cmd/api/src/daemons/datapipe/analysis.go](cmd/api/src/daemons/datapipe/analysis.go#L25) | 25 | `"github.com/specterops/bloodhound/cmd/api/src/analysis/ad"` |
| [cmd/api/src/daemons/datapipe/agi.go](cmd/api/src/daemons/datapipe/agi.go#L30) | 30 | `adAnalysis "github.com/specterops/bloodhound/packages/go/analysis/ad"` |
| [cmd/api/src/test/integration/harnesses.go](cmd/api/src/test/integration/harnesses.go#L30) | 30 | `adAnalysis "github.com/specterops/bloodhound/packages/go/analysis/ad"` |
| [cmd/api/src/api/v2/dataquality.go](cmd/api/src/api/v2/dataquality.go#L29) | 29 | `"github.com/specterops/bloodhound/packages/go/analysis/ad"` |
| [cmd/api/src/api/v2/edge.go](cmd/api/src/api/v2/edge.go#L28) | 28 | `"github.com/specterops/bloodhound/packages/go/analysis/ad"` |
| [cmd/api/src/services/dataquality/dataquality.go](cmd/api/src/services/dataquality/dataquality.go#L25) | 25 | `"github.com/specterops/bloodhound/cmd/api/src/analysis/ad"` |
| [cmd/api/src/queries/graph_integration_test.go](cmd/api/src/queries/graph_integration_test.go#L36) | 36 | `adAnalysis "github.com/specterops/bloodhound/packages/go/analysis/ad"` |
| [cmd/api/src/analysis/membership_integration_test.go](cmd/api/src/analysis/membership_integration_test.go#L27) | 27 | `analysis "github.com/specterops/bloodhound/packages/go/analysis/ad"` |
| [cmd/api/src/analysis/analysis_integration_test.go](cmd/api/src/analysis/analysis_integration_test.go#L28) | 28 | `adAnalysis "github.com/specterops/bloodhound/packages/go/analysis/ad"` |
| [cmd/api/src/analysis/post_integration_test.go](cmd/api/src/analysis/post_integration_test.go#L26) | 26 | `ad2 "github.com/specterops/bloodhound/packages/go/analysis/ad"` |

### B4. Files importing `analysis/azure` (package-level)

| File | Line | Import |
|------|------|--------|
| [cmd/api/src/daemons/datapipe/analysis.go](cmd/api/src/daemons/datapipe/analysis.go#L26) | 26 | `"github.com/specterops/bloodhound/cmd/api/src/analysis/azure"` |
| [cmd/api/src/daemons/datapipe/agi.go](cmd/api/src/daemons/datapipe/agi.go#L31) | 31 | `azureAnalysis "github.com/specterops/bloodhound/packages/go/analysis/azure"` |
| [cmd/api/src/services/dataquality/dataquality.go](cmd/api/src/services/dataquality/dataquality.go#L26) | 26 | `"github.com/specterops/bloodhound/cmd/api/src/analysis/azure"` |
| [packages/go/analysis/post_integration_test.go](packages/go/analysis/post_integration_test.go#L27) | 27 | `azureAnalysis "github.com/specterops/bloodhound/packages/go/analysis/azure"` |

### B5. Files importing `analysis/hybrid`

| File | Line | Import |
|------|------|--------|
| [cmd/api/src/analysis/azure/post.go](cmd/api/src/analysis/azure/post.go#L25) | 25 | `"github.com/specterops/bloodhound/packages/go/analysis/hybrid"` (also being deleted) |
| [cmd/api/src/analysis/hybrid/hybrid_integration_test.go](cmd/api/src/analysis/hybrid/hybrid_integration_test.go#L27) | 27 | `"github.com/specterops/bloodhound/packages/go/analysis/hybrid"` (also being deleted) |

### B6. Files importing `analysis/ad/wellknown`

| File | Line | Import |
|------|------|--------|
| [cmd/api/src/test/integration/harnesses.go](cmd/api/src/test/integration/harnesses.go#L31) | 31 | `"github.com/specterops/bloodhound/packages/go/analysis/ad/wellknown"` |
| [cmd/api/src/migrations/manifest.go](cmd/api/src/migrations/manifest.go#L29) | 29 | `"github.com/specterops/bloodhound/packages/go/analysis/ad/wellknown"` |

### B7. Files importing `analysis/tiering` (references AD kinds internally)

| File | Line | Import |
|------|------|--------|
| [cmd/api/src/api/bloodhoundgraph/properties.go](cmd/api/src/api/bloodhoundgraph/properties.go#L21) | 21 | `"github.com/specterops/bloodhound/packages/go/analysis/tiering"` |
| [cmd/api/src/api/v2/ad_entity.go](cmd/api/src/api/v2/ad_entity.go#L28) | 28 | `"github.com/specterops/bloodhound/packages/go/analysis/tiering"` (also being deleted) |
| [cmd/api/src/model/unified_graph.go](cmd/api/src/model/unified_graph.go#L23) | 23 | `"github.com/specterops/bloodhound/packages/go/analysis/tiering"` |

Note: `packages/go/analysis/tiering/` itself imports `graphschema/ad` and needs rewriting.

### B8. Files importing `go/ein` package (use AD/Azure types from it)

| File | Line | Import |
|------|------|--------|
| [cmd/api/src/services/graphify/convertors.go](cmd/api/src/services/graphify/convertors.go#L25) | 25 | `"github.com/specterops/bloodhound/packages/go/ein"` |
| [cmd/api/src/services/graphify/decoders.go](cmd/api/src/services/graphify/decoders.go#L27) | 27 | `"github.com/specterops/bloodhound/packages/go/ein"` |
| [cmd/api/src/services/graphify/ingestrelationships.go](cmd/api/src/services/graphify/ingestrelationships.go#L26) | 26 | `"github.com/specterops/bloodhound/packages/go/ein"` |
| [cmd/api/src/services/graphify/ingestrelationships_integration_test.go](cmd/api/src/services/graphify/ingestrelationships_integration_test.go#L25) | 25 | `"github.com/specterops/bloodhound/packages/go/ein"` |
| [cmd/api/src/services/graphify/azure_convertors.go](cmd/api/src/services/graphify/azure_convertors.go#L30) | 30 | `"github.com/specterops/bloodhound/packages/go/ein"` (also being deleted) |
| [cmd/api/src/services/graphify/models.go](cmd/api/src/services/graphify/models.go#L23) | 23 | `"github.com/specterops/bloodhound/packages/go/ein"` |
| [cmd/api/src/services/graphify/ingest.go](cmd/api/src/services/graphify/ingest.go#L35) | 35 | `"github.com/specterops/bloodhound/packages/go/ein"` |
| [cmd/api/src/services/graphify/endpoint/digest.go](cmd/api/src/services/graphify/endpoint/digest.go#L23) | 23 | `"github.com/specterops/bloodhound/packages/go/ein"` |
| [cmd/api/src/services/graphify/endpoint/fetch.go](cmd/api/src/services/graphify/endpoint/fetch.go#L27) | 27 | `"github.com/specterops/bloodhound/packages/go/ein"` |
| [cmd/api/src/services/graphify/endpoint/resolver.go](cmd/api/src/services/graphify/endpoint/resolver.go#L22) | 22 | `"github.com/specterops/bloodhound/packages/go/ein"` |
| [cmd/api/src/services/graphify/endpoint/fetch_test.go](cmd/api/src/services/graphify/endpoint/fetch_test.go#L22) | 22 | `"github.com/specterops/bloodhound/packages/go/ein"` |
| [cmd/api/src/services/graphify/endpoint/fetch_integration_test.go](cmd/api/src/services/graphify/endpoint/fetch_integration_test.go#L28) | 28 | `"github.com/specterops/bloodhound/packages/go/ein"` |
| [cmd/api/src/services/graphify/endpoint/digest_test.go](cmd/api/src/services/graphify/endpoint/digest_test.go#L23) | 23 | `"github.com/specterops/bloodhound/packages/go/ein"` |
| [cmd/api/src/services/graphify/ingestnodes.go](cmd/api/src/services/graphify/ingestnodes.go#L27) | 27 | `"github.com/specterops/bloodhound/packages/go/ein"` |
| [cmd/api/src/services/graphify/ingestnodes_integration_test.go](cmd/api/src/services/graphify/ingestnodes_integration_test.go#L27) | 27 | `"github.com/specterops/bloodhound/packages/go/ein"` |
| [cmd/api/src/test/fixtures/fixtures/expected_ingest_adcs.go](cmd/api/src/test/fixtures/fixtures/expected_ingest_adcs.go#L21) | 21 | `"github.com/specterops/bloodhound/packages/go/ein"` |

### B9. Frontend files importing from `graphSchema.ts`

| File | Line | Imports Used |
|------|------|--------------|
| [packages/javascript/bh-shared-ui/src/index.ts](packages/javascript/bh-shared-ui/src/index.ts#L56) | 56 | `export * from './graphSchema'` |
| [packages/javascript/bh-shared-ui/src/constants.ts](packages/javascript/bh-shared-ui/src/constants.ts#L28) | 28 | AD/Azure enums from `./graphSchema` |
| [packages/javascript/bh-shared-ui/src/views/Explore/ExploreSearch/EdgeFilter/edgeCategories.test.tsx](packages/javascript/bh-shared-ui/src/views/Explore/ExploreSearch/EdgeFilter/edgeCategories.test.tsx#L16) | 16 | `ActiveDirectoryPathfindingEdgesMatchFrontend, AzurePathfindingEdges` |
| [packages/javascript/bh-shared-ui/src/views/Explore/ExploreSearch/EdgeFilter/edgeCategories.tsx](packages/javascript/bh-shared-ui/src/views/Explore/ExploreSearch/EdgeFilter/edgeCategories.tsx#L17) | 17 | `ActiveDirectoryRelationshipKind, AzureRelationshipKind` |
| [packages/javascript/bh-shared-ui/src/views/Explore/fragments.tsx](packages/javascript/bh-shared-ui/src/views/Explore/fragments.tsx#L19) | 19 | `ActiveDirectoryKindProperties, AzureKindProperties, CommonKindProperties` |
| [packages/javascript/bh-shared-ui/src/views/Explore/BasicObjectInfoFields.tsx](packages/javascript/bh-shared-ui/src/views/Explore/BasicObjectInfoFields.tsx#L19) | 19 | `ActiveDirectoryNodeKind, AzureNodeKind, CommonKindProperties` |
| [packages/javascript/bh-shared-ui/src/views/Explore/EdgeInfo/EdgeInfoContent.tsx](packages/javascript/bh-shared-ui/src/views/Explore/EdgeInfo/EdgeInfoContent.tsx#L20) | 20 | `ActiveDirectoryKindProperties, CommonKindProperties` |
| [packages/javascript/bh-shared-ui/src/views/Explore/EdgeInfo/EdgeInfoContent.test.tsx](packages/javascript/bh-shared-ui/src/views/Explore/EdgeInfo/EdgeInfoContent.test.tsx#L25) | 25 | AD/Azure types from `../../../graphSchema` |
| [packages/javascript/bh-shared-ui/src/views/DataQuality/TenantInfo.tsx](packages/javascript/bh-shared-ui/src/views/DataQuality/TenantInfo.tsx#L24) | 24 | `AzureNodeKind` |
| [packages/javascript/bh-shared-ui/src/views/DataQuality/DomainInfo.tsx](packages/javascript/bh-shared-ui/src/views/DataQuality/DomainInfo.tsx#L24) | 24 | `ActiveDirectoryNodeKind` |
| [packages/javascript/bh-shared-ui/src/components/EntityInfoDataTable/EntityInfoDataTable.test.tsx](packages/javascript/bh-shared-ui/src/components/EntityInfoDataTable/EntityInfoDataTable.test.tsx#L19) | 19 | `ActiveDirectoryNodeKind, AzureNodeKind` |
| [packages/javascript/bh-shared-ui/src/components/ExploreSearchCombobox/ExploreSearchCombobox.test.tsx](packages/javascript/bh-shared-ui/src/components/ExploreSearchCombobox/ExploreSearchCombobox.test.tsx#L21) | 21 | `ActiveDirectoryNodeKind` |
| [packages/javascript/bh-shared-ui/src/components/ExploreTable/explore-table-utils.ts](packages/javascript/bh-shared-ui/src/components/ExploreTable/explore-table-utils.ts#L17) | 17 | `CommonKindProperties` |
| [packages/javascript/bh-shared-ui/src/components/HelpTexts/shared/ACLInheritance.tsx](packages/javascript/bh-shared-ui/src/components/HelpTexts/shared/ACLInheritance.tsx#L19) | 19 | `ActiveDirectoryKindProperties` (also being deleted) |
| [packages/javascript/bh-shared-ui/src/hooks/useSearch/useSearch.tsx](packages/javascript/bh-shared-ui/src/hooks/useSearch/useSearch.tsx#L18) | 18 | `ActiveDirectoryNodeKind, AzureNodeKind` |
| [packages/javascript/bh-shared-ui/src/hooks/useSearch/useSearch.test.ts](packages/javascript/bh-shared-ui/src/hooks/useSearch/useSearch.test.ts#L17) | 17 | `ActiveDirectoryNodeKind` |
| [packages/javascript/bh-shared-ui/src/components/AssetGroupFilters/AssetGroupFilters.test.tsx](packages/javascript/bh-shared-ui/src/components/AssetGroupFilters/AssetGroupFilters.test.tsx#L21) | 21 | `ActiveDirectoryNodeKind` |
| [packages/javascript/bh-shared-ui/src/components/EntityInfoDataTableGraphed/EntityInfoDataTableGraphed.test.tsx](packages/javascript/bh-shared-ui/src/components/EntityInfoDataTableGraphed/EntityInfoDataTableGraphed.test.tsx#L19) | 19 | `ActiveDirectoryNodeKind, AzureNodeKind` |
| [packages/javascript/bh-shared-ui/src/components/EntityInfo/EntityInfoHeader.test.tsx](packages/javascript/bh-shared-ui/src/components/EntityInfo/EntityInfoHeader.test.tsx#L21) | 21 | `AzureNodeKind` |
| [packages/javascript/bh-shared-ui/src/components/EntityInfo/EntityInfoDataTableList.test.tsx](packages/javascript/bh-shared-ui/src/components/EntityInfo/EntityInfoDataTableList.test.tsx#L19) | 19 | `ActiveDirectoryNodeKind` |
| [packages/javascript/bh-shared-ui/src/components/EntityInfo/EntityInfoPanel.test.tsx](packages/javascript/bh-shared-ui/src/components/EntityInfo/EntityInfoPanel.test.tsx#L20) | 20 | `AzureNodeKind` |
| [packages/javascript/bh-shared-ui/src/components/EntityInfo/EntityInfoDataTableList.tsx](packages/javascript/bh-shared-ui/src/components/EntityInfo/EntityInfoDataTableList.tsx#L18) | 18 | `ActiveDirectoryNodeKind` |
| [packages/javascript/bh-shared-ui/src/components/EntityInfo/EntityInfoContent.test.tsx](packages/javascript/bh-shared-ui/src/components/EntityInfo/EntityInfoContent.test.tsx#L18) | 18 | `ActiveDirectoryNodeKind, AzureNodeKind` |
| [packages/javascript/bh-shared-ui/src/mocks/factories/initial.ts](packages/javascript/bh-shared-ui/src/mocks/factories/initial.ts#L18) | 18 | `ActiveDirectoryNodeKind` |
| [packages/javascript/bh-shared-ui/src/utils/entityInfoDisplay.ts](packages/javascript/bh-shared-ui/src/utils/entityInfoDisplay.ts#L30) | 30 | AD/Azure types from `../graphSchema` |
| [packages/javascript/bh-shared-ui/src/utils/entityInfoDisplay.test.ts](packages/javascript/bh-shared-ui/src/utils/entityInfoDisplay.test.ts#L22) | 22 | AD/Azure types from `../graphSchema` |
| [packages/javascript/bh-shared-ui/src/utils/icons.ts](packages/javascript/bh-shared-ui/src/utils/icons.ts#L49) | 49 | `ActiveDirectoryNodeKind, AzureNodeKind` |
| [packages/javascript/bh-shared-ui/src/utils/content.ts](packages/javascript/bh-shared-ui/src/utils/content.ts#L19) | 19 | `ActiveDirectoryNodeKind, AzureNodeKind` |

### B10. Frontend files importing from `commonSearches` / `HelpTexts`

| File | Line | Import |
|------|------|--------|
| [packages/javascript/bh-shared-ui/src/hooks/usePrebuiltQueries/usePrebuiltQueries.tsx](packages/javascript/bh-shared-ui/src/hooks/usePrebuiltQueries/usePrebuiltQueries.tsx#L18) | 18-19 | `commonSearchesAGI`, `commonSearchesAGT` |
| [packages/javascript/bh-shared-ui/src/views/Explore/EdgeInfo/EdgeInfoContent.tsx](packages/javascript/bh-shared-ui/src/views/Explore/EdgeInfo/EdgeInfoContent.tsx#L18) | 18-19 | `EdgeInfoComponents` from `HelpTexts`, `ACLInheritance` |
| [packages/javascript/bh-shared-ui/src/views/Explore/EdgeInfo/EdgeInfoContent.test.tsx](packages/javascript/bh-shared-ui/src/views/Explore/EdgeInfo/EdgeInfoContent.test.tsx#L19) | 19 | `INHERITANCE_DROPDOWN_DESCRIPTION` from `HelpTexts/shared/ACLInheritance` |
| [packages/javascript/bh-shared-ui/src/components/index.ts](packages/javascript/bh-shared-ui/src/components/index.ts#L88) | 88-89 | `export * from './HelpTexts/index'`, `export { default as EdgeInfoComponents }` |

### B11. Frontend routing files referencing collectors

| File | Line | Reference |
|------|------|-----------|
| [cmd/ui/src/routes/index.ts](cmd/ui/src/routes/index.ts#L28) | 28, 85-86 | `DownloadCollectors` lazy import and route registration |
| [cmd/ui/src/routes/constants.ts](cmd/ui/src/routes/constants.ts#L29) | 29 | `ROUTE_DOWNLOAD_COLLECTORS` constant |

---

## Appendix C — Deletion Summary Statistics

| Category | Files to Delete | Directories to Delete |
|----------|----------------:|---------------------:|
| CUE schemas | 2 | 2 |
| Generated Go schemas | 7 | 2 |
| Ingest converters (ein) | 5 | 0 |
| Analysis (packages/go) | ~52 | 5 |
| Analysis (cmd/api/src) | 11 | 3 |
| API handlers | 6 | 0 |
| Data quality models | 2 | 0 |
| Frontend HelpTexts | ~230+ | 115+ |
| Frontend schemas/searches | 4 | 0 |
| Collector download UI | 7 | 3 |
| Valid edges | 1 | 0 |
| Other (C#, graphify, fixtures) | 4 | 0 |
| **TOTAL** | **~330+** | **~130+** |

| Category | Files to Fix (import cleanup) |
|----------|------------------------------:|
| Go files importing graphschema/ad | 52 (unique, excluding deleted files) |
| Go files importing graphschema/azure | 37 (unique, excluding deleted files) |
| Go files importing analysis/ad | 10 |
| Go files importing analysis/azure | 4 |
| Go files importing analysis/hybrid | 2 (both also being deleted) |
| Go files importing ad/wellknown | 2 |
| Go files importing ein (AD/Azure types) | 16 |
| Frontend files importing graphSchema.ts | 28 |
| Frontend files importing HelpTexts/commonSearches | 4 |
| Frontend routing files | 2 |
| **TOTAL FILES NEEDING IMPORT FIXES** | **~90 unique files** |
