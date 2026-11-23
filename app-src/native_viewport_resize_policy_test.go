package main

import "testing"

func TestNativeViewportResizeOnlyForCurrentHandler(t *testing.T) {
 for _, tc := range []struct{
  label string
  postedEpoch,currentEpoch uint64
  postedPane,currentPane uintptr
  active,want bool
 }{
  {"current",4,4,7,7,true,true},
  {"replaced handler",3,4,7,7,true,false},
  {"wrong pane",4,4,7,8,true,false},
  {"unloaded",4,4,7,7,false,false},
  {"no pane",4,4,0,0,true,false},
 } {
  if got:=shouldRunNativeViewportResize(tc.postedEpoch,tc.currentEpoch,tc.postedPane,tc.currentPane,tc.active);got!=tc.want {
   t.Errorf("%s: got %v want %v",tc.label,got,tc.want)
  }
 }
}
