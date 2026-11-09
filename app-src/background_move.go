package main

type BackgroundMoveStatus string

const (
	BackgroundMoveSucceeded            BackgroundMoveStatus = "succeeded"
	BackgroundMoveCopiedSourceRetained BackgroundMoveStatus = "copied-source-retained"
	BackgroundMoveFailed               BackgroundMoveStatus = "failed"
)

type BackgroundMoveResult struct {
	Source string
	Target string
	Status BackgroundMoveStatus
	Record MoveRecord
	Err    error
}

// executeBackgroundCopyMove performs the cross-volume fallback as two explicit
// stages. A failed copy never removes the source. If the copy succeeds but the
// source cannot be removed, the destination copy is intentionally retained so
// user data is never discarded merely to make the operation look atomic.
func executeBackgroundCopyMove(source, target string, copyFn func(string, string) error, removeFn func(string) error) BackgroundMoveResult {
	result := BackgroundMoveResult{Source: source, Target: target}
	if err := copyFn(source, target); err != nil {
		result.Status = BackgroundMoveFailed
		result.Err = err
		return result
	}
	if err := removeFn(source); err != nil {
		result.Status = BackgroundMoveCopiedSourceRetained
		result.Err = err
		return result
	}
	result.Status = BackgroundMoveSucceeded
	result.Record = MoveRecord{From: source, To: target, Kind: MoveKindMove}
	return result
}

// startBackgroundCopyMove deliberately returns a buffered result channel
// immediately. The worker may take minutes; callers can keep their event loop
// responsive while waiting for completion.
func startBackgroundCopyMove(source, target string, copyFn func(string, string) error, removeFn func(string) error) <-chan BackgroundMoveResult {
	results := make(chan BackgroundMoveResult, 1)
	go func() {
		results <- executeBackgroundCopyMove(source, target, copyFn, removeFn)
		close(results)
	}()
	return results
}

type BackgroundMoveProgress struct {
	Source string
	Target string
	Copied int64
	Total  int64
}

// startBackgroundCopyMoveWithProgress is the observable Go 9C variant of the
// fallback worker. Progress callbacks describe copy bytes only; source removal
// still occurs only after the verified copy completes successfully.
func startBackgroundCopyMoveWithProgress(source, target string, copyFn func(string, string, CopyProgressFunc) error, removeFn func(string) error, progressFn CopyProgressFunc) <-chan BackgroundMoveResult {
	results := make(chan BackgroundMoveResult, 1)
	go func() {
		result := BackgroundMoveResult{Source: source, Target: target}
		if err := copyFn(source, target, progressFn); err != nil {
			result.Status = BackgroundMoveFailed
			result.Err = err
			results <- result
			close(results)
			return
		}
		if err := removeFn(source); err != nil {
			result.Status = BackgroundMoveCopiedSourceRetained
			result.Err = err
			results <- result
			close(results)
			return
		}
		result.Status = BackgroundMoveSucceeded
		result.Record = MoveRecord{From: source, To: target, Kind: MoveKindMove}
		results <- result
		close(results)
	}()
	return results
}
