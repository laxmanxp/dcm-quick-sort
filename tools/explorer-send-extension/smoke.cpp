// Isolated COM qualification for DCMQuickSortExplorerSend.dll.
// Loads the DLL directly, with a disposable LOCALAPPDATA and synthetic CF_HDROP.
// Does NOT register a Windows shell extension, invoke any move or edit user files.
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <shlobj.h>
#include <shellapi.h>
#include <objbase.h>
#include <filesystem>
#include <fstream>
#include <string>
#include <vector>
#include <iostream>
#include <atomic>

class Drop final : public IDataObject {
 std::atomic<ULONG> refs{1};
 std::vector<std::wstring> items;
public:
 explicit Drop(std::vector<std::wstring> files):items(std::move(files)){}
 HRESULT STDMETHODCALLTYPE QueryInterface(REFIID iid,void** out)override{
  if(!out)return E_POINTER;*out=nullptr;
  if(IsEqualIID(iid,IID_IUnknown)||IsEqualIID(iid,IID_IDataObject))*out=static_cast<IDataObject*>(this);
  if(!*out)return E_NOINTERFACE;AddRef();return S_OK;
 }
 ULONG STDMETHODCALLTYPE AddRef()override{return ++refs;}
 ULONG STDMETHODCALLTYPE Release()override{ULONG n=--refs;if(!n)delete this;return n;}
 HRESULT STDMETHODCALLTYPE GetData(FORMATETC* fmt,STGMEDIUM* medium)override{
  if(!fmt||!medium)return E_POINTER;
  if(fmt->cfFormat!=CF_HDROP || !(fmt->tymed&TYMED_HGLOBAL))return DV_E_FORMATETC;
  size_t bytes=sizeof(DROPFILES)+sizeof(wchar_t);
  for(auto& item:items)bytes+=(item.size()+1)*sizeof(wchar_t);
  HGLOBAL h=GlobalAlloc(GMEM_MOVEABLE,bytes);
  if(!h)return E_OUTOFMEMORY;
  void* ptr=GlobalLock(h);
  if(!ptr){GlobalFree(h);return E_OUTOFMEMORY;}
  auto* drop=reinterpret_cast<DROPFILES*>(ptr);*drop=DROPFILES{};
  drop->pFiles=sizeof(DROPFILES);drop->fWide=TRUE;
  wchar_t* buf=reinterpret_cast<wchar_t*>(reinterpret_cast<char*>(ptr)+sizeof(DROPFILES));
  for(auto& item:items){wcscpy_s(buf,item.size()+1,item.c_str());buf+=item.size()+1;}
  *buf=L'\0';
  GlobalUnlock(h);
  medium->tymed=TYMED_HGLOBAL;medium->hGlobal=h;medium->pUnkForRelease=nullptr;
  return S_OK;
 }
 HRESULT STDMETHODCALLTYPE GetDataHere(FORMATETC*,STGMEDIUM*)override{return E_NOTIMPL;}
 HRESULT STDMETHODCALLTYPE QueryGetData(FORMATETC* fmt)override{return fmt&&fmt->cfFormat==CF_HDROP?S_OK:DV_E_FORMATETC;}
 HRESULT STDMETHODCALLTYPE GetCanonicalFormatEtc(FORMATETC*,FORMATETC*)override{return E_NOTIMPL;}
 HRESULT STDMETHODCALLTYPE SetData(FORMATETC*,STGMEDIUM*,BOOL)override{return E_NOTIMPL;}
 HRESULT STDMETHODCALLTYPE EnumFormatEtc(DWORD,IEnumFORMATETC**)override{return E_NOTIMPL;}
 HRESULT STDMETHODCALLTYPE DAdvise(FORMATETC*,DWORD,IAdviseSink*,DWORD*)override{return E_NOTIMPL;}
 HRESULT STDMETHODCALLTYPE DUnadvise(DWORD)override{return E_NOTIMPL;}
 HRESULT STDMETHODCALLTYPE EnumDAdvise(IEnumSTATDATA**)override{return E_NOTIMPL;}
};
static const CLSID CLSID_QuickSortSend=
 {0x8f2da0c1,0x7d79,0x40aa,{0xa0,0x30,0x19,0x64,0x5d,0x8c,0x4a,0x91}};
