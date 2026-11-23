package main

// pdfToolbarSlot is a control's horizontal placement in the opaque PDF bar.
// Indices match pdfToolbarHandles().
type pdfToolbarSlot struct {
 Index int
 X int32
 Width int32
}

// pdfToolbarPlan fits the entire toolbar in one row at ordinary preview
// widths. Narrow controls share the available width instead of disappearing.
// At extremely small widths optional commands yield to page + rotation.
func pdfToolbarPlan(width int32) []pdfToolbarSlot {
 preferred := [...]int32{30,42,32,48,30,48,48,32,32,30,30,67}
 minimum := [...]int32{22,30,24,28,22,35,35,26,26,22,22,40}
 order := [...]int{0,1,2,3,4,5,6,7,8,9,10,11}
 if width < 454 {
  order = [...]int{0,1,2,4,7,8,11,5,6,9,10,3}
 }
 available := width-10 // five-pixel horizontal margins
 if available <= 0 {return nil}
 included:=make([]int,0,len(order))
 used:=int32(0)
 // Keep the two rotation icons and page navigation. Optional elements are
 // removed from the end only if even their minimum widths cannot fit.
 for _,idx:=range order {
  needed:=minimum[idx]+4
  if used+needed>available {continue}
  included=append(included,idx)
  used+=needed
 }
 if len(included)==0{return nil}
 // Distribute spare room in visual order up to each control's preferred size.
 spare:=available-used
 widths:=make([]int32,len(included))
 for i,idx:=range included {widths[i]=minimum[idx]}
 for spare>0 {
  changed:=false
  for i,idx:=range included {
   if spare==0 {break}
   if widths[i]<preferred[idx] {widths[i]++;spare--;changed=true}
  }
  if !changed {break}
 }
 x:=int32(5)
 slots:=make([]pdfToolbarSlot,0,len(included))
 for i,idx:=range included {
  slots=append(slots,pdfToolbarSlot{Index:idx,X:x,Width:widths[i]})
  x+=widths[i]+4
 }
 return slots
}
