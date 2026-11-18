//go:build windows

package main

import (
 "bufio"
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "os"
 "os/exec"
 "strings"
 "sync"
 "syscall"
 "time"
)

const (
 flowPDFiumProtocolVersion=1
 flowPDFiumMaxReplyBytes=256<<10
 flowPDFiumRequestDeadline=15*time.Second
 flowPDFiumShutdownDeadline=2*time.Second
)

type flowPDFiumRequest struct {
 Version int `json:"version"`
 ID string `json:"id"`
 Command string `json:"command"`
 InputPath string `json:"inputPath,omitempty"`
 OutputPath string `json:"outputPath,omitempty"`
 Mode string `json:"mode,omitempty"`
 PageIndex int `json:"pageIndex,omitempty"`
 Rotation int `json:"rotation,omitempty"`
 Zoom int `json:"zoom,omitempty"`
 ViewportWidth int `json:"viewportWidth,omitempty"`
 ViewportHeight int `json:"viewportHeight,omitempty"`
}
type flowPDFiumReply struct {
 Version int `json:"version"`
 ID string `json:"id"`
 OK bool `json:"ok"`
 Result string `json:"result"`
 Error string `json:"error"`
 CacheBytes int64 `json:"cacheBytes"`
}
type flowPDFiumPipeResult struct {line string;err error}

// Canceled renders are expected when the user changes files. They must never
// start a transport retry or a one-shot fallback for the obsolete document.
var errFlowPDFiumCanceled = errors.New("PDFium render canceled after selection change")

// flowPDFiumProcess is a private, sequential child of the exact Quick Sort
// instance that created it. It never listens on a TCP/global named pipe.
// The mutex covers the whole request/response exchange and child lifecycle.
type flowPDFiumProcess struct {
 mu sync.Mutex
 executable string
 cmd *exec.Cmd
 stdin io.WriteCloser
 scanner *bufio.Scanner
 sequence uint64

 // The GUI can cancel a stale worker without acquiring mu, which is held
 // during the blocking request/response. Every generation is monotonic
 // within one Quick Sort process.
 activeMu sync.Mutex
 activeProcess *os.Process
 activeGeneration uint64
 canceledThrough uint64
}

func (p *flowPDFiumProcess) obsolete(generation uint64) bool {
 p.activeMu.Lock()
 defer p.activeMu.Unlock()
 return p.canceledThrough>0 && generation<=p.canceledThrough
}

// cancelThrough does not wait for the IPC mutex. It records a cancellation
// boundary for requests not yet registered, and kills only the exact child
// owned by an already active obsolete request. Future generations are safe.
func (p *flowPDFiumProcess) cancelThrough(generation uint64) bool {
 if generation==0 {return false}
 p.activeMu.Lock()
 defer p.activeMu.Unlock()
 if generation>p.canceledThrough{p.canceledThrough=generation}
 if p.activeProcess==nil||p.activeGeneration>generation{return false}
 _=p.activeProcess.Kill()
 return true
}

// registerActive also covers the race where the UI stopped the selection
// before the rendering goroutine obtained mu and started its worker.
func (p *flowPDFiumProcess) registerActive(generation uint64) bool {
 p.activeMu.Lock()
 defer p.activeMu.Unlock()
 if p.canceledThrough>0 && generation<=p.canceledThrough{return false}
 p.activeGeneration=generation
 p.activeProcess=p.cmd.Process
 return true
}

func (p *flowPDFiumProcess) clearActive(generation uint64) bool {
 p.activeMu.Lock()
 defer p.activeMu.Unlock()
 if p.activeGeneration==generation{
  p.activeGeneration=0
  p.activeProcess=nil
 }
 return p.canceledThrough>0 && generation<=p.canceledThrough
}

