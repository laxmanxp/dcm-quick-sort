//go:build windows

package main

import (
	"math"
	"unsafe"
)

// Two native owner-drawn rotation icons. Drawing the curved arrows using
// GDI lines avoids relying on a font that may render Unicode arrows as '?'.
// The BUTTON window title retains a useful screen-reader/accessibility name.
var (
	procPDFIconMoveTo = gdi32.NewProc("MoveToEx")
	procPDFIconLineTo = gdi32.NewProc("LineTo")
)

func drawPDFRotateIconButton(dis *DRAWITEMSTRUCT, clockwise bool) {
	if dis == nil || dis.HDC == 0 {
		return
	}
	rc := dis.RcItem
	fill, border, stroke := rgb(250, 252, 255), rgb(176, 192, 211), colorAccent
	if dis.ItemState&ODS_SELECTED != 0 {
		fill = rgb(218, 230, 249)
		border = colorAccent
	}
	if dis.ItemState&ODS_DISABLED != 0 {
		fill, border, stroke = rgb(243, 245, 247), rgb(206, 212, 219), rgb(160, 170, 181)
	}

	brush, _, _ := procCreateSolidBrush.Call(fill)
	framePen, _, _ := procCreatePen.Call(PS_SOLID, 1, border)
	oldBrush, _, _ := procSelectObject.Call(dis.HDC, brush)
	oldPen, _, _ := procSelectObject.Call(dis.HDC, framePen)
	procRoundRect.Call(dis.HDC, uintptr(rc.Left), uintptr(rc.Top), uintptr(rc.Right), uintptr(rc.Bottom), 8, 8)
	procSelectObject.Call(dis.HDC, oldBrush)
	procSelectObject.Call(dis.HDC, oldPen)
	procDeleteObject.Call(brush)
	procDeleteObject.Call(framePen)

	iconPen, _, _ := procCreatePen.Call(PS_SOLID, 2, stroke)
	oldPen, _, _ = procSelectObject.Call(dis.HDC, iconPen)
	// Use the same vector icon geometry as conventional curved-arrow
	// controls. Screen Y grows downward: increasing angles are clockwise.
	// Mirroring horizontally produces the counterclockwise twin.
	scale := math.Min(float64(rc.Right-rc.Left-8)/24, float64(rc.Bottom-rc.Top-4)/24)
	if scale < 0.25 {
		scale = 0.25
	}
	originX := float64(rc.Left+rc.Right)/2 - 12*scale
	originY := float64(rc.Top+rc.Bottom)/2 - 12*scale
	pressed := 0.0
	if dis.ItemState&ODS_SELECTED != 0 {
		pressed = 1
	}
	point := func(x, y float64) (uintptr, uintptr) {
		if !clockwise { x = 24 - x }
		return uintptr(int32(math.Round(originX+x*scale+pressed))),
			uintptr(int32(math.Round(originY+y*scale+pressed)))
	}
	line := func(fromX, fromY, toX, toY float64) {
		x1, y1 := point(fromX, fromY)
		x2, y2 := point(toX, toY)
		procPDFIconMoveTo.Call(dis.HDC, x1, y1, 0)
		procPDFIconLineTo.Call(dis.HDC, x2, y2)
	}
	// Arc from lower right around the outside to the arrow tip (21,11).
	// This follows ~312 degrees of a circle, leaving a readable break.
	first := true
	var lastX, lastY float64
	for angle := 42.0; angle <= 354; angle += 12 {
		radians := angle * math.Pi / 180
		x, y := 12+9*math.Cos(radians), 12+9*math.Sin(radians)
		if !first { line(lastX,lastY,x,y) }
		first = false
		lastX,lastY = x,y
	}
	line(lastX,lastY,21,11)
	line(21,3,21,11)
	line(13,11,21,11)
	procSelectObject.Call(dis.HDC, oldPen)
	procDeleteObject.Call(iconPen)

	if dis.ItemState&ODS_FOCUS != 0 {
		focus := RECT{Left:rc.Left+3,Top:rc.Top+3,Right:rc.Right-3,Bottom:rc.Bottom-3}
		procDrawFocusRect.Call(dis.HDC,uintptr(unsafe.Pointer(&focus)))
	}
}
