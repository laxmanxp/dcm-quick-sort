from pathlib import Path
root=Path(__file__).resolve().parents[1]
p=root/'app-src/main_windows.go'; original=p.read_bytes()
assert __import__('hashlib').sha256(original).hexdigest()=='9957928f4c8f3f6ee962ec3e5583b349b888c2caaedd0d58fc7beaf16c1edac4','main source changed'
s=original.decode()
def one(a,b):
 global s
 assert s.count(a)==1,(a,s.count(a));s=s.replace(a,b)
one('type appState struct {','type appState struct {\n\tfilterTypeLabel, filterExtensionLabel, filterAgeLabel, filterSizeLabel uintptr')
for text,field in [('File Type:','filterTypeLabel'),('Extension:','filterExtensionLabel'),('Age:','filterAgeLabel'),('Size:','filterSizeLabel')]:
 one('createControl("STATIC", "'+text+'",','app.'+field+' = createControl("STATIC", "'+text+'",')
a=s.index('func layoutUI('); b=s.index('\nfunc populateFileList()',a); head=s[:a];tail=s[b:];s=s[a:b]
one('previewH := h - 335','filters := compactFilterLayout(w)\n\theaderY := filters.HeaderY\n\tpreviewTop := headerY + 28\n\tpreviewH := h - 335 - (headerY - 120)')
one('\trecordAttachmentTrace("layout-before-categoryCombo")', '''	for i, label := range []uintptr{app.filterTypeLabel, app.filterExtensionLabel, app.filterAgeLabel, app.filterSizeLabel, app.filterSortLabel} {
		r := filters.Labels[i]
		procMoveWindow.Call(label, uintptr(r.X), uintptr(r.Y), uintptr(r.W), uintptr(r.H), 0)
	}
	recordAttachmentTrace("layout-before-categoryCombo")''')
for i,(field,args,height) in enumerate([('categoryCombo','85, 82, 165, 260',260),('extensionCombo','340, 82, 150, 320',320),('filterAgeCombo','545, 82, 135, 220',220),('filterSizeCombo','735, 82, 135, 220',220)]):
 one(f'moveLayoutCombo(app.{field}, {args})',f'moveLayoutCombo(app.{field}, uintptr(filters.Fields[{i}].X), uintptr(filters.Fields[{i}].Y), uintptr(filters.Fields[{i}].W), {height})')
x=s.index('\tclearW := int32(105)');y=s.index('\trecordAttachmentTrace("layout-before-countLabel")',x)
s=s[:x]+'''	recordAttachmentTrace("layout-before-filterSortCombo")
	moveLayoutCombo(app.filterSortCombo, uintptr(filters.Fields[4].X), uintptr(filters.Fields[4].Y), uintptr(filters.Fields[4].W), 220)
	recordAttachmentTrace("layout-after-filterSortCombo")
	recordAttachmentTrace("layout-before-filterClearBtn")
	procMoveWindow.Call(app.filterClearBtn, uintptr(filters.Clear.X), uintptr(filters.Clear.Y), uintptr(filters.Clear.W), uintptr(filters.Clear.H), 0)
	recordAttachmentTrace("layout-after-filterClearBtn")
'''+s[y:]
s=s.replace(', 120,',', uintptr(headerY),').replace(', 116,',', uintptr(headerY-4),')
x=s.index('\tundoW := int32(68)');y=s.index('\trecordAttachmentTrace("layout-before-fileList")',x)
s=s[:x]+'''	processed := compactProcessedLayout(processedX, processedW, headerY, previewH)
	recordAttachmentTrace("layout-before-processedTitleLabel")
	procMoveWindow.Call(app.processedTitleLabel, uintptr(processed.Title.X), uintptr(processed.Title.Y), uintptr(processed.Title.W), uintptr(processed.Title.H), 0)
	recordAttachmentTrace("layout-after-processedTitleLabel")
	recordAttachmentTrace("layout-before-undoBatchBtn")
	procMoveWindow.Call(app.undoBatchBtn, uintptr(processed.UndoBatch.X), uintptr(processed.UndoBatch.Y), uintptr(processed.UndoBatch.W), uintptr(processed.UndoBatch.H), 0)
	recordAttachmentTrace("layout-after-undoBatchBtn")
	recordAttachmentTrace("layout-before-undoBtn")
	procMoveWindow.Call(app.undoBtn, uintptr(processed.Undo.X), uintptr(processed.Undo.Y), uintptr(processed.Undo.W), uintptr(processed.Undo.H), 0)
	recordAttachmentTrace("layout-after-undoBtn")
'''+s[y:]
s=s.replace(', 148,',', uintptr(previewTop),')
one('procMoveWindow.Call(app.processedSearchEdit, uintptr(processedX), uintptr(previewTop), uintptr(processedW), 25, 0)', 'procMoveWindow.Call(app.processedSearchEdit, uintptr(processed.Search.X), uintptr(processed.Search.Y), uintptr(processed.Search.W), uintptr(processed.Search.H), 0)')
x=s.index('\tprocessedListY := int32(181)');y=s.index('\trecordAttachmentTrace("layout-before-processedList")',x)
s=s[:x]+s[y:]
one('procMoveWindow.Call(app.processedList, uintptr(processedX), uintptr(processedListY), uintptr(processedW), uintptr(processedListH), 0)','procMoveWindow.Call(app.processedList, uintptr(processed.List.X), uintptr(processed.List.Y), uintptr(processed.List.W), uintptr(processed.List.H), 0)')
one('y := int32(158) + previewH','y := previewTop + 10 + previewH')
one('labelW := int32(70)','labelW := int32(90)')
one('pathLabelW := int32(90)','pathLabelW := int32(115)')
s=s.replace('procMoveWindow.Call(app.moveBtn, uintptr(headerY), uintptr(by), 145, 34, 0)', 'procMoveWindow.Call(app.moveBtn, 120, uintptr(by), 145, 34, 0)')
result=(head+s+tail).encode()
assert __import__('hashlib').sha256(result).hexdigest()=='8fecb6ce233f78ec178c892d9c7c3ba82ae8183846718ed99de1a7d75fef9453','layout output differs'
p.write_bytes(result)
print('Exact compact layout patch applied')