static void require(bool ok,const char* msg){if(!ok){std::cerr<<"FAIL: "<<msg<<"\n";ExitProcess(2);}}
static std::wstring menuLabel(HMENU menu,UINT pos){
 wchar_t buf[256]{};
 GetMenuStringW(menu,pos,buf,256,MF_BYPOSITION);
 return buf;
}
int wmain(int argc,wchar_t** argv){
 require(argc==4,"usage: smoke <dll> <helper-exe> <receiver-fixture-exe>");
 HRESULT init=CoInitializeEx(nullptr,COINIT_APARTMENTTHREADED);
 require(SUCCEEDED(init),"COM init");
 auto root=std::filesystem::temp_directory_path() / (L"DCMQuickSortShellIsolated-"+std::to_wstring(GetCurrentProcessId()));
 auto config=root/L"DCM Quick Sort";
 std::filesystem::create_directories(config);
 auto accounts=root/L"Accounts Records";
 auto letters=root/L"Letters";
 std::filesystem::create_directories(accounts);
 std::filesystem::create_directories(letters);
 auto install=root/L"Programs"/L"DCM Quick Sort";
 std::filesystem::create_directories(install);
 std::filesystem::copy_file(argv[2],install/L"DCMQuickSortExplorerSend.exe",std::filesystem::copy_options::overwrite_existing);
 std::filesystem::copy_file(argv[3],install/L"DCM-QuickSort.exe",std::filesystem::copy_options::overwrite_existing);
 auto fileA=root/L"first selected.txt",fileB=root/L"second file.png";
 {std::ofstream(fileA).put('a');std::ofstream(fileB).put('b');}
 // Use ASCII-only disposable test paths to simplify UTF8 fixture serialization.
 std::ofstream cfg(config/L"destinations.conf",std::ios::binary);
 cfg<<"Accounts="<<accounts.string()<<"\nCorrespondence="<<letters.string()<<"\nUnavailable="<< (root/L"not-found").string()<<"\n";
 cfg.close();
 wchar_t previous[32768]{};
 DWORD prev=GetEnvironmentVariableW(L"LOCALAPPDATA",previous,32768);
 require(SetEnvironmentVariableW(L"LOCALAPPDATA",root.c_str())!=0,"isolate LOCALAPPDATA");
 HMODULE dll=LoadLibraryW(argv[1]);
 require(dll!=nullptr,"LoadLibrary DLL");
 using GetClass=HRESULT (STDAPICALLTYPE *)(REFCLSID,REFIID,void**);
 auto getter=reinterpret_cast<GetClass>(GetProcAddress(dll,"DllGetClassObject"));
 require(getter!=nullptr,"DllGetClassObject exported");
 IClassFactory* factory=nullptr;
 require(SUCCEEDED(getter(CLSID_QuickSortSend,IID_IClassFactory,reinterpret_cast<void**>(&factory))),"COM class factory");
 IShellExtInit* ext=nullptr;
 require(SUCCEEDED(factory->CreateInstance(nullptr,IID_IShellExtInit,reinterpret_cast<void**>(&ext))),"create shell extension");
 IContextMenu* ctx=nullptr;
 require(SUCCEEDED(ext->QueryInterface(IID_IContextMenu,reinterpret_cast<void**>(&ctx))),"IContextMenu interface");
 Drop* selected=new Drop({fileA.wstring(),fileB.wstring()});
 require(SUCCEEDED(ext->Initialize(nullptr,selected,nullptr)),"initialize from 2 CF_HDROP paths");
 selected->Release();
 HMENU top=CreatePopupMenu();
 require(top!=nullptr,"create test menu");
 HRESULT hr=ctx->QueryContextMenu(top,0,100,200,CMF_NORMAL);
 require(SUCCEEDED(hr)&&HRESULT_CODE(hr)==2,"two valid favorites consumed two command IDs");
 require(GetMenuItemCount(top)==1,"one parent submenu added");
 require(menuLabel(top,0)==L"Send to QuickSort Destination","parent title");
 HMENU sub=GetSubMenu(top,0);
 require(sub!=nullptr&&GetMenuItemCount(sub)==2,"two dynamic child items");
 require(menuLabel(sub,0)==L"Accounts" && menuLabel(sub,1)==L"Correspondence","favorite order and labels");
 require(GetMenuItemID(sub,0)==100 && GetMenuItemID(sub,1)==101,"child command offset");
 std::cout<<"PASS: 2 selected files, 2 live destination favorites, one dynamic submenu\n";
 CMINVOKECOMMANDINFO invoke{};
 invoke.cbSize=sizeof(invoke);
 invoke.lpVerb=MAKEINTRESOURCEA(0); // Accounts, first submenu item
 require(SUCCEEDED(ctx->InvokeCommand(&invoke)),"invoke Accounts selection");
 auto receipt=root/L"shell-lab-receipt.json";
 for(int i=0;i<100 && !std::filesystem::exists(receipt);i++)Sleep(100);
 require(std::filesystem::exists(receipt),"batch helper delivered single manifest to harmless receiver");
 std::ifstream in(receipt,std::ios::binary);
 std::string payload((std::istreambuf_iterator<char>(in)),std::istreambuf_iterator<char>());
 in.close();
 require(payload.find("\"alias_name\":\"Accounts\"")!=std::string::npos,"selected alias carried");
 require(payload.find("first selected.txt")!=std::string::npos && payload.find("second file.png")!=std::string::npos,"all selected file paths carried");
 require(std::filesystem::file_size(fileA)==1 && std::filesystem::file_size(fileB)==1,"source files unchanged");
 std::cout<<"PASS: submenu command launched one helper handoff carrying both files; fixture source bytes unchanged\n";
 DestroyMenu(top);
 Drop* folder=new Drop({accounts.wstring()});
 require(SUCCEEDED(ext->Initialize(nullptr,folder,nullptr)),"initialize folder selection");
 folder->Release();
 top=CreatePopupMenu();
 hr=ctx->QueryContextMenu(top,0,100,200,CMF_NORMAL);
 require(SUCCEEDED(hr)&&GetMenuItemCount(top)==0,"folder selection suppressed");
 DestroyMenu(top);
 std::cout<<"PASS: folder selection is excluded\n";
 // QueryContextMenu must honor the available command-ID range and default-only mode.
 Drop* again=new Drop({fileA.wstring(),fileB.wstring()});
 require(SUCCEEDED(ext->Initialize(nullptr,again,nullptr)),"reinitialize file selection");
 again->Release();
 top=CreatePopupMenu();
 hr=ctx->QueryContextMenu(top,0,100,100,CMF_NORMAL);
 require(SUCCEEDED(hr)&&HRESULT_CODE(hr)==1,"one available ID permits one destination");
 require(GetMenuItemCount(GetSubMenu(top,0))==1,"command limit clips submenu");
 DestroyMenu(top);
 top=CreatePopupMenu();
 hr=ctx->QueryContextMenu(top,0,100,200,CMF_DEFAULTONLY);
 require(SUCCEEDED(hr)&&HRESULT_CODE(hr)==0&&GetMenuItemCount(top)==0,"default-only menu suppressed");
 DestroyMenu(top);
 invoke.lpVerb=MAKEINTRESOURCEA(77);
 require(ctx->InvokeCommand(&invoke)==E_INVALIDARG,"invalid command offset refused");
 std::cout<<"PASS: bounded command IDs, default-only suppression, invalid verb refusal\\n";
 // Unavailable destinations and vanished selected files must not create a handoff.
 std::filesystem::remove(config/L"destinations.conf");
 Drop* noFavorites=new Drop({fileA.wstring(),fileB.wstring()});
 require(SUCCEEDED(ext->Initialize(nullptr,noFavorites,nullptr)),"reinitialize without destination config");
 noFavorites->Release();
 top=CreatePopupMenu();
 hr=ctx->QueryContextMenu(top,0,100,200,CMF_NORMAL);
 require(SUCCEEDED(hr)&&GetMenuItemCount(top)==0,"missing favorites hide submenu");
 DestroyMenu(top);
 std::cout<<"PASS: absent destination configuration hides submenu\\n";
 ctx->Release();ext->Release();factory->Release();
 auto canUnload=reinterpret_cast<HRESULT (STDAPICALLTYPE *)()>(GetProcAddress(dll,"DllCanUnloadNow"));
 require(canUnload && canUnload()==S_OK,"COM object released");
 FreeLibrary(dll);
 if(prev>0&&prev<32768)SetEnvironmentVariableW(L"LOCALAPPDATA",previous);
 else SetEnvironmentVariableW(L"LOCALAPPDATA",nullptr);
 std::error_code cleanupError;
 for(int i=0;i<50;i++){
   cleanupError.clear();
   std::filesystem::remove_all(root,cleanupError);
   if(!cleanupError || !std::filesystem::exists(root))break;
   Sleep(100);
 }
 require(!std::filesystem::exists(root),"disposable fixture cleanup");

 CoUninitialize();
 std::cout<<"PASS: unloaded DLL and removed disposable fixture\n";
 return 0;
}
