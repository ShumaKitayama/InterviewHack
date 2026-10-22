# Design Documents Are the Source of Truth (Critical)

In this repository, **documents under `DesignDocument/` take precedence over code**.
AI agents **must read the relevant design documents before any implementation, modification, or refactor**.
If there is any conflict, **the design is correct and code must be aligned to it**.

Authoritative references (fixed numbering):

- DesignDocument/0_Comparison.md
- DesignDocument/1_Requirementdefinition.md
- DesignDocument/2_DomainDesign.md
- DesignDocument/3_UIUXDesign.md
- DesignDocument/4_LLMDesign.md
- DesignDocument/5_FrontendDesign.md
- DesignDocument/6_AuthAccountCahrgingDesign.md
- DesignDocument/7_BackendDesign.md
- DesignDocument/8_DBDesign.md
- DesignDocument/9_SecurityDesign.md
- DesignDocument/10_AnalyzeDesign.md
- DesignDocument/11_TestDesign.md
- DesignDocument/12_Operation.md
- DesignDocument/13_Schedule.md
- DesignDocument/14_CICDDesign.md

Direction memo: `DesignDocument/DirectionChange_20261022.md` records the 2026-10-22 redesign (decisions D1–D16) and the cross-chapter contract (section 2.10). Read it together with the numbered docs.

Mandatory rules:

1. Identify and review **at least one** relevant design document before making changes.
2. When implementing, **state which chapter/section you follow** in comments or the PR description.
3. If design and code diverge, **update the design first**, then change the code.
4. **Do not introduce new concepts** (tables/APIs/flows) that are not described in the design.
5. **BFF boundaries, the prompt_sessions state machine, auth policy, app-layer authorization (user_id scoping), idempotency, and CI/CD** must follow their respective design docs.

Pre-flight checklist (before you start):

- Which design docs (numbers) are relevant?
- Have you confirmed boundaries (the iOS app reaches data only through the BFF; every query is scoped by user_id)? (5_FrontendDesign, 7_BackendDesign, 8_DBDesign)
- Are you preserving the prompt_sessions state machine (created → pasted → saved / abandoned)? (2_DomainDesign, 7_BackendDesign)
- Do you satisfy idempotency (conditional UPDATE on commit; share tokens stored only as hashes)? (7,8,9)
- Are you following the auth policy (Firebase Auth, Email/Password first; BFF verifies ID tokens)? (5,6,7,9)
- Does the BFF stay free of LLM APIs and outbound calls to arbitrary URLs? (4,7,9)
- Are you propagating correlation IDs (X-Request-Id)? (5,7,14)
- Are you following the test strategy? (11_TestDesign)
- Are you respecting deploy/secrets/environment separation? (14_CICDDesign, 9_SecurityDesign)
- Does this change follow Clean Architecture / DDD layers? (DesignDocument/7_BackendDesign.md)

---

# AGENTS.md

## Project Overview

