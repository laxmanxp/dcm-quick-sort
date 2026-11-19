from pathlib import Path
import hashlib
root=Path(__file__).resolve().parents[1]
for name,expected in {'app-src/control_updates_windows.go':'6ddf79ea3b47a884935e4d7b11ef1c35bccc063bf92fa915f5041b33e72fd42d','app-src/control_transition_windows_test.go':'911ca02bd766a4957962a3daf772f740bbe36e7f2fe27464f3562838bd1ee146','app-src/control_updates_windows_test.go':'fe14b4c907ebc447832a32326c4f3e451ae0adac16f5cef02508ea294e440e20'}.items():
 assert hashlib.sha256((root/name).read_bytes()).hexdigest()==expected,(name,'authored source mismatch')
p=root/'app-src/main_windows.go';b=p.read_bytes()
expected='9957928f4c8f3f6ee962ec3e5583b349b888c2caaedd0d58fc7beaf16c1edac4'
if hashlib.sha256(b).hexdigest()!=expected:
 assert hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()=='cd2817f500ccd7a81c0931668f77113d1f7d769d','main changed'
 s=b.decode()
 pairs=[
 ('procMoveWindow.Call(app.sourceAliasCombo, 85, 13, uintptr(aliasWTop), 240, 0)','moveLayoutCombo(app.sourceAliasCombo, 85, 13, uintptr(aliasWTop), 240)'),
 ('procMoveWindow.Call(app.categoryCombo, 85, 82, 165, 260, 0)','moveLayoutCombo(app.categoryCombo, 85, 82, 165, 260)'),
 ('procMoveWindow.Call(app.extensionCombo, 340, 82, 150, 320, 0)','moveLayoutCombo(app.extensionCombo, 340, 82, 150, 320)'),
 ('procMoveWindow.Call(app.filterAgeCombo, 545, 82, 135, 220, 0)','moveLayoutCombo(app.filterAgeCombo, 545, 82, 135, 220)'),
 ('procMoveWindow.Call(app.filterSizeCombo, 735, 82, 135, 220, 0)','moveLayoutCombo(app.filterSizeCombo, 735, 82, 135, 220)'),
 ('procMoveWindow.Call(app.filterSortCombo, uintptr(sortComboX), 82, uintptr(sortW), 220, 0)','moveLayoutCombo(app.filterSortCombo, uintptr(sortComboX), 82, uintptr(sortW), 220)'),
 ('procMoveWindow.Call(app.renameToolCombo, uintptr(renameComboX), uintptr(y), uintptr(renameComboW), 220, 0)','moveLayoutCombo(app.renameToolCombo, uintptr(renameComboX), uintptr(y), uintptr(renameComboW), 220)'),
 ('procMoveWindow.Call(app.destAliasCombo, uintptr(fieldX), uintptr(destY), uintptr(aliasW), 260, 0)','moveLayoutCombo(app.destAliasCombo, uintptr(fieldX), uintptr(destY), uintptr(aliasW), 260)'),
 ('procShowWindow.Call(hwnd, show)','setPreviewControlVisible(hwnd, show != 0)'),
 ('procEnableWindow.Call(hwnd, zoomEnabled)','setPreviewControlEnabled(hwnd, zoomEnabled != 0)'),
 ('procEnableWindow.Call(app.zoomFitBtn, fitEnabled)','setPreviewControlEnabled(app.zoomFitBtn, fitEnabled != 0)'),
 ('procEnableWindow.Call(app.zoomFitWidthBtn, fitWidthEnabled)','setPreviewControlEnabled(app.zoomFitWidthBtn, fitWidthEnabled != 0)')]
 for old,new in pairs:
  assert s.count(old)==1,(old,'not unique')
  s=s.replace(old,new)
 assert hashlib.sha256(s.encode()).hexdigest()==expected,'output changed'
 p.write_bytes(s.encode())
print('Exact live-state control update patch verified')
