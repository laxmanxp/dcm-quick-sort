//go:build windows

package main

import (
	"testing"
	"unsafe"
)

func TestContextMenuRejectsInvalidTargets(t *testing.T) {
	for _, c := range []struct {
		name       string
		count, hit int
		inside     bool
		want       int
	}{
		{"first-row", 2, 0, true, 0}, {"second-row", 2, 1, true, 1},
		{"empty", 0, 0, true, -1}, {"negative", 2, -1, true, -1},
		{"past-last", 2, 2, true, -1}, {"blank-space", 2, 1, false, -1},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := qsContextIndex(c.count, c.hit, c.inside); got != c.want {
				t.Fatalf("got %d want %d", got, c.want)
			}
		})
	}
	rect := RECT{Left: 0, Top: 20, Right: 180, Bottom: 40}
	for _, c := range []struct {
		p      POINT
		inside bool
	}{
		{POINT{X: 8, Y: 30}, true}, {POINT{X: 0, Y: 20}, true},
		{POINT{X: 180, Y: 30}, false}, {POINT{X: 8, Y: 40}, false},
		{POINT{X: -1, Y: 30}, false}, {POINT{X: 8, Y: 41}, false},
	} {
		if got := qsContextContains(rect, c.p); got != c.inside {
			t.Fatalf("point=%+v inside=%v want %v", c.p, got, c.inside)
		}
	}
}
func TestContextMenuCancelIsNotAnOpenCommand(t *testing.T) {
	for _, c := range []struct {
		command, offset uintptr
		selected        bool
	}{
		{0, 0, false}, {1, 0, true}, {2, 1, true}, {0x7FFF, 0x7FFE, true}, {0x8000, 0, false},
	} {
		got, ok := qsContextOffset(c.command)
		if got != c.offset || ok != c.selected {
			t.Fatalf("command=%d got %d/%v want %d/%v", c.command, got, ok, c.offset, c.selected)
		}
	}
}
func TestContextMenuInvokeABI(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("release native acceptance target is Windows x64")
	}
	var c qsInvokeCommand
	if unsafe.Sizeof(c) != 104 || unsafe.Offsetof(c.VerbW) != 64 || unsafe.Offsetof(c.Point) != 96 {
		t.Fatalf("incorrect Windows x64 CMINVOKECOMMANDINFOEX layout: size=%d verbW=%d point=%d", unsafe.Sizeof(c), unsafe.Offsetof(c.VerbW), unsafe.Offsetof(c.Point))
	}
}