func (p *flowPDFiumProcess) startLocked(exe string)(int,error){
 if p.cmd!=nil && strings.EqualFold(p.executable,exe) {
  return p.cmd.Process.Pid,nil
 }
 if p.cmd!=nil {p.discardLocked()}
 cmd:=exec.Command(exe,"-ServeJSONL")
 cmd.Env=developmentPDFiumWorkerEnvironment()
 cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:createNoWindow}
 stdin,err:=cmd.StdinPipe()
 if err!=nil{return 0,err}
 stdout,err:=cmd.StdoutPipe()
 if err!=nil {stdin.Close();return 0,err}
 // The protocol uses stdout only. Keep stderr out of the request stream.
 cmd.Stderr=io.Discard
 if err:=cmd.Start();err!=nil {stdin.Close();return 0,err}
 scanner:=bufio.NewScanner(stdout)
 scanner.Buffer(make([]byte,4096),flowPDFiumMaxReplyBytes)
 p.cmd,p.stdin,p.scanner,p.executable=cmd,stdin,scanner,exe
 return cmd.Process.Pid,nil
}

// discardLocked terminates only our previously spawned child, never
// arbitrary processes found by executable name.
func (p *flowPDFiumProcess) discardLocked(){
 if p.cmd==nil{return}
 _=p.stdin.Close()
 _=p.cmd.Process.Kill()
 _=p.cmd.Wait()
 p.cmd=nil
 p.stdin=nil
 p.scanner=nil
 p.executable=""
}

func (p *flowPDFiumProcess) exchangeLocked(req flowPDFiumRequest,deadline time.Duration)(flowPDFiumReply,error){
 if p.cmd==nil {return flowPDFiumReply{},errors.New("PDFium service is not started")}
 line,err:=json.Marshal(req)
 if err!=nil{return flowPDFiumReply{},err}
 if len(line)>64<<10{return flowPDFiumReply{},errors.New("PDFium request exceeds protocol limit")}
 if _,err=p.stdin.Write(append(line,'\n'));err!=nil {
  return flowPDFiumReply{},fmt.Errorf("PDFium pipe write: %w",err)
 }
 resultC:=make(chan flowPDFiumPipeResult,1)
 scanner:=p.scanner
 go func(){
  if scanner.Scan() {resultC<-flowPDFiumPipeResult{line:scanner.Text()};return}
  err:=scanner.Err()
  if err==nil {err=io.EOF}
  resultC<-flowPDFiumPipeResult{err:err}
 }()
 timer:=time.NewTimer(deadline)
 defer timer.Stop()
 select {
 case data:=<-resultC:
  if data.err!=nil {return flowPDFiumReply{},fmt.Errorf("PDFium pipe read: %w",data.err)}
  var reply flowPDFiumReply
  if err:=json.Unmarshal([]byte(data.line),&reply);err!=nil{return reply,fmt.Errorf("PDFium protocol JSON: %w",err)}
  if reply.Version!=flowPDFiumProtocolVersion||reply.ID!=req.ID{
   return reply,errors.New("PDFium protocol version/request ID mismatch")
  }
  return reply,nil
 case <-timer.C:
  // Unblock the owned scanner goroutine before returning. The next request
  // will start a fresh child; a hung renderer cannot monopolize the pipe.
  _=p.cmd.Process.Kill()
  select {
  case <-resultC:
  case <-time.After(2*time.Second):
  }
  return flowPDFiumReply{},errors.New("PDFium persistent render deadline exceeded")
 }
}

func flowPDFiumMode(req PDFRenderRequest)string{
 if req.FitWidth{return "fitwidth"}
 if req.Fit{return "fit"}
 return "zoom"
}