- **Project Name:** InterviewHack
- **Purpose:** Interview / ES preparation assistant for job hunting and career change (iPhone app)
- **Architecture:**
  - **Client:** iOS (SwiftUI, SwiftData cache, Firebase Auth iOS SDK) — `ios/`
  - **Backend:** Go + Gin BFF on Cloud Run — `backend/`
  - **DB:** Neon (PostgreSQL), accessed only by the BFF
  - **Auth:** Firebase Auth (ID tokens verified by the BFF with Google's public keys)
  - **AI:** no LLM API. "AI連携": the BFF builds prompts, the user pastes them into ChatGPT / Gemini, and the BFF parses the pasted response (4_LLMDesign)
  - **Payments:** none in the MVP (D14)
- **Coding Agents Purpose:**
  - Generate implementation code, tests, and migrations
  - Provide refactors and bug fixes consistent with the architecture
  - Respect boundaries, security, and error-handling standards

---

## Setup & Environment

### Prerequisites

- Go 1.27+
- PostgreSQL 15+ for local development and integration tests (or a Neon dev branch)
- Swift 6 (Command Line Tools is enough for `swift test` of `ios/InterviewHackKit`)
- Xcode + XcodeGen for building the iOS app target (`ios/project.yml`)

### Setup Commands

```bash
# Backend: configure, migrate, run (from the repository root)
cp backend/.env.example backend/.env
DATABASE_URL=postgres://postgres@localhost:5432/interviewhack ./backend/scripts/migrate.sh
cd backend && go run ./cmd/api

# Backend tests (integration tests run only when TEST_DATABASE_URL is set)
cd backend && go vet ./... && go test ./...
cd backend && TEST_DATABASE_URL=postgres://postgres@localhost:5432/interviewhack_test go test ./...
cd backend && bash scripts/arch_check.sh

# iOS logic / UI package tests
cd ios/InterviewHackKit && swift test
```

### Environment Variables

Backend (see README for the full table):

- `DATABASE_URL` (secret), `FIREBASE_PROJECT_ID`, `SHARE_BASE_URL`, `APP_ENV`, `LOG_LEVEL`
- Feature flags: `FEATURE_SHARE_VIEW`, `FEATURE_SHARE_CREATE`, `FEATURE_COMMIT_DISABLED_USE_CASES`

iOS: `GoogleService-Info-Dev.plist` / `GoogleService-Info-Prod.plist` (not committed) and the xcconfig per build configuration (`Debug-Dev` / `Release-Prod`).

---

## Coding Conventions

### Languages & Standards

- **Go:** idiomatic Go (error handling, structured logging with `log/slog`, camelCase JSON tags), `gofmt`
- **Swift:** SwiftUI + Observation, Swift Testing, async/await; keep testable logic in `InterviewHackKit`
- **Templates:** prompt templates and parsers are pure functions in `backend/internal/domain/service/prompt`, versioned per use case (4.3.6)

### Do

- Use consistent naming (camelCase for variables, PascalCase for components)
- Write unit tests for logic-heavy modules
- Keep commits atomic and meaningful
- Include consistent error handling and logging

### Don’t

- Use inline styles arbitrarily
- Ignore lint errors
- Write untyped or unsafe Go code
- Hard-code API keys or secrets

---

## Project Structure

```
/
├─ DesignDocument/          # design docs (source of truth)
├─ backend/
│  ├─ cmd/api/              # main: config, DI, server
│  ├─ internal/{domain,usecase,port,adapter/http,infra,app}
│  ├─ migrations/           # idempotent SQL, applied in order by scripts/migrate.sh
│  └─ scripts/              # migrate.sh, arch_check.sh
├─ ops/costguard/           # cost guard: stops the prod API near the free tier (separate Go module, 12.12)
└─ ios/
   ├─ InterviewHackKit/     # Swift package: models, API client, cache, AI連携 flow, SwiftUI views, tests
   ├─ App/                  # app target: Firebase, SwiftData, entry point
   └─ project.yml           # XcodeGen
```

---

## Boundaries & Responsibilities

### Client vs BFF

- The iOS app never connects to the database. Every read and write goes through the BFF.
- Every repository query is scoped by `user_id` taken from the verified token; other users' rows are reported as 404.
- The only unauthenticated lookups are `GET /health` and the share page `GET /s/{token}` (by token hash).
- The BFF does not call LLM APIs, external AI services or arbitrary URLs. Research material text is extracted on the device.
  (Always refer to DesignDocument/7_BackendDesign.md and DesignDocument/8_DBDesign.md for details.)

### API Contracts

- Follow the existing API scheme and consistent error codes
- Propagate and log `X-Request-Id` for correlation

---

## Backend Architecture (Clean Architecture / DDD)

**The source of truth is DesignDocument/7_BackendDesign.md.** All backend changes must follow it.

### Principles

- The backend follows **DDD-oriented Clean Architecture**.
- Dependency direction must be inward: **Domain <- Usecase(Application) <- Adapter(Interface) <- Infrastructure**.
- Business rules live in **Domain** and must not depend on frameworks or external SDKs.

### Hard Rules (Must)

- **Domain** must NOT import: `gin`, `pgx`, `firebase`, `net/http`, or any network/DB SDK (checked by `backend/scripts/arch_check.sh`).
- **Usecase/Application** must NOT depend on Gin contexts or concrete SDKs; depend on interfaces only.
- **Infrastructure** is the only layer allowed to touch the DB (pgx), the network (Firebase public keys) and logging backends.
- Repositories and external clients are defined as **interfaces** in inner layers and implemented in Infrastructure.
- DTO ↔ Usecase Input ↔ Domain ↔ Persistence model conversions happen in **Adapter/Infrastructure**, not in Domain.

### Go Package Layout

```
backend/
  cmd/api/                 # main, server bootstrap
  internal/
    domain/                # entity, vo, error, service/{prompt,share,session,es}
    usecase/               # account, asset, company, research, es, interview, qa, dashboard, prompt, share
    port/                  # repository and external interfaces
    adapter/http/          # handler, middleware, presenter, dto, sharepage
    infra/                 # postgres, firebase, system, logging, config
    app/                   # DI shared by cmd/api and the integration tests
```

### What Agents Must Do

- When adding business logic, create/modify **Domain** + **Usecase** first, then wire via **Adapter/Infra**.
- Do not place business rules in handlers or infra implementations.
- If unsure about layer placement, **check DesignDocument/7_BackendDesign.md first**.

---

## Testing Guidelines

**The source of truth is DesignDocument/11_TestDesign.md.**

- Domain/Usecase should be unit-tested with mocks for ports; Adapter/Infra should be covered by integration tests.

### Unit Tests

- Backend: Go tests (table-driven for templates / parsers / state transitions); integration tests in `internal/app` against PostgreSQL with a stub token verifier
- iOS: Swift Testing in `ios/InterviewHackKit` (API client, cache, AI連携 flow, contract tests that decode real BFF responses)

### Coverage

- Aim for ≥ 80% coverage on backend critical logic

### CI Validation

Before merge:

```bash
cd backend && gofmt -l . && go vet ./... && go test ./...
cd backend && TEST_DATABASE_URL=<test db> go test ./...
cd backend && bash scripts/arch_check.sh
cd backend && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...
cd backend && go run golang.org/x/vuln/cmd/govulncheck@latest ./...
cd ops/costguard && gofmt -l . && go vet ./... && go test ./...
cd ios/InterviewHackKit && swift test
```

The same gates run in `.github/workflows/ci-backend.yml`, `ci-costguard.yml` and `ci-ios.yml`; production deploys run through `cd-production.yml` (manual approval). See README "CI/CD".

---

## Quality & Rules

### Static Analysis

- Backend: `gofmt`, `go vet`, `scripts/arch_check.sh`
- iOS: compiler warnings treated as issues

### Formatting

- Format code before commit
- Add meaningful comments to exported APIs

### Migrations

- Always provide idempotent DB migrations

---

## Deployment & Workflow

**The source of truth is DesignDocument/14_CICDDesign.md. If there is a conflict, update the design first.**

### Deployment Order

1. Migrations (backward compatible, idempotent)
2. Backend (Cloud Run)
3. Smoke tests

Production requires manual approval in CI. The iOS app is installed from Xcode to the developer's device in phase 1.

### Rollbacks

- Backend: Cloud Run previous revision
- iOS: reinstall the previous build from Xcode

---

## Security Considerations

**The source of truth is DesignDocument/9_SecurityDesign.md. App-layer authorization, share-link safety and key management must follow it.**

- Never commit keys or secrets (`DATABASE_URL`, `GoogleService-Info*.plist`)
- Verify Firebase ID tokens (signature, `aud`, `iss`, `exp`) and trust only the verified uid
- Scope every query by `user_id`; return 404 for other users' resources
- Store only the SHA-256 hash of share tokens; serve the share page with noindex / no-store / no-referrer / CSP
- Never log bodies, prompts, pasted responses, tokens or emails
- Sanitize and bound all inputs on the backend

---

## AI Agent Specific

### How to Generate Code

1. Follow architectural constraints
2. Respect unit test requirements
3. Do not violate boundary rules
4. Make small, incremental changes

### When to Ask

- Before modifying API structures
- When adding new third-party integrations

### When Not to Modify

- Secrets or CI/CD tooling without human approval
- Do not collapse layers or move business logic into handlers/infra; follow Clean Architecture (see DesignDocument/7_BackendDesign.md).

---

## Pull Request Guidelines

- Title format: `[frontend] Fix X` / `[backend] Add Y`
- Always run CI before requesting review
- Include related issue/requirement IDs
- Describe which layers you changed (Domain / Usecase / Adapter / Infrastructure) and reference DesignDocument/7_BackendDesign.md

---

## Do/Don’t Summary

### Do

- Add meaningful error messages
- Append unit tests for new logic
- Use structured logs
- Propagate `X-Request-Id`

### Don’t

- Add new tables without schema review
- Push directly to main/master
- Bypass CI checks
- Let the client access the database directly, or add LLM API calls to the BFF

---

## Rules for Design Changes (Mandatory)

- List affected design doc numbers (e.g., 2,6,7,8,9,14)
- Update the design docs first → review → implement → update tests
- Agents **must not implement changes without updating designs**
- If differences appear, **reflect them back into the design docs**

---

### Task Completion Report (Mandatory)

When a task is completed, you must output a report in Markdown format under the `ai_message_log` directory.

The report must include:

- What was implemented
- Why that implementation was chosen
- Issues encountered
- How those issues were resolved
- What remains to be done
- **Completion percentage (%)** and its rationale — break down by design document or component area, stating which parts are **done**, **in-progress**, and **not-yet-started**. The percentage must be grounded in concrete deliverables (files, features, tests) relative to the design specification, not a subjective estimate.

The file name must follow this format:
`YYYYMMDD_HHMMSS_<number>.md`  
(i.e., today’s date + current time + a sequence number)

---

### Documentation Maintenance

Update `README.md` and `AGENTS.md` whenever you think it is necessary.  
In particular, always keep execution commands and setup instructions up to date and reflect any changes immediately.
