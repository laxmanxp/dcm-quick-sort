package main

// The splitter is intentionally session-local: dragging does not modify
// production settings, file metadata, queue selections or preview state.
var queueSplitterWidth int32
var queueSplitterDragging bool

func queueSplitterClamp(width, requested int32) int32 {
 if width < 760 {width=760}
 max:=width-470 // reserve the existing 420px preview + 50px margins
 if max<260 {max=260}
 if requested<260 {return 260}
 if requested>max {return max}
 return requested
}
func queueSplitterDefault(width int32) int32 {
 list:=width*35/100
 if list<260 {list=260}
 if list>460 {list=460}
 return queueSplitterClamp(width,list)
}
func queueSplitterPosition(width int32) int32 {
 if queueSplitterWidth==0 {return queueSplitterDefault(width)}
 return queueSplitterClamp(width,queueSplitterWidth)
}
