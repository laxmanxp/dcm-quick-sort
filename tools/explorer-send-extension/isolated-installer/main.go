package main

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "flag"
 "fmt"
 "io"
 "os"
 "path/filepath"
 "strings"
)

const (
 isolatedMarker = "DCMQuickSortIsolatedAcceptance-"
 installSchema = 1
)
type stagedFile struct {
 Name string `json:"name"`
 Length int64 `json:"length"`
 SHA256 string `json:"sha256"`
}
type stagedPackage struct {
 Schema int `json:"schema"`
 Component string `json:"component"`
 Passed bool `json:"passed"`
 EnabledByDefault bool `json:"enabled_by_default"`
 InstallationPerformed bool `json:"installation_performed"`
 RegistryModified bool `json:"registry_modified"`
 ExplorerRestarted bool `json:"explorer_restarted"`
 Files []stagedFile `json:"files"`
}
type installation struct {
 Schema int `json:"schema"`
 Purpose string `json:"purpose"`
 Files []stagedFile `json:"files"`
}
func safeRoot(root string)(string,error){
 if root==""{return "",errors.New("test root is required")}
 full,err:=filepath.Abs(root)
 if err!=nil{return "",err}
 full=filepath.Clean(full)
 temp,err:=filepath.Abs(os.TempDir())
 if err!=nil{return "",err}
 if !strings.EqualFold(filepath.Dir(full),filepath.Clean(temp)) || !strings.HasPrefix(filepath.Base(full),isolatedMarker) ||
   len(filepath.Base(full))<=len(isolatedMarker) {
  return "",errors.New("test root must be a direct, uniquely named child of the system temp directory")
 }
 return full,nil
}
func shaFile(path string)(stagedFile,error){
 f,err:=os.Open(path);if err!=nil{return stagedFile{},err}
 defer f.Close()
 stat,err:=f.Stat();if err!=nil{return stagedFile{},err}
 if !stat.Mode().IsRegular(){return stagedFile{},errors.New("not an ordinary file")}
 h:=sha256.New()
 n,err:=io.Copy(h,f);if err!=nil{return stagedFile{},err}
 return stagedFile{Name:filepath.Base(path),Length:n,SHA256:hex.EncodeToString(h.Sum(nil))},nil
}
func sameFile(a,b stagedFile)bool{
 return a.Name==b.Name && a.Length==b.Length && a.Length>0 && len(a.SHA256)==64 && strings.EqualFold(a.SHA256,b.SHA256)
}
func readJSON(path string, dest interface{})error{
 raw,err:=os.ReadFile(path);if err!=nil{return err}
 raw=bytesWithoutBOM(raw)
 return json.Unmarshal(raw,dest)
}
func bytesWithoutBOM(b []byte)[]byte{
 if len(b)>=3 && b[0]==0xef && b[1]==0xbb && b[2]==0xbf{return b[3:]}
 return b
}
func expectedPackage(packageDir string)([]stagedFile,error){
 var p stagedPackage
 if err:=readJSON(filepath.Join(packageDir,"manifest.json"),&p);err!=nil{return nil,err}
 if p.Schema!=1 || p.Component!="DCM QuickSort Explorer Send" || !p.Passed ||
  p.EnabledByDefault || p.InstallationPerformed || p.RegistryModified || p.ExplorerRestarted ||
  len(p.Files)!=2 {return nil,errors.New("invalid staged Explorer package")}
 allowed:=map[string]bool{"DCMQuickSortExplorerSend.dll":true,"DCMQuickSortExplorerSend.exe":true}
 seen:=map[string]bool{}
 for _,item:=range p.Files{
  if !allowed[item.Name] || seen[item.Name]{return nil,errors.New("unexpected or duplicate Explorer component")}
  seen[item.Name]=true
  actual,err:=shaFile(filepath.Join(packageDir,"payload",item.Name))
  if err!=nil{return nil,err}
  if !sameFile(item,actual){return nil,fmt.Errorf("Explorer payload changed: %s",item.Name)}
 }
 return p.Files,nil
}
func copyVerified(from,to string,want stagedFile)error{
 in,err:=os.Open(from);if err!=nil{return err}
 defer in.Close()
 out,err:=os.OpenFile(to,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600);if err!=nil{return err}
 _,err=io.Copy(out,in)
 closeErr:=out.Close()
 if err!=nil{return err}
 if closeErr!=nil{return closeErr}
 got,err:=shaFile(to)
 if err!=nil{return err}
 if !sameFile(want,got){return errors.New("copied artifact failed integrity check")}
 return nil
}
func install(root,packageDir,appPath string)error{
 root,err:=safeRoot(root);if err!=nil{return err}
 if filepath.Base(appPath)!="DCM-QuickSort.exe"{return errors.New("app path must be named DCM-QuickSort.exe")}
 items,err:=expectedPackage(packageDir);if err!=nil{return err}
 app,err:=shaFile(appPath);if err!=nil{return err}
 if app.Length==0{return errors.New("empty application")}
 entries:=append([]stagedFile{app},items...)
 if err:=os.Mkdir(root,0700);err!=nil{return fmt.Errorf("create fresh test installation: %w",err)}
 // A failure leaves only a disposable test directory; never touch live paths.
 success:=false
 defer func(){if !success{_ = os.RemoveAll(root)}}()
 appDir:=filepath.Join(root,"app")
 if err:=os.Mkdir(appDir,0700);err!=nil{return err}
 for _,entry:=range entries{
  from:=appPath
  if entry.Name!=app.Name{from=filepath.Join(packageDir,"payload",entry.Name)}
  if err:=copyVerified(from,filepath.Join(appDir,entry.Name),entry);err!=nil{return err}
 }
 record:=installation{Schema:installSchema,Purpose:"disposable Explorer package lifecycle acceptance; no COM registration",Files:entries}
 raw,err:=json.MarshalIndent(record,"","  ");if err!=nil{return err}
 if err:=os.WriteFile(filepath.Join(root,"installation.json"),raw,0600);err!=nil{return err}
 success=true
 return nil
}
func uninstall(root string)error{
 root,err:=safeRoot(root);if err!=nil{return err}
 var record installation
 if err:=readJSON(filepath.Join(root,"installation.json"),&record);err!=nil{return err}
 if record.Schema!=installSchema || record.Purpose!="disposable Explorer package lifecycle acceptance; no COM registration" || len(record.Files)!=3 {
  return errors.New("installation ownership marker is invalid")
 }
 required:=map[string]bool{"DCM-QuickSort.exe":true,"DCMQuickSortExplorerSend.dll":true,"DCMQuickSortExplorerSend.exe":true}
 seen:=map[string]bool{}
 for _,entry:=range record.Files {
  if !required[entry.Name] || seen[entry.Name]{return errors.New("unexpected installed file")}
  seen[entry.Name]=true
  actual,err:=shaFile(filepath.Join(root,"app",entry.Name));if err!=nil{return err}
  if !sameFile(entry,actual){return fmt.Errorf("refusing removal of modified or foreign file: %s",entry.Name)}
 }
 // Require exactly the owned files, not an arbitrary directory tree.
 listing,err:=os.ReadDir(filepath.Join(root,"app"));if err!=nil{return err}
 if len(listing)!=3{return errors.New("unrecognized files in test install; refusing removal")}
 top,err:=os.ReadDir(root);if err!=nil{return err}
 if len(top)!=2{return errors.New("unexpected test installation entries; refusing removal")}
 for _,entry:=range record.Files {
  if err:=os.Remove(filepath.Join(root,"app",entry.Name));err!=nil{return err}
 }
 if err:=os.Remove(filepath.Join(root,"app"));err!=nil{return err}
 if err:=os.Remove(filepath.Join(root,"installation.json"));err!=nil{return err}
 return os.Remove(root)
}
func main(){
 mode:=flag.String("mode","","install or uninstall")
 root:=flag.String("root","","disposable test installation root")
 pkg:=flag.String("package","","staged Explorer package directory")
 app:=flag.String("app","","source DCM-QuickSort.exe")
 flag.Parse()
 var err error
 switch *mode{
 case "install":err=install(*root,*pkg,*app)
 case "uninstall":err=uninstall(*root)
 default:err=errors.New("explicit -mode=install or -mode=uninstall required")
 }
 if err!=nil{fmt.Fprintln(os.Stderr,"FAILED:",err);os.Exit(1)}
 fmt.Println("PASS:",*mode,"isolated QuickSort package; no registry or live installation modified")
}
