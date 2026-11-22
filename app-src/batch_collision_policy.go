package main

// Batch collisions never remove a duplicate or replace an occupied destination.
// The decision applies only to the currently inspected file unless cancelled.
type batchCollisionDecision uint8

const (
	batchCollisionCancel batchCollisionDecision = iota
	batchCollisionRename
	batchCollisionSkip
)

// Windows MessageBox Yes/No/Cancel return codes. Dismissal is Cancel.
func batchCollisionChoice(button int) batchCollisionDecision {
	switch button {
	case 6: // IDYES
		return batchCollisionRename
	case 7: // IDNO
		return batchCollisionSkip
	default:
		return batchCollisionCancel
	}
}

// On cancellation, every unstarted item must count as not moved. Previously
// completed transactions remain independently journaled and undoable.
func batchCancelledRemainder(completed, issues, remaining int) (int, int) {
	if remaining < 0 {
		remaining = 0
	}
	return completed + remaining, issues + remaining
}
