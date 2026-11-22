package main

import (
 "fmt"
 "path/filepath"
 "strings"
)

// historyTransactionNarration describes the recorded action, not the current
// state of the filesystem. Undo preflight verifies files before any changes.
func historyTransactionNarration(rec MoveRecord) string {
 from,to:=rec.From,rec.To
 var action,undo string
 switch rec.Kind {
 case MoveKindReplace:
  action="An existing destination file was replaced by the source file. QuickSort kept a backup of the previous destination."
  undo="Move the current destination file back to its original source path, then restore the previously replaced destination file from its backup."
 case MoveKindDuplicateRemoved:
  action="QuickSort found matching file contents at the destination and removed the redundant source copy."
  undo="Restore the removed source copy from the matching destination file after verifying the stored file identity."
 default:
  if isRenameOnlyRecord(rec) {
   action="The file was renamed within the same folder."
   undo="Rename the file back to its original name in the same folder."
  } else {
   action="The file was moved from its original location to the destination."
   undo="Move the file from the destination back to its original location."
  }
 }
 var b strings.Builder
 b.WriteString("ACTION TAKEN\r\n")
 b.WriteString(action)
 b.WriteString("\r\n\r\nORIGINAL FILE\r\n")
 b.WriteString(from)
 b.WriteString("\r\n\r\nRESULTING LOCATION\r\n")
 b.WriteString(to)
 if rec.Kind==MoveKindReplace && strings.TrimSpace(rec.ReplacedBackup)!="" {
  b.WriteString("\r\n\r\nREPLACED FILE BACKUP\r\n")
  b.WriteString(rec.ReplacedBackup)
 }
 b.WriteString("\r\n\r\nWHAT UNDO WILL DO\r\n")
 b.WriteString(undo)
 b.WriteString("\r\n\r\n")
 if rec.BatchID!="" {
  b.WriteString(fmt.Sprintf("BATCH\r\nThis transaction belongs to a batch. Undo affects this selected transaction; Undo Batch acts on the latest eligible completed batch, which may be different.\r\n\r\n"))
 }
 b.WriteString("Undo is subject to filesystem and safety checks. Select Undo to attempt the reversal.")
 return b.String()
}

func historyEmptyNarration() string {
 return "TRANSACTION DETAILS\r\n\r\nSelect a processed file in History to see what QuickSort did and what Undo would attempt.\r\n\r\nSelecting a date group does not select a transaction."
}

func historyBatchNarration(history []MoveRecord) string {
 id:=latestUndoableBatchID(history)
 if id=="" {return ""}
 return fmt.Sprintf("\r\n\r\nUNDO BATCH\r\nThe latest eligible batch contains %d transaction(s). Undo Batch requests confirmation, checks the files, then attempts to restore the batch newest-first.",len(batchHistoryIndexes(history,id)))
}

func historyOperationName(rec MoveRecord) string {
 switch rec.Kind {
 case MoveKindReplace: return "REPLACED FILE"
 case MoveKindDuplicateRemoved: return "REMOVED DUPLICATE"
 default:
  if isRenameOnlyRecord(rec){return "RENAMED FILE"}
  if filepath.Clean(rec.From)==filepath.Clean(rec.To){return "FILE OPERATION"}
  return "MOVED FILE"
 }
}
