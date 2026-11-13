package main

import (
 "context"
 "errors"
 "sync/atomic"
 "testing"
 "time"
)

func TestPreparedResourceOwnership(t *testing.T) {
 t.Run("start returns while preparation is blocked", func(t *testing.T) {
  gate := make(chan struct{}); entered := make(chan struct{}); returned := make(chan *preparedResource[*int], 1)
  go func(){ returned <- newPreparedResource(func(context.Context)(*int,error){close(entered);<-gate;v:=42;return &v,nil},func(*int)error{return nil}) }()
  <-entered
  var p *preparedResource[*int]
  select {case p= <-returned:case <-time.After(time.Second):close(gate);t.Fatal("preparation blocked caller")}
  p.Cancel();close(gate)
  waitPreparation(t,p)
 })
 t.Run("late cancelled result disposed exactly once",func(t *testing.T){
  gate:=make(chan struct{});var disposed atomic.Int32
  p:=newPreparedResource(func(context.Context)(*int,error){<-gate;v:=1;return &v,nil},func(v *int)error{if v==nil || *v!=1{t.Error("wrong resource")};disposed.Add(1);return nil})
  p.Cancel();p.Cancel();close(gate)
  waitPreparation(t,p)
  if _,_,ok:=p.Take();ok{t.Fatal("cancelled result transferred")}
  if disposed.Load()!=1{t.Fatal("cleanup count",disposed.Load())}
 })
 t.Run("accepted resource transfers ownership",func(t *testing.T){
  var disposed atomic.Int32
  p:=newPreparedResource(func(context.Context)(*int,error){v:=9;return &v,nil},func(*int)error{disposed.Add(1);return nil})
  until:=time.Now().Add(time.Second)
  for {v,err,ok:=p.Take();if ok{if err!=nil || v==nil || *v!=9{t.Fatal(v,err)};break};if time.Now().After(until){t.Fatal("no prepared result")};time.Sleep(time.Millisecond)}
  p.Cancel();waitPreparation(t,p)
  if disposed.Load()!=0{t.Fatal("worker disposed transferred resource")}
 })
 t.Run("preparation error retains partial resource for cleanup",func(t *testing.T){
  fail:=errors.New("build failed");var disposed atomic.Int32
  p:=newPreparedResource(func(context.Context)(*int,error){v:=3;return &v,fail},func(v *int)error{if v==nil{t.Error("lost partial resource")};disposed.Add(1);return nil})
  until:=time.Now().Add(time.Second)
  for {v,err,ok:=p.Take();if ok{if v!=nil || !errors.Is(err,fail){t.Fatal(v,err)};break};if time.Now().After(until){t.Fatal("no error result")};time.Sleep(time.Millisecond)}
  waitPreparation(t,p);if disposed.Load()!=1{t.Fatal("partial resource not disposed")}
 })
 t.Run("cleanup failure retains ownership for retry",func(t *testing.T){
  fail:=errors.New("release not confirmed")
  p:=newPreparedResource(func(context.Context)(*int,error){v:=7;return &v,nil},func(*int)error{return fail})
  p.Cancel();out:=waitPreparation(t,p)
  if out.value==nil || *out.value!=7 || !errors.Is(out.err,fail){t.Fatal("lost cleanup failure/resource",out)}
 })
 t.Run("completion waits for cancellation cleanup",func(t *testing.T){
  gate:=make(chan struct{});entered:=make(chan struct{})
  p:=newPreparedResource(func(context.Context)(*int,error){v:=1;return &v,nil},func(*int)error{close(entered);<-gate;return nil})
  p.Cancel();<-entered
  if _,ok:=p.PollDone();ok{t.Fatal("reported done before release")}
  close(gate);waitPreparation(t,p)
 })
 t.Run("cancellation reaches worker context",func(t *testing.T){
  p:=newPreparedResource(func(ctx context.Context)(*int,error){<-ctx.Done();return nil,ctx.Err()},func(*int)error{return nil})
  p.Cancel();waitPreparation(t,p)
 })
}
func waitPreparation(t *testing.T,p *preparedResource[*int]) preparedOutcome[*int] {
 t.Helper();until:=time.Now().Add(2*time.Second)
 for {if out,ok:=p.PollDone();ok{return out};if time.Now().After(until){t.Fatal("preparation/cleanup did not finish")};time.Sleep(time.Millisecond)}
}
