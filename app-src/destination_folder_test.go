package main

import (
 "os"
 "path/filepath"
 "testing"
)

func TestExistingDestinationFolder(t *testing.T){
 root:=t.TempDir()
 got,err:=resolveExistingDestinationFolder(root)
 if err!=nil||got!=filepath.Clean(root){t.Fatalf("valid folder: %q %v",got,err)}
 for _,value:=range []string{"","relative/folder",filepath.Join(root,"missing")} {
  if got,err:=resolveExistingDestinationFolder(value);err==nil {t.Fatalf("accepted invalid destination %q as %q",value,got)}
 }
 file:=filepath.Join(root,"document.txt")
 if err:=os.WriteFile(file,[]byte("fixture"),0600);err!=nil{t.Fatal(err)}
 if got,err:=resolveExistingDestinationFolder(file);err==nil{t.Fatalf("accepted file as destination: %s",got)}
}

func TestDestinationFavoritesOnlyExistingAbsoluteFolders(t *testing.T){
 root:=t.TempDir()
 values:=[]DestinationAlias{
  {Name:"Valid",Path:root},
  {Name:"Old logical Vault address",Path:"Office/Reports"},
  {Name:"Missing",Path:filepath.Join(root,"missing")},
 }
 got:=existingDestinationFolderFavorites(values)
 if len(got)!=1||got[0].Name!="Valid" {t.Fatalf("unexpected favorites: %#v",got)}
}
