package main

import "testing"

func TestImageZoomAnchorPan(t *testing.T){
 x,y:=imageZoomAnchorPan(1000,800,1200,960,500,400,250,200,0,0)
 if x!=0 || y!=0 {t.Fatalf("center anchored got %d,%d",x,y)}
 x,y=imageZoomAnchorPan(1000,800,1200,960,500,400,100,100,0,0)
 if x!=30 || y!=20 {t.Fatalf("pointer anchor got %d,%d, expected 30,20",x,y)}
 x,y=imageZoomAnchorPan(800,600,900,700,500,400,-300,-300,0,0)
 if x!=0 || y!=0 {t.Fatalf("off image pointer got %d,%d",x,y)}
}
func TestImagePreviewRotation(t *testing.T){
 for _,tc:=range []struct{current,step,want int}{{0,90,90},{90,90,180},{270,90,0},{0,-90,270},{90,-90,0}}{
  if got:=nextImagePreviewRotation(tc.current,tc.step);got!=tc.want {t.Errorf("%v got %d",tc,got)}
 }
}
