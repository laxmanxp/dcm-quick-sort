package main

// PDFViewMode is a viewer navigation policy; it does not alter the PDF file.
type PDFViewMode string

const (
 PDFViewSingle PDFViewMode = "single"
 PDFViewContinuous PDFViewMode = "continuous"
)

func normalizePDFViewMode(mode PDFViewMode) PDFViewMode {
 if mode==PDFViewContinuous {return PDFViewContinuous}
 return PDFViewSingle
}

// pdfScrollStep determines how a scroll gesture moves across the current
// bitmap. A new page is requested only after reaching the page boundary;
// remaining delta is deliberately not carried into another unrendered page.
func pdfScrollStep(page, count int, position, viewport, content, delta int32) (nextPage int, nextPosition int32) {
 nextPage=int(page)
 if count<=0 || viewport<=0 || content<=0 {return nextPage,0}
 maxScroll:=content-viewport
 if maxScroll<0 {maxScroll=0}
 target:=int64(position)+int64(delta)
 if target<0 {
  if page>0 {return page-1,-1}
  return page,0
 }
 if target>int64(maxScroll) {
  if page+1<count {return page+1,0}
  return page,maxScroll
 }
 return page,int32(target)
}
