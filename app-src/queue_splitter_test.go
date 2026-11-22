package main

import "testing"

func TestQueueSplitterBounds(t *testing.T) {
 cases:=[]struct{width,requested,want int32}{
  {760,0,260},{760,900,290},{920,600,450},
  {1350,300,300},{1350,1200,880},{1500,200,260},
 }
 for _,c:=range cases {
  if got:=queueSplitterClamp(c.width,c.requested);got!=c.want {
   t.Errorf("clamp(%d,%d)=%d want %d",c.width,c.requested,got,c.want)
  }
 }
}

func TestQueueSplitterDefaultAndSessionPosition(t *testing.T) {
 saved:=queueSplitterWidth
 defer func(){queueSplitterWidth=saved}()
 queueSplitterWidth=0
 if got:=queueSplitterPosition(1200);got!=420 {t.Fatalf("default=%d",got)}
 queueSplitterWidth=600
 if got:=queueSplitterPosition(1200);got!=600 {t.Fatalf("drag position=%d",got)}
 if got:=queueSplitterPosition(760);got!=290 {t.Fatalf("resize clamp=%d",got)}
 if queueSplitterWidth!=600 {t.Fatal("resizing discarded preferred splitter position")}
}
