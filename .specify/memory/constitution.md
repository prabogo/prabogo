<!--
SYNC IMPACT REPORT
==================
Version change: (unversioned template) → 1.0.0
Modified principles: N/A (initial ratification)
Added sections:
  - Core Principles (I–V)
  - Architecture Standards
  - Development Workflow
  - Governance
Removed sections: N/A (initial creation)
Templates requiring updates:
  - .specify/templates/plan-template.md ✅ (Constitution Check gates align with principles below)
  - .specify/templates/spec-template.md ✅ (no mandatory section additions required)
  - .specify/templates/tasks-template.md ✅ (test-first tasks, observability tasks align)
Follow-up TODOs: None — all placeholders resolved.
-->

# Prabogo Constitution

## Core Principles

### I. Hexagonal Architecture (NON-NEGOTIABLE)

Prabogo MUST strictly follow the Ports and Adapters (Hexagonal Architecture) pattern:

- The **domain** layer contains all business logic and MUST NOT import adapter or infrastructure packages.
- **Ports** are interfaces defined in `port/inbound/` and `port/outbound/`; they form the only contract
  between the domain and the outside world.
- **Adapters** in `adapter/inbound/` and `adapter/outbound/` implement ports and MUST NOT contain
  business logic — only translation and delegation.
- External technologies (PostgreSQL, RabbitMQ, Redis, Fiber, Temporal) MUST be isolated behind
  outbound port interfaces so they can be replaced without touching domain code.
- No circular imports are permitted across layers.

### II. Dependency Injection & No Global State

All components MUST be wired together via explicit dependency injection:

- Global variables are forbidden. State MUST flow through constructor parameters or function arguments.
- Application bootstrap lives in `internal/app.go`; adapters and domain services are instantiated once
  and passed down.
- Interfaces MUST be used for every inter-component boundary to keep components independently testable.

### III. Test-First Domain Logic (NON-NEGOTIABLE)

Unit tests for all domain logic MUST be written before or alongside implementation:

- The Red-Green-Refactor cycle MUST be followed: write a failing test, implement to make it pass, then
  refactor.
- Domain tests MUST be runnable in isolation with no external services (use mocks from `tests/mocks/`).
- Integration tests live in `tests/integration/` and require real infrastructure via Docker Compose.
- Mock generation MUST use `golang/mock`; generated mocks live in `tests/mocks/port/`.

### IV. Observability & Structured Logging

Every adapter and domain operation of significance MUST be observable:

- Structured logging is REQUIRED via the `utils/log/` package (logrus-based, with hook support).
- Activity tracking for key operations MUST use `utils/activity/`.
- Errors MUST be wrapped with context using `palantir/stacktrace` or `pkg/errors` before propagation.
- No silent error swallowing — every caught error MUST be either handled or propagated with context.

### V. Simplicity & Go Idioms

Prabogo MUST remain a lean, idiomatic Go framework:

- YAGNI: implement only what is required by the current spec. No speculative abstractions.
- Follow standard Go naming conventions: `camelCase` for unexported, `PascalCase` for exported symbols.
- Makefile targets MUST be used for code generation, testing, and build automation.
- Go version MUST be >= 1.24.0 as declared in `go.mod`.
- Dependencies MUST be justified; avoid adding transitive complexity without clear benefit.

## Architecture Standards

The following technology choices are locked for the current major version:

| Concern          | Technology           | Location                        |
|------------------|----------------------|---------------------------------|
| HTTP server      | Fiber v2             | `adapter/inbound/fiber/`        |
| Message broker   | RabbitMQ (amqp091)   | `adapter/inbound/rabbitmq/`     |
| Workflow engine  | Temporal             | `adapter/inbound/temporal/`     |
| Database         | PostgreSQL (lib/pq)  | `adapter/outbound/postgres/`    |
| Cache            | Redis                | `adapter/outbound/redis/`       |
| Migrations       | Goose v3             | `internal/migration/postgres/`  |
| Auth tokens      | golang-jwt/jwt v5    | `utils/jwt/`                    |

Replacing any technology MUST be done by creating a new adapter implementing the existing port interface.
The port interface MUST NOT change for a replacement unless a minor or major version bump is warranted.

## Development Workflow

1. **Spec first**: All features begin with a spec via `/speckit.specify` before any code is written.
2. **Plan before implementing**: Run `/speckit.plan` to produce design artifacts before tasks.
3. **Tasks drive implementation**: Use `/speckit.tasks` and `/speckit.implement` for structured execution.
4. **Tests gate merges**: All domain unit tests MUST pass (`make test`) before a feature is considered done.
5. **Docker for infrastructure**: External services MUST be started via `docker-compose.yml` for local
   development and integration tests.
6. **No force-push to `master`**: The main branch is protected; changes go through feature branches.

## Governance

This constitution supersedes all informal coding conventions documented elsewhere. Amendments require:

1. A written rationale explaining what principle changes and why.
2. A version bump following semantic versioning (see Sync Impact Report above for bump rules).
3. Updates to all dependent templates (plan, spec, tasks) to reflect changed gates or guidance.
4. All active feature specs in-flight MUST be reviewed for compatibility with the amended principle.

All PRs MUST pass the Constitution Check gate defined in the plan template before merging.
Complexity deviations from Principle V MUST be documented in the plan's Complexity Tracking table.

**Version**: 1.0.0 | **Ratified**: 2026-05-03 | **Last Amended**: 2026-05-03
