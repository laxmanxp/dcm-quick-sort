package main

import (
 "strings"
 "testing"
)

func TestHistoryNarrationExplainsRealUndoSemantics(t *testing.T){
 cases:=[]struct{name string;rec MoveRecord;action,undo string}{
  {"rename",MoveRecord{From:`C:\Incoming\Old.txt`,To:`C:\Incoming\New.txt`,Kind:MoveKindMove},"renamed","Rename the file back"},
  {"move",MoveRecord{From:`C:\Incoming\File.pdf`,To:`D:\Archive\File.pdf`,Kind:MoveKindMove},"moved","Move the file from the destination"},
  {"replace",MoveRecord{From:`C:\Incoming\File.pdf`,To:`D:\Archive\File.pdf`,Kind:MoveKindReplace,ReplacedBackup:`D:\Recovery\backup`},"replaced","restore the previously replaced destination"},
  {"duplicate",MoveRecord{From:`C:\Incoming\File.pdf`,To:`D:\Archive\File.pdf`,Kind:MoveKindDuplicateRemoved},"matching file contents","Restore the removed source copy"},
 }
 for _,tc:=range cases{
  t.Run(tc.name,func(t *testing.T){
   got:=historyTransactionNarration(tc.rec)
   for _,expected:=range []string{tc.action,tc.undo,tc.rec.From,tc.rec.To,"WHAT UNDO WILL DO","safety checks"}{
    if !strings.Contains(strings.ToLower(got),strings.ToLower(expected)){t.Fatalf("narration missing %q: %s",expected,got)}
   }
  })
 }
}

func TestHistoryEmptyNarration(t *testing.T){
 if !strings.Contains(historyEmptyNarration(),"Select a processed file"){t.Fatal("missing selection instruction")}
}
