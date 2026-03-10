# Residual Backlog Shrink Loop - Execution Report (2026-03-10)

## Phase goal

Let the latest pipeline run automatically re-evaluate unresolved historical residual items inside the same change batch so the backlog collapses onto the newest run instead of accumulating across retries.

## What was delivered

### 1. Change-batch historical residual query

- Added `ListOpenChangeBatchResidualItems(...)` in `backend/internal/store/followups.go`.
- The query joins `residual_items` with `pipeline_runs` so the backend can find open historical residuals for one `change_batch_id` while excluding the current latest run.
- This keeps the reconciliation logic focused on the active delivery loop instead of the whole project backlog.

### 2. Latest-run-only backlog reconciliation

- `syncRunFollowups(...)` now triggers `reconcileHistoricalResidualBacklog(...)`.
- Reconciliation only runs when:
  - the run belongs to a change batch
  - the run is terminal-like (`completed`, `failed`, `awaiting_human`)
  - the run is the `change_batch.latest_run_id`
- Historical sync-generated residual items are then processed as follows:
  - fixed issues -> auto-resolved with a backlog re-evaluation note
  - repeated issues -> old residual closed as superseded, newest run keeps the active open item

### 3. Timeline visibility for autonomous reassessment

- The latest run now writes a `backlog-reconcile` event summarizing how many historical items were resolved and how many were carried forward.
- This lets users see that the Agent is not only retrying, but also actively shrinking and consolidating backlog state.

## Validation

### Backend tests

Executed:

- `go test ./internal/api ./internal/store -run "ResidualBacklog|Followups|ListOpenChangeBatchResidualItems|AutoAdvance|RequirementSessionSync"`

Result:

- Passed.

### Super Dev pipeline

Executed:

- `super-dev task status residual-backlog-shrink-loop`
- `super-dev task run residual-backlog-shrink-loop`
- `super-dev quality --type all`

Result:

- task run: `5/5` completed
- quality gate: `87/100` passed

## Files changed in this phase

- `backend/internal/api/followups.go`
- `backend/internal/api/followups_reconcile_test.go`
- `backend/internal/store/followups.go`
- `backend/internal/store/auto_advance_test.go`
- `docs/AGENT_CONFIRMED_DELIVERY_LOOP_PROGRESS.md`
- `output/residual-backlog-shrink-loop-execution-report.md`

## What this unlocks

- historical residual items no longer pile up across retry runs
- the latest run becomes the single source of truth for unresolved delivery work
- the autonomous loop can make cleaner follow-up decisions because stale backlog items are closed automatically

## Next recommended slice

1. Add a final acceptance / pre-release handoff view that combines preview decision, quality outcome, and release package status.
2. Surface background auto-advance and backlog shrink progress more clearly in the simplified delivery page.
3. Optionally add a change-batch backlog summary API/card if you want users to review historical shrink trends directly.
