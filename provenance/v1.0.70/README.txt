DCM - Quick Sort v1.0.70

Focus: deterministic Batch Undo.

New in 1.0.70:
- Batch moves carry a durable batch identifier in the move journal.
- Processed pane has a dedicated Undo Batch action.
- Undo Batch restores the latest completed batch newest-first.
- The whole batch is preflighted before the first filesystem change.
- Individual Undo remains available for one processed transaction.
- Batch identity survives restart through the durable journal.

Build target: Windows x64, Go 1.23.x compatible, CGO disabled.
