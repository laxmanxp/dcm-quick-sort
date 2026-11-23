package main

// pdfViewportNeedsRerender compares the effective render viewport rather
// than the outer window rectangle. Identical dimensions must not spawn work.
func pdfViewportNeedsRerender(active bool, oldWidth,oldHeight,newWidth,newHeight int) bool {
 return active && newWidth>0 && newHeight>0 &&
  (oldWidth!=newWidth || oldHeight!=newHeight)
}
