package main

// pdfPageTop returns the page's y-coordinate relative to the PDF content area.
// The toolbar is excluded before this calculation. Fit Width starts directly
// below it; Fit Page remains centered within the space below the toolbar.
// Manual zoom keeps its existing centered/pan behavior.
func pdfPageTop(contentTop, contentBottom, pageHeight, panY int32, fitPage,fitWidth bool) int32 {
 height:=contentBottom-contentTop
 if height<0 {height=0}
 if fitWidth {
  // Existing wheel pan uses centered coordinates; subtract the nominal
  // top-edge pan so an initial fit-width page always begins at contentTop.
  topPan:=previewTopPanOffset(pageHeight,height-8)
  return contentTop+4+panY-topPan
 }
 return contentTop+(height-pageHeight)/2+panY
}
