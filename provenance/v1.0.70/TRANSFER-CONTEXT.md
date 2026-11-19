# DCM Quick Sort v1.0.70 transfer context

## Baseline
Built from the exact qualified v1.0.69 Source-and-Installer archive.

## Change
v1.0.70 adds deterministic Batch Undo. Each successful multi-select batch move receives a durable `batch_id`. The Processed pane exposes `Undo Batch`, which targets the most recent still-undoable batch.

## Safety model
Before changing any file, Batch Undo preflights every record in the batch. It then unwinds newest-first. Individual move journal entries are marked undone after their files are restored. Existing single-record Undo remains unchanged.

## Deferred
No new learning, AI, prediction, or smart destination/rename behavior was added. Smartness is intentionally deferred.

## Next deterministic improvement
Conflict-handling UX/policy refinement.
