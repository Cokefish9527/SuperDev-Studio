# Background Auto Advance Worker - Execution Report (2026-03-10)

## Goal

Make safe auto-advance persistent on the server so delivery can continue even when no page is open.

## Delivered

### 1. Persistent background auto-advance worker

- Added `backend/internal/api/auto_advance_worker.go`.
- The worker periodically scans leaf terminal runs and reuses the existing safe `autoAdvancePipeline(...)` decision path.
- This keeps all dispatch rules in one place instead of duplicating retry / preview / approval logic.

### 2. Backend app lifecycle and config

- Extended `backend/internal/app/app.go` with background worker settings:
  - `SUPERDEV_STUDIO_AUTO_ADVANCE_WORKER_ENABLED`
  - `SUPERDEV_STUDIO_AUTO_ADVANCE_WORKER_INTERVAL`
  - `SUPERDEV_STUDIO_AUTO_ADVANCE_WORKER_BATCH_SIZE`
- The backend now starts the worker automatically during app runtime.
- Default behavior is enabled so the product is closer to the intended autonomous mode out of the box.

### 3. Store support for persistent dispatch reconciliation

- Added `ListAutoAdvanceCandidateRuns(...)` to scan terminal leaf runs only.
- Added `TouchPipelineRun(...)` so preview or residual changes can move a run back to the front of the reconciliation queue.
- Added `SyncRequirementSessionsLatestRunByChangeBatch(...)` so a newly created retry run updates the requirement session pointer used by the simplified delivery flow.

### 4. Change-batch to requirement-session latest run sync

- Updated pipeline change-batch touch logic so every new run also updates related requirement sessions.
- This means a user can close the page, come back later, and still land on the current retry run instead of the stale failed one.

### 5. Auto-advance audit trail and simple UX copy

- Rerun dispatch now records an event on the source run as well as the new run.
- Updated `SimpleDeliveryPage` copy to explicitly state that the backend keeps auto-advancing even after the page is closed.

## Validation

### Backend

Executed:

- `gofmt -w internal/api/auto_advance.go internal/api/auto_advance_worker.go internal/api/auto_advance_worker_test.go internal/app/app.go internal/app/app_test.go internal/pipeline/manager_superdev_helpers.go internal/store/store.go internal/store/preview_sessions.go internal/store/followups.go internal/store/auto_advance_test.go`
- `go test ./internal/api ./internal/store ./internal/app -run 'AutoAdvance|RequirementSessionSync|LoadConfigReadsAutoAdvanceWorkerEnv'`
- `go test ./internal/pipeline -run TestDoesNotExist`

Result: passed.

### Frontend

Executed:

- `npm test -- src/pages/SimpleDeliveryPage.test.tsx`
- `npm run build`

Result: passed.

### Super Dev flow

Executed:

- `python C:/Users/bmkz/.codex/skills/super-dev/scripts/super_dev_status.py --root .`
- `super-dev task status background-auto-advance-worker`
- `super-dev task run background-auto-advance-worker`
- `super-dev quality --type all`
- `cmd /c "echo y| super-dev spec archive background-auto-advance-worker"`
- `python C:/Users/bmkz/.codex/skills/super-dev/scripts/super_dev_status.py --root .`

Result:

- task run completed with `4/4`
- quality gate passed with `87/100`
- change archived to `.super-dev/archive/background-auto-advance-worker/`
- workspace returned to `Active change_id: none`

## Key files

- `backend/internal/api/auto_advance.go`
- `backend/internal/api/auto_advance_worker.go`
- `backend/internal/api/auto_advance_worker_test.go`
- `backend/internal/app/app.go`
- `backend/internal/app/app_test.go`
- `backend/internal/pipeline/manager_superdev_helpers.go`
- `backend/internal/store/store.go`
- `backend/internal/store/preview_sessions.go`
- `backend/internal/store/followups.go`
- `backend/internal/store/auto_advance_test.go`
- `frontend/src/pages/SimpleDeliveryPage.tsx`
- `output/background-auto-advance-worker-execution-report.md`

## Product impact

The delivery loop is now materially closer to the target autonomous workflow:

1. User confirms the requirement draft.
2. Delivery starts.
3. Safe follow-up dispatch no longer depends on the page remaining open.
4. If a failed leaf run should retry, the backend can create the next run automatically.
5. When the user returns, the simplified page can resolve to the latest run through the synced requirement session pointer.
