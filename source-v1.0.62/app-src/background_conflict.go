package main

type BackgroundConflictResult struct {
	Source string
	Target string
	Info   DestinationConflictInfo
	Err    error
}

func startBackgroundConflictInspection(source, target string, inspectFn func(string, string) (DestinationConflictInfo, error)) <-chan BackgroundConflictResult {
	results := make(chan BackgroundConflictResult, 1)
	go func() {
		info, err := inspectFn(source, target)
		results <- BackgroundConflictResult{Source: source, Target: target, Info: info, Err: err}
		close(results)
	}()
	return results
}

type BackgroundConflictActionKind string

const (
	BackgroundConflictReplace         BackgroundConflictActionKind = "replace"
	BackgroundConflictRemoveDuplicate BackgroundConflictActionKind = "remove-duplicate"
)

type BackgroundConflictActionResult struct {
	Kind   BackgroundConflictActionKind
	Source string
	Target string
	Record MoveRecord
	Err    error
}

func startBackgroundConflictAction(kind BackgroundConflictActionKind, source, target string, actionFn func() (MoveRecord, error)) <-chan BackgroundConflictActionResult {
	results := make(chan BackgroundConflictActionResult, 1)
	go func() {
		rec, err := actionFn()
		results <- BackgroundConflictActionResult{Kind: kind, Source: source, Target: target, Record: rec, Err: err}
		close(results)
	}()
	return results
}
