package main

// Geometry is in the existing host's logical coordinates. Recompute from the
// current client width; never retain a previous window/DPI/handle's rectangles.
type compactRect struct{ X, Y, W, H int32 }
type filterStripLayout struct {
	Labels, Fields [5]compactRect
	Clear          compactRect
	HeaderY        int32
	Rows           int
}

func compactFilterLayout(width int32) filterStripLayout {
	const margin, gap, labelGap, rowY, rowStep = int32(20), int32(14), int32(6), int32(82), int32(34)
	labels := [5]int32{80, 80, 40, 40, 64}
	fields := [5]int32{160, 150, 160, 135, 260}
	const clearWidth = int32(110)
	needed := clearWidth + 5*gap
	for i := range labels {
		needed += labels[i] + labelGap + fields[i]
	}
	result := filterStripLayout{Rows: 1, HeaderY: 120}
	if width-2*margin >= needed {
		x := margin
		for i := range labels {
			result.Labels[i] = compactRect{x, rowY + 5, labels[i], 20}
			result.Fields[i] = compactRect{x + labels[i] + labelGap, rowY, fields[i], 28}
			x += labels[i] + labelGap + fields[i] + gap
		}
		result.Clear = compactRect{width - margin - clearWidth, rowY - 1, clearWidth, 28}
		return result
	}
	result.Rows, result.HeaderY = 2, 154
	column := (width - 2*margin - 2*gap) / 3
	for i := 0; i < 3; i++ {
		x := margin + int32(i)*(column+gap)
		result.Labels[i] = compactRect{x, rowY + 5, labels[i], 20}
		result.Fields[i] = compactRect{x + labels[i] + labelGap, rowY, column - labels[i] - labelGap, 28}
	}
	secondY := rowY + rowStep
	result.Labels[3] = compactRect{margin, secondY + 5, labels[3], 20}
	result.Fields[3] = compactRect{margin + labels[3] + labelGap, secondY, column - labels[3] - labelGap, 28}
	x := margin + column + gap
	result.Labels[4] = compactRect{x, secondY + 5, labels[4], 20}
	result.Clear = compactRect{width - margin - clearWidth, secondY - 1, clearWidth, 28}
	fieldX := x + labels[4] + labelGap
	result.Fields[4] = compactRect{fieldX, secondY, result.Clear.X - gap - fieldX, 28}
	return result
}

type processedHeaderLayout struct {
	Title, UndoBatch, Undo, Search, List compactRect
}

func compactProcessedLayout(x, width, headerY, previewHeight int32) processedHeaderLayout {
	// Title/count has its own line. Buttons never subtract from its text width.
	top := headerY + 28
	return processedHeaderLayout{
		Title:     compactRect{x, headerY, width, 24},
		UndoBatch: compactRect{x + width - 164, top, 88, 27},
		Undo:      compactRect{x + width - 68, top, 68, 27},
		Search:    compactRect{x, top + 34, width, 25},
		List:      compactRect{x, top + 67, width, previewHeight - 67},
	}
}
