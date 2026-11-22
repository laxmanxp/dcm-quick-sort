package main

// pdfToolbarSlot describes the position of one toolbar control in the
// fixed-height PDF toolbar. Indices match pdfToolbarHandles().
type pdfToolbarSlot struct {
 Index int
 X int32
 Width int32
}

// pdfToolbarPlan is independent of HWNDs so the actual small-preview layout
// can be tested even when the Windows top-level minimum width prevents it.
func pdfToolbarPlan(width int32) []pdfToolbarSlot {
 widths := [...]int32{30,42,32,48,30,48,48,32,32,30,30}
 order := [...]int{0,1,2,3,4,5,6,7,8,9,10}
 if width < 454 {
  // Page back/edit/Go/next, then rotate left/right. Decoration and
  // optional fit/zoom controls yield space first.
  order = [...]int{0,1,2,4,7,8,5,6,9,10,3}
 }
 x := int32(5)
 slots := make([]pdfToolbarSlot,0,len(widths))
 for _,index := range order {
  w:=widths[index]
  if x+w+4>width { continue }
  slots=append(slots,pdfToolbarSlot{Index:index,X:x,Width:w})
  x+=w+4
 }
 return slots
}
