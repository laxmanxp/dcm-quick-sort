package main

import "testing"

func compactIntersects(a, b compactRect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

func TestCompactFilterLayout(t *testing.T) {
	for _, width := range []int32{760, 1024, 1120, 1184, 1264, 1400, 1440, 1920} {
		v := compactFilterLayout(width)
		all := []compactRect{v.Clear}
		for i := range v.Fields {
			a, b := v.Labels[i], v.Fields[i]
			if a.X+a.W+6 != b.X || a.Y != b.Y+5 || b.W < 125 {
				t.Fatalf("width%d field%d bad label/field geometry: %+v %+v", width, i, a, b)
			}
			all = append(all, a, b)
		}
		if v.Fields[4].W < 250 {
			t.Fatalf("long sort value too narrow at %d: %d", width, v.Fields[4].W)
		}
		for i, a := range all {
			if a.X < 20 || a.X+a.W > width-20 || a.Y+a.H > v.HeaderY-4 {
				t.Fatalf("control escapes strip at %d: %+v header=%d", width, a, v.HeaderY)
			}
			for _, b := range all[i+1:] {
				if compactIntersects(a, b) {
					t.Fatalf("overlap at %d: %+v %+v", width, a, b)
				}
			}
		}
		if width <= 1264 && v.Rows != 2 || width >= 1440 && v.Rows != 1 {
			t.Fatalf("wrong wrapping at %d", width)
		}
	}
	if compactFilterLayout(1120) != compactFilterLayout(1120) {
		t.Fatal("layout is not deterministic")
	}
}

func TestCompactProcessedHeader(t *testing.T) {
	for _, w := range []int32{200, 228, 260, 320} {
		for _, y := range []int32{120, 154} {
			p := compactProcessedLayout(900, w, y, 300)
			items := []compactRect{p.Title, p.UndoBatch, p.Undo, p.Search, p.List}
			if p.Title.W != w || p.List.Y+p.List.H != y+28+300 {
				t.Fatal("column title/bottom misaligned")
			}
			for i, a := range items {
				if a.X < 900 || a.X+a.W > 900+w || a.W <= 0 || a.H <= 0 {
					t.Fatalf("outside column: %+v", a)
				}
				for _, b := range items[i+1:] {
					if compactIntersects(a, b) {
						t.Fatalf("header overlap: %+v %+v", a, b)
					}
				}
			}
		}
	}
}
