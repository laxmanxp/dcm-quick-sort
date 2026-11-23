package main

// A posted native preview resize belongs to exactly one live preview host
// generation. Never let an old callback resize a newly attached COM handler.
func shouldRunNativeViewportResize(postedEpoch, currentEpoch uint64, postedPane, currentPane uintptr, active bool) bool {
 return active && currentPane != 0 && postedPane == currentPane && postedEpoch == currentEpoch
}
