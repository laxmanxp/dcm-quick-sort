package main

// Rotation is a quarter-turn count, clockwise from the source orientation.
// It describes only the on-screen preview; files are never rewritten.
func rotatePreviewQuarterTurns(current, delta int) int {
	turns := (current + delta) % 4
	if turns < 0 {
		turns += 4
	}
	return turns
}

func rotatedPreviewDimensions(width, height int32, turns int) (int32, int32) {
	if rotatePreviewQuarterTurns(turns, 0)%2 != 0 {
		return height, width
	}
	return width, height
}

type previewRotationPoint struct{ X, Y int32 }

// GDI+ maps source top-left, top-right, bottom-left onto these three
// destination points. This rotates the drawn image, not the source bitmap.
func previewRotationPoints(x, y, width, height int32, turns int) [3]previewRotationPoint {
	switch rotatePreviewQuarterTurns(turns, 0) {
	case 1: // clockwise
		return [3]previewRotationPoint{{x + width, y}, {x + width, y + height}, {x, y}}
	case 2:
		return [3]previewRotationPoint{{x + width, y + height}, {x, y + height}, {x + width, y}}
	case 3: // counter-clockwise
		return [3]previewRotationPoint{{x, y + height}, {x, y}, {x + width, y + height}}
	default:
		return [3]previewRotationPoint{{x, y}, {x + width, y}, {x, y + height}}
	}
}
