package main

// PDF pixels use the content rectangle BELOW the opaque toolbar. In Fit Width,
// vertical scrolling is top-origin: zero means the page begins immediately
// below the toolbar and negative positions reveal later content. Fit Page and
// manual zoom retain their centered presentation within that same rectangle.
func pdfTopAnchoredPan(pageHeight, viewportHeight, pan int32) int32 {
 if pageHeight<=viewportHeight || viewportHeight<=0 {return 0}
 minimum:=viewportHeight-pageHeight
 if pan<minimum{return minimum}
 if pan>0{return 0}
 return pan
}

func pdfPageTop(contentTop, contentBottom, pageHeight, panY int32, fitPage, fitWidth bool) int32 {
 if fitWidth {return contentTop+pdfTopAnchoredPan(pageHeight,contentBottom-contentTop,panY)}
 height:=contentBottom-contentTop
 if height<0 {height=0}
 return contentTop+(height-pageHeight)/2+panY
}
