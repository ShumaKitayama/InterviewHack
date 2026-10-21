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

Redesign in progress (2026-10-22): `DesignDocument/DirectionChange_20261022.md` records the new direction (iOS SwiftUI + Go BFF on Cloud Run + Neon + Firebase Auth; no LLM API, no Stripe, no Supabase) and the cross-chapter contract (section 2.10). The design docs above are being revised to match it. Until code migration completes, the sections of this file below that describe Supabase, Stripe, credits, OpenAI/Gemini and the React frontend describe the **current code**, not the target design — follow the revised design docs for any new work.

Mandatory rules:

1. Identify and review **at least one** relevant design document before making changes.
2. When implementing, **state which chapter/section you follow** in comments or the PR description.
3. If design and code diverge, **update the design first**, then change the code.
4. **Do not introduce new concepts** (tables/APIs/flows) that are not described in the design.
5. **BFF boundaries, credit state machine, auth policy, RLS, idempotency, and CI/CD** must follow their respective design docs.

Pre-flight checklist (before you start):

- Which design docs (numbers) are relevant?
- Have you confirmed boundaries (Supabase direct vs BFF vs Service Role only)? (8_DBDesign, 5_FrontendDesign)
- Are you preserving Credit Reserve/Commit/Reversed? (6_AuthAccountCahrgingDesign, 7_BackendDesign, 9_SecurityDesign)
- Do you satisfy idempotency (Stripe event.id / external_ref)? (6,7,8,9)
- Are you following the auth policy (Google primary + Email/Password fallback)? (5,6,7,9)
- Are you propagating correlation IDs (X-Request-Id)? (5,7,14)
- Are you following the test strategy? (11_TestDesign)
- Are you respecting deploy/secrets/environment separation? (14_CICDDesign, 9_SecurityDesign)
- Does this change follow Clean Architecture / DDD layers? (DesignDocument/7_BackendDesign.md)

---

# AGENTS.md

## Project Overview

- **Project Name:** InterviewHack
- **Purpose:** AI assistant service for job hunting and career change
- **Architecture:**
  - **Frontend:** Vite + React + PWA (Cloudflare Pages)
  - **Backend:** Go + Gin BFF (Cloud Run)
  - **DB/Auth/Storage:** Supabase
  - **Payments:** Stripe Checkout + Webhook
  - **AI Providers:** OpenAI API / Gemini API
- **Coding Agents Purpose:**
  - Generate implementation code, tests, and migrations
  - Provide refactors and bug fixes consistent with the architecture
  - Respect boundaries, security, and error-handling standards

---

## Setup & Environment

### Prerequisites

- Node.js 20.14.0+
- Go 1.24.5+
- Supabase MCP
- Stripe CLI (for webhook testing) and MCP

### Setup Commands

```bash
# Install frontend deps
npm ci

# Start frontend dev server
npm run dev

# Build frontend
npm run build

# Start backend
go run ./backend

# Run backend tests
go test ./backend/...
```

### Environment Variables

Include (see README for full list):

- Supabase: `SUPABASE_URL`, `SUPABASE_ANON_KEY`
- Stripe: `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`
- AI: `OPENAI_API_KEY`, `GEMINI_API_KEY`

---

## Coding Conventions

### Languages & Standards

- **Go:** idiomatic Go (error handling, structured logging, JSON tags)
- **React/TS:** functional components + Hooks, strict TypeScript
- **Styling:** Tailwind CSS
- **Data Fetch:** TanStack Query
- **Form Validation:** React Hook Form + Zod

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
├─ frontend/
│  ├─ src/
│  │  ├─ app/
│  │  ├─ components/
│  │  └─ features/
├─ backend/
│  ├─ api/
│  ├─ services/
│  └─ db/
└─ infra/
```

---

## Boundaries & Responsibilities

### Supabase vs BFF

- Supabase direct CRUD is allowed for safe tables (profiles, read-only balances)
- **BFF only** for:
  ```
  - generation_jobs
  - credit_balances (write)
  - credit_transactions (write)
  - stripe_events
  ```
  (Always refer to DesignDocument/8_DBDesign.md and DesignDocument/5_FrontendDesign.md for details.)

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

- **Domain** must NOT import: `gin`, `supabase`, `stripe`, `openai`, `gemini`, or any network/DB SDK.
- **Usecase/Application** must NOT depend on Gin contexts or concrete SDKs; depend on interfaces only.
- **Infrastructure** is the only layer allowed to touch DB, network, Stripe, Supabase, OpenAI/Gemini, logging backends.
- Repositories and external clients are defined as **interfaces** in inner layers and implemented in Infrastructure.
- DTO ↔ Usecase Input ↔ Domain ↔ Persistence model conversions happen in **Adapter/Infrastructure**, not in Domain.

### Expected Go Package Layout (example)

```
backend/
  cmd/
    api/                 # main, DI, server bootstrap (Gin)
  internal/
    domain/              # entities, value objects, domain services, domain errors
    usecase/             # application/usecase layer (transaction boundaries, orchestration)
    port/                # interfaces (repositories, external clients)
    adapter/
      http/              # Gin handlers, DTOs, presenters
    infra/
      supabase/          # DB implementations
      stripe/            # payment implementations
      llm/               # OpenAI/Gemini implementations
      logging/
      config/
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

- Frontend: Vitest / React Testing Library
- Backend: Go tests (table-driven where applicable)

### Coverage

- Aim for ≥ 80% coverage on backend critical logic

### CI Validation

Before merge:

```bash
npm run lint
npm run typecheck
npm test
go test ./backend/...
```

---

## Quality & Rules

### Static Analysis

- Frontend: ESLint + Prettier
- Backend: `go vet` + linter

### Formatting

- Format code before commit
- Add meaningful comments to exported APIs

### Migrations

- Always provide idempotent DB migrations

---

## Deployment & Workflow

**The source of truth is DesignDocument/14_CICDDesign.md. If there is a conflict, update the design first.**

### Deployment Order

1. Backend → migrations
2. Frontend
3. Smoke tests

Production requires manual approval in CI.

### Rollbacks

- Backend: Cloud Run previous revision
- Frontend: Cloudflare Pages re-deploy previous build

---

## Security Considerations

**The source of truth is DesignDocument/9_SecurityDesign.md. Webhook idempotency, RLS, and key management must follow it.**

- Never commit keys or secrets
- Validate Stripe webhook signatures
- Enforce RLS in Supabase
- Sanitize all inputs on the backend

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
- Write to BFF-only tables from the client

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
