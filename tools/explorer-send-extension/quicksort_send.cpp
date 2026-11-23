// DCM QuickSort classic Explorer context-menu extension.
// File selections only; it never moves or opens the selected files itself.
// Registration is deliberately separate and must be qualified in an isolated
// Windows account/VM before an installer may enable it.
// Copyright (c) DCM QuickSort contributors.
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <shlobj.h>
#include <shellapi.h>
#include <objbase.h>
#include <algorithm>
#include <atomic>
#include <cwctype>
#include <string>
#include <vector>
#include <utility>

static HINSTANCE g_module = nullptr;
static std::atomic<long> g_instances{0};
static std::atomic<long> g_locks{0};

struct Favorite { std::wstring name, path; };
static std::wstring trim(const std::wstring& s) {
  auto first = s.find_first_not_of(L" \t\r\n");
  if (first == std::wstring::npos) return L"";
  auto last = s.find_last_not_of(L" \t\r\n");
  return s.substr(first, last-first+1);
}
static std::wstring envPath(const wchar_t* name) {
  DWORD n = GetEnvironmentVariableW(name,nullptr,0);
  if (!n || n > 32767) return L"";
  std::wstring v(n,L'\0');
  DWORD got = GetEnvironmentVariableW(name,v.data(),n);
  if (!got || got>=n) return L"";
  v.resize(got);
  return v;
}
static std::wstring utf8(const std::string& bytes) {
  if(bytes.empty())return L"";
  int n=MultiByteToWideChar(CP_UTF8,MB_ERR_INVALID_CHARS,bytes.data(),int(bytes.size()),nullptr,0);
  if(n<=0)return L"";
  std::wstring w(n,L'\0');
  if(!MultiByteToWideChar(CP_UTF8,MB_ERR_INVALID_CHARS,bytes.data(),int(bytes.size()),w.data(),n))return L"";
  return w;
}
static std::vector<Favorite> readFavorites() {
  std::vector<Favorite> out;
  std::wstring base=envPath(L"LOCALAPPDATA");
  if(base.empty())return out;
  std::wstring path=base+L"\\DCM Quick Sort\\destinations.conf";
  HANDLE file=CreateFileW(path.c_str(),GENERIC_READ,FILE_SHARE_READ|FILE_SHARE_WRITE|FILE_SHARE_DELETE,
     nullptr,OPEN_EXISTING,FILE_ATTRIBUTE_NORMAL,nullptr);
  if(file==INVALID_HANDLE_VALUE)return out;
  LARGE_INTEGER len{};
  if(!GetFileSizeEx(file,&len) || len.QuadPart<0 || len.QuadPart>256*1024) {CloseHandle(file);return out;}
  std::string raw(size_t(len.QuadPart),'\0');
  DWORD read=0;
  BOOL ok=raw.empty() || ReadFile(file,raw.data(),DWORD(raw.size()),&read,nullptr);
  CloseHandle(file);
  if(!ok)return out;
  raw.resize(read);
  std::wstring content=utf8(raw);
  if(content.empty()&&!raw.empty())return out;
  size_t pos=0;
  while(pos<content.size()&&out.size()<48) {
    size_t end=content.find(L'\n',pos);
    if(end==std::wstring::npos)end=content.size();
    std::wstring line=trim(content.substr(pos,end-pos));
    pos=end+1;
    if(line.empty()||line[0]==L'#'||line[0]==L';')continue;
    size_t eq=line.find(L'=');
    if(eq==std::wstring::npos)continue;
    std::wstring name=trim(line.substr(0,eq)),folder=trim(line.substr(eq+1));
    if(name.empty()||folder.empty()||folder.find(L'\0')!=std::wstring::npos)continue;
    if(folder.size()<3 || folder.find(L"..\\")!=std::wstring::npos)continue;
    DWORD attributes=GetFileAttributesW(folder.c_str());
    if(attributes==INVALID_FILE_ATTRIBUTES || !(attributes&FILE_ATTRIBUTE_DIRECTORY))continue;
    bool duplicate=false;
    for(auto& favorite:out) if(_wcsicmp(favorite.name.c_str(),name.c_str())==0){duplicate=true;break;}
    if(!duplicate)out.push_back({name,folder});
  }
  return out;
}
static std::vector<std::wstring> selectedFiles(IDataObject* object) {
  std::vector<std::wstring> files;
  if(!object)return files;
  FORMATETC fmt{CF_HDROP,nullptr,DVASPECT_CONTENT,-1,TYMED_HGLOBAL};
  STGMEDIUM medium{};
  if(FAILED(object->GetData(&fmt,&medium)))return files;
  HDROP drop=static_cast<HDROP>(GlobalLock(medium.hGlobal));
  if(drop){
    UINT count=DragQueryFileW(drop,0xffffffff,nullptr,0);
    if(count>0 && count<=256) {
      bool valid=true;
      for(UINT i=0;i<count;i++){
        UINT chars=DragQueryFileW(drop,i,nullptr,0);
        if(!chars || chars>=32760){valid=false;break;}
        std::wstring path(chars+1,L'\0');
        if(DragQueryFileW(drop,i,path.data(),chars+1)!=chars){valid=false;break;}
        path.resize(chars);
        DWORD attrs=GetFileAttributesW(path.c_str());
        if(attrs==INVALID_FILE_ATTRIBUTES || (attrs&(FILE_ATTRIBUTE_DIRECTORY|FILE_ATTRIBUTE_REPARSE_POINT))){valid=false;break;}
        files.push_back(std::move(path));
      }
      if(!valid)files.clear();
    }
    GlobalUnlock(medium.hGlobal);
  }
  ReleaseStgMedium(&medium);
  return files;
}
static std::wstring quoteArg(const std::wstring& v){
  std::wstring result=L"\"";
  size_t slash=0;
  for(wchar_t c:v){
    if(c==L'\\'){++slash;continue;}
    if(c==L'"'){result.append(slash*2+1,L'\\');result+=c;slash=0;continue;}
    result.append(slash,L'\\');slash=0;result+=c;
  }
  result.append(slash*2,L'\\');
  return result+L"\"";
}
static bool launchBatch(const Favorite& favorite,const std::vector<std::wstring>& files){
  std::wstring base=envPath(L"LOCALAPPDATA");
  if(base.empty())return false;
  std::wstring app=base+L"\\Programs\\DCM Quick Sort\\DCM-QuickSort.exe";
  std::wstring helper=base+L"\\Programs\\DCM Quick Sort\\DCMQuickSortExplorerSend.exe";
  if(GetFileAttributesW(app.c_str())==INVALID_FILE_ATTRIBUTES ||
     GetFileAttributesW(helper.c_str())==INVALID_FILE_ATTRIBUTES)return false;
  std::wstring cmd=quoteArg(helper)+L" -app "+quoteArg(app)+L" -alias "+quoteArg(favorite.name)+L" --";
  for(auto& path:files){
    cmd+=L" "+quoteArg(path);
    if(cmd.size()>30000)return false; // Windows CreateProcess command-line bound
  }
  std::vector<wchar_t> buffer(cmd.begin(),cmd.end());
  buffer.push_back(L'\0');
  STARTUPINFOW si{};si.cb=sizeof(si);
  PROCESS_INFORMATION pi{};
  BOOL ok=CreateProcessW(helper.c_str(),buffer.data(),nullptr,nullptr,FALSE,
     CREATE_NO_WINDOW,nullptr,nullptr,&si,&pi);
  if(ok){CloseHandle(pi.hThread);CloseHandle(pi.hProcess);}
  return ok!=FALSE;
}
class Menu final : public IShellExtInit, public IContextMenu {
  std::atomic<ULONG> refs{1};
  std::vector<std::wstring> files;
  std::vector<Favorite> entries;
public:
  Menu(){++g_instances;}
  ~Menu()override{--g_instances;}
  HRESULT STDMETHODCALLTYPE QueryInterface(REFIID riid,void** ppv)override{
    if(!ppv)return E_POINTER;
    *ppv=nullptr;
    if(IsEqualIID(riid,IID_IUnknown)||IsEqualIID(riid,IID_IShellExtInit))
      *ppv=static_cast<IShellExtInit*>(this);
    else if(IsEqualIID(riid,IID_IContextMenu))
      *ppv=static_cast<IContextMenu*>(this);
    if(!*ppv)return E_NOINTERFACE;
    AddRef();return S_OK;
  }
  ULONG STDMETHODCALLTYPE AddRef()override{return ++refs;}
  ULONG STDMETHODCALLTYPE Release()override{
    ULONG n=--refs;if(!n)delete this;return n;
  }
  HRESULT STDMETHODCALLTYPE Initialize(PCIDLIST_ABSOLUTE,IDataObject* data,HKEY)override{
    files=selectedFiles(data);
    entries.clear();
    if(!files.empty())entries=readFavorites();
    return S_OK;
  }
  HRESULT STDMETHODCALLTYPE QueryContextMenu(HMENU menu,UINT position,UINT first,UINT last,UINT flags)override{
    if(flags&CMF_DEFAULTONLY || files.empty() || entries.empty() || first>last)return MAKE_HRESULT(SEVERITY_SUCCESS,0,0);
    size_t capacity=size_t(last-first)+1;
    if(entries.size()>capacity)entries.resize(capacity);
    HMENU child=CreatePopupMenu();
    if(!child)return E_OUTOFMEMORY;
    for(size_t i=0;i<entries.size();i++){
      if(!AppendMenuW(child,MF_STRING,first+UINT(i),entries[i].name.c_str())){
        DestroyMenu(child);return E_FAIL;
      }
    }
    if(!InsertMenuW(menu,position,MF_BYPOSITION|MF_POPUP,
       reinterpret_cast<UINT_PTR>(child),L"Send to QuickSort Destination")){
      DestroyMenu(child);return E_FAIL;
    }
    return MAKE_HRESULT(SEVERITY_SUCCESS,0,UINT(entries.size()));
  }
  HRESULT STDMETHODCALLTYPE InvokeCommand(LPCMINVOKECOMMANDINFO command)override{
    if(!command || HIWORD(command->lpVerb))return E_INVALIDARG;
    unsigned index=LOWORD(command->lpVerb);
    if(index>=entries.size()||files.empty())return E_INVALIDARG;
    if(!launchBatch(entries[index],files)){
      MessageBoxW(command->hwnd,L"QuickSort could not start the batch handoff. Check the installed helper and selection size. No files were moved.",
        L"QuickSort Explorer integration",MB_OK|MB_ICONWARNING);
      return E_FAIL;
    }
    return S_OK;
  }
  HRESULT STDMETHODCALLTYPE GetCommandString(UINT_PTR index,UINT flags,UINT*,LPSTR out,UINT max)override{
    if(index>=entries.size()||!out||!max)return E_INVALIDARG;
    if(flags==GCS_VERBW){
      std::wstring verb=L"QuickSortSend"+std::to_wstring(index);
      wcsncpy_s(reinterpret_cast<wchar_t*>(out),max,verb.c_str(),_TRUNCATE);
      return S_OK;
    }
    if(flags==GCS_VERBA){
      std::string verb="QuickSortSend"+std::to_string(index);
      strncpy_s(out,max,verb.c_str(),_TRUNCATE);
      return S_OK;
    }
    return E_NOTIMPL;
  }
};
class Factory final:public IClassFactory{
 std::atomic<ULONG> refs{1};
public:
 HRESULT STDMETHODCALLTYPE QueryInterface(REFIID iid,void** out)override{
  if(!out)return E_POINTER;
  *out=nullptr;
  if(IsEqualIID(iid,IID_IUnknown)||IsEqualIID(iid,IID_IClassFactory))*out=static_cast<IClassFactory*>(this);
  if(!*out)return E_NOINTERFACE;
  AddRef();return S_OK;
 }
 ULONG STDMETHODCALLTYPE AddRef()override{return ++refs;}
 ULONG STDMETHODCALLTYPE Release()override{ULONG n=--refs;if(!n)delete this;return n;}
 HRESULT STDMETHODCALLTYPE CreateInstance(IUnknown* parent,REFIID iid,void** out)override{
  if(parent)return CLASS_E_NOAGGREGATION;
  Menu* obj=new(std::nothrow) Menu();
  if(!obj)return E_OUTOFMEMORY;
  HRESULT hr=obj->QueryInterface(iid,out);
  obj->Release();return hr;
 }
 HRESULT STDMETHODCALLTYPE LockServer(BOOL lock)override{
  if(lock)++g_locks;else --g_locks;
  return S_OK;
 }
};
// DCM QuickSort Explorer Send CLSID (owned by this project).
static const CLSID CLSID_QuickSortSend=
 {0x8f2da0c1,0x7d79,0x40aa,{0xa0,0x30,0x19,0x64,0x5d,0x8c,0x4a,0x91}};
extern "C" BOOL WINAPI DllMain(HINSTANCE instance,DWORD reason,LPVOID){
 if(reason==DLL_PROCESS_ATTACH){g_module=instance;DisableThreadLibraryCalls(instance);}
 return TRUE;
}
extern "C" __declspec(dllexport) HRESULT STDAPICALLTYPE DllGetClassObject(REFCLSID clsid,REFIID iid,void** out){
 if(!IsEqualCLSID(clsid,CLSID_QuickSortSend))return CLASS_E_CLASSNOTAVAILABLE;
 Factory* factory=new(std::nothrow) Factory();
 if(!factory)return E_OUTOFMEMORY;
 HRESULT hr=factory->QueryInterface(iid,out);factory->Release();return hr;
}
extern "C" __declspec(dllexport) HRESULT STDAPICALLTYPE DllCanUnloadNow(){
 return g_instances.load()==0 && g_locks.load()==0 ? S_OK:S_FALSE;
}
