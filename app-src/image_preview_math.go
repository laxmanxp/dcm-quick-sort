package main

// imageZoomAnchorPan keeps a document point under the pointer during a zoom.
// All coordinates are in preview-client pixels and pan is measured from the
// center of the content rectangle. The function is independent of GDI+.
func imageZoomAnchorPan(oldW, oldH, newW, newH, viewW, viewH, pointerX, pointerY, panX, panY int32) (int32,int32) {
 if oldW<=0 || oldH<=0 || newW<=0 || newH<=0 || viewW<=0 || viewH<=0 {return 0,0}
 left:=(viewW-oldW)/2+panX
 top:=(viewH-oldH)/2+panY
 fx:=float64(pointerX-left)/float64(oldW)
 fy:=float64(pointerY-top)/float64(oldH)
 if fx<0 || fx>1 || fy<0 || fy>1 {return clampPreviewPan(newW,viewW,panX),clampPreviewPan(newH,viewH,panY)}
 x:=int32(float64(pointerX)-float64(viewW-newW)/2-fx*float64(newW))
 y:=int32(float64(pointerY)-float64(viewH-newH)/2-fy*float64(newH))
 return clampPreviewPan(newW,viewW,x),clampPreviewPan(newH,viewH,y)
}

// Rotation applies to the bitmap held for the preview, never the disk file.
func nextImagePreviewRotation(current,step int)int{
 value:=(current+step)%360
 if value<0{value+=360}
 return value
}
