package main

import "testing"

func TestExcelCOMRectUsesPhysicalSizeWhenAvailable(t *testing.T) {
	cases := []struct{name string;logicalW,logicalH,physicalW,physicalH int32;excel,available bool;wantW,wantH int32}{
		{"blank-nonexcel",722,390,902,487,false,true,722,390},
		{"error-no-physical-reading",722,390,0,0,true,false,722,390},
		{"success-125percent",722,390,902,487,true,true,902,487},
		{"success-100percent",902,487,902,487,true,true,902,487},
		{"bad-physical-zero",722,390,0,487,true,true,722,390},
	}
	for _,tt:=range cases {t.Run(tt.name,func(t *testing.T){
		w,h:=excelCOMRectSize(tt.logicalW,tt.logicalH,tt.physicalW,tt.physicalH,tt.excel,tt.available)
		if w!=tt.wantW||h!=tt.wantH {t.Fatalf("got %dx%d want %dx%d",w,h,tt.wantW,tt.wantH)}
	})}
}

func TestXLSXExcelCOMRectStagePolicy(t *testing.T) {
	cases := []struct {
		name, method string
		logicalW, logicalH, physicalW, physicalH int32
		excel, available bool
		wantW, wantH int32
	}{
		{"initial-125percent-stays-logical", "SetWindow", 722, 390, 902, 487, true, true, 722, 390},
		{"initial-100percent-stays-logical", "SetWindow", 902, 487, 902, 487, true, true, 902, 487},
		{"initial-setrect-stays-logical", "InitialSetRect", 722, 390, 902, 487, true, true, 722, 390},
		{"resize-125percent-uses-physical", "SetRect", 722, 390, 902, 487, true, true, 902, 487},
		{"resize-nonexcel-stays-logical", "SetRect", 722, 390, 902, 487, false, true, 722, 390},
		{"resize-failed-measurement-stays-logical", "SetRect", 722, 390, 0, 0, true, false, 722, 390},
		{"unknown-operation-stays-logical", "Other", 722, 390, 902, 487, true, true, 722, 390},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, h := excelCOMRectForCall(tc.method, tc.logicalW, tc.logicalH, tc.physicalW, tc.physicalH, tc.excel, tc.available)
			if w != tc.wantW || h != tc.wantH {
				t.Fatalf("%s got %dx%d want %dx%d", tc.method, w, h, tc.wantW, tc.wantH)
			}
		})
	}
}
