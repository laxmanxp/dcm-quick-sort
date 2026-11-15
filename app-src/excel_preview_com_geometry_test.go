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
