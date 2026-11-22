package main

import (
 "fmt"
 "os"
 "path/filepath"
 "strings"
)

// resolveExistingDestinationFolder does not depend on a Vault root or index.
// Destinations must be existing real folders; a mistyped path must never
// implicitly create a new folder during Move.
func resolveExistingDestinationFolder(value string) (string,error) {
 value=strings.TrimSpace(value)
 if value=="" {return "",fmt.Errorf("choose a destination folder")}
 if !filepath.IsAbs(value) {return "",fmt.Errorf("destination must be an absolute folder path")}
 cleaned:=filepath.Clean(value)
 st,err:=os.Stat(cleaned)
 if err!=nil {return "",fmt.Errorf("destination folder is unavailable: %w",err)}
 if !st.IsDir() {return "",fmt.Errorf("destination must be a folder, not a file")}
 return cleaned,nil
}

func existingDestinationFolderFavorites(aliases []DestinationAlias) []DestinationAlias {
 out:=make([]DestinationAlias,0,len(aliases))
 for _,alias:=range aliases {
  if strings.TrimSpace(alias.Name)=="" {continue}
  if path,err:=resolveExistingDestinationFolder(alias.Path);err==nil {
   alias.Path=path
   out=append(out,alias)
  }
 }
 return out
}