// render restarts the private child once after a pipe/protocol failure.
// A valid PDFium error response (damaged/password-protected PDF) is final,
// and is not retried or silently converted to a different rendering result.
func (p *flowPDFiumProcess) render(exe string,req PDFRenderRequest,deadline time.Duration,started func(int))(PDFRenderMetadata,int,error){
 p.mu.Lock()
 defer p.mu.Unlock()
 if p.obsolete(req.Generation) {
  _=os.Remove(req.OutputPath)
  return PDFRenderMetadata{},0,errFlowPDFiumCanceled
 }
 var last error
 for attempt:=0;attempt<2;attempt++ {
  if p.obsolete(req.Generation) {
   _=os.Remove(req.OutputPath)
   return PDFRenderMetadata{},0,errFlowPDFiumCanceled
  }
  pid,err:=p.startLocked(exe)
  if err!=nil {last=fmt.Errorf("PDFium service start: %w",err);p.discardLocked();continue}
  if !p.registerActive(req.Generation) {
   p.discardLocked()
   _=os.Remove(req.OutputPath)
   return PDFRenderMetadata{},0,errFlowPDFiumCanceled
  }
  if started!=nil {started(pid)}
  p.sequence++
  id:=fmt.Sprintf("pdf-%d-%d",req.Generation,p.sequence)
  cmd:=flowPDFiumRequest{
   Version:flowPDFiumProtocolVersion,ID:id,Command:"render",
   InputPath:req.InputPath,OutputPath:req.OutputPath,Mode:flowPDFiumMode(req),
   PageIndex:req.PageIndex,Rotation:normalizePDFPreviewRotation(req.Rotation),Zoom:clampPDFZoom(req.Zoom),
   ViewportWidth:req.ViewportWidth,ViewportHeight:req.ViewportHeight,
  }
  reply,err:=p.exchangeLocked(cmd,deadline)
  wasCanceled:=p.clearActive(req.Generation)
  if wasCanceled {
   p.discardLocked()
   _=os.Remove(req.OutputPath)
   return PDFRenderMetadata{},pid,errFlowPDFiumCanceled
  }
  if err!=nil{
   last=err
   p.discardLocked()
   // A crashed child may have written the output but lost its response.
   _=os.Remove(req.OutputPath)
   continue
  }
  if !reply.OK {
   if reply.Error==""{reply.Error="renderer rejected request"}
   return PDFRenderMetadata{},pid,errors.New("PDFium: "+reply.Error)
  }
  meta,err:=parsePDFRendererOutput(reply.Result)
  if err!=nil{
   last=err
   p.discardLocked()
   _=os.Remove(req.OutputPath)
   continue
  }
  return meta,pid,nil
 }
 if last==nil{last=errors.New("PDFium service unavailable")}
 return PDFRenderMetadata{},0,last
}

func (p *flowPDFiumProcess) close(){
 p.mu.Lock()
 defer p.mu.Unlock()
 if p.cmd==nil{return}
 p.sequence++
 req:=flowPDFiumRequest{Version:flowPDFiumProtocolVersion,ID:fmt.Sprintf("shutdown-%d",p.sequence),Command:"shutdown"}
 reply,err:=p.exchangeLocked(req,flowPDFiumShutdownDeadline)
 if err==nil && reply.OK && reply.Result=="bye" {
  // The child has acknowledged shutdown; Wait reaps its exact PID.
  _=p.stdin.Close()
  _=p.cmd.Wait()
  p.cmd,p.stdin,p.scanner=nil,nil,nil
  p.executable=""
  return
 }
 p.discardLocked()
}

// On a persistent IPC transport failure, preserve the pre-existing one-shot
// PDFium route under a separate deadline. Decoder refusals remain errors:
// the fallback is specifically for a broken/unresponsive service.
func flowPDFiumOneShotFallback(executable string,req PDFRenderRequest)(PDFRenderMetadata,int,error){
 ctx,cancel:=context.WithTimeout(context.Background(),flowPDFiumRequestDeadline)
 defer cancel()
 cmd:=exec.CommandContext(ctx,executable,pdfiumRendererArgs(req)...)
 cmd.Env=developmentPDFiumWorkerEnvironment()
 cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:createNoWindow}
 var output bytes.Buffer
 cmd.Stdout=&output
 cmd.Stderr=&output
 if err:=cmd.Start();err!=nil{return PDFRenderMetadata{},0,err}
 pid:=cmd.Process.Pid
 recordPDFTrace(PDFTraceEvent{Event:"persistent-fallback-started",Generation:req.Generation,InputPath:req.InputPath,OutputPath:req.OutputPath,RendererPID:pid})
 err:=cmd.Wait()
 if ctx.Err()!=nil {return PDFRenderMetadata{},pid,fmt.Errorf("PDFium one-shot fallback deadline: %w",ctx.Err())}
 if err!=nil {return PDFRenderMetadata{},pid,fmt.Errorf("PDFium one-shot fallback: %w: %s",err,strings.TrimSpace(output.String()))}
 meta,err:=parsePDFRendererOutput(output.String())
 return meta,pid,err
}

var sharedFlowPDFiumProcess flowPDFiumProcess

func closeSharedFlowPDFiumProcess(){sharedFlowPDFiumProcess.close()}
