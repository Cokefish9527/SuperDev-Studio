# Auto Dispatch Loop - Execution Report (2026-03-10)

## Goal

Close the next phase of the autonomous delivery loop by adding structured `next_command` decisions plus a safe `auto-advance` API.

## Delivered

### 1. Structured next-step dispatch

- Added `next_command` to persisted `agent_evaluations` records.
- Threaded `next_command` through:
  - store models
  - SQLite schema backfill
  - Eino runtime evaluation payloads
  - Windows fallback runtime
  - API responses
- The evaluator now works with a controlled next-command vocabulary instead of free-form shell text.

### 2. Safe auto-advance API

- Added `POST /api/pipeline/runs/{runID}/auto-advance`.
- The endpoint synchronizes residual items, approval gates, and preview sessions before deciding what to do.
- Safe behaviors now include:
  - rerun terminal deliveries when the latest decision is `rerun_delivery`
  - block on open approval gates
  - block on `awaiting_human`
  - block on pending preview review
  - return noop/complete when no safe follow-up execution is required
- The endpoint never auto-approves risky actions.

### 3. Follow-up reuse of agent dispatch intent

- Residual suggested commands now prefer evaluator `next_command` mappings.
- Failed runs now guide users toward the unified `auto-advance` endpoint instead of scattered retry-only actions.

### 4. Pipeline UI updates

- Added `next_command` to frontend types and API client.
- `PipelinePage` now shows dispatch intent in evaluation summaries.
- Added an `Auto advance` action for completed/failed runs so operators can trigger the next safe step from the UI.

### 5. Test coverage

- Store persistence test now verifies `next_command` round-trip.
- Backend API tests cover:
  - safe rerun on failed runs
  - preview-review blocking
  - approval-gate blocking
  - `await_human` exposure from agent evaluation
- Frontend `PipelinePage` regression tests continue passing after the new action path.

## Validation

### Backend

Executed:

- `go test ./internal/store ./internal/api -run 'AutoAdvance|Preview|Residual|Approval|NeedHuman|Requirement|Completion'`
- `go test ./internal/agentruntime/... ./internal/pipeline -run TestDoesNotExist`

Result: passed.

### Frontend

Executed:

- `npm test -- src/pages/PipelinePage.test.tsx`
- `npm run build`

Result: passed.

### Super Dev flow

Executed:

- `python C:/Users/bmkz/.codex/skills/super-dev/scripts/super_dev_status.py --root .`
- `super-dev task status auto-dispatch-loop`
- `super-dev task run auto-dispatch-loop`
- `super-dev quality --type all`
- `cmd /c "echo y| super-dev spec archive auto-dispatch-loop"`

Result:

- task run completed with `12/12`
- quality gate passed with `80/100`
- change archived to `.super-dev/archive/auto-dispatch-loop/`

## Key files

- `backend/internal/agentruntime/runtime.go`
- `backend/internal/agentruntime/eino/runtime.go`
- `backend/internal/agentruntime/eino/runtime_stub_windows_amd64_go124.go`
- `backend/internal/api/auto_advance.go`
- `backend/internal/api/followups.go`
- `backend/internal/api/server.go`
- `backend/internal/api/server_test.go`
- `backend/internal/pipeline/manager_agent_helpers.go`
- `backend/internal/pipeline/manager_fullcycle_helpers.go`
- `backend/internal/store/agent_runtime.go`
- `backend/internal/store/models.go`
- `backend/internal/store/store.go`
- `backend/internal/store/store_test.go`
- `frontend/src/api/client.ts`
- `frontend/src/pages/PipelinePage.tsx`
- `frontend/src/pages/PipelinePage.test.tsx`
- `frontend/src/types.ts`

## Remaining broader roadmap items

This change closes the automatic dispatch loop foundation, but the larger product roadmap still has follow-on work:

1. Merge the same auto-advance loop into the simplified user-facing delivery page.
2. Let the evaluator continuously re-assess residuals after each new run and keep dispatching until completion.
3. Surface a simpler final acceptance view for requirement -> delivery -> preview -> release.
