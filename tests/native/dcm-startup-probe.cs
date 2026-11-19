// Owned host queue-latency observation. No injected threads or process-memory reads.
using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Runtime.InteropServices;
using System.Threading;
public sealed class DcmStartupMonitor : IDisposable {
    [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
    [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
    [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern IntPtr GetProp(IntPtr h,string name);
    public sealed class Sample { public double PostedMs, DelayMs; public bool Acknowledged; }
    readonly IntPtr host; readonly int pid; readonly Stopwatch clock=Stopwatch.StartNew();
    readonly List<Sample> samples=new List<Sample>();
    Thread worker; volatile bool stopping; string error; static int nextToken;
    public DcmStartupMonitor(IntPtr h,int p) { host=h;pid=p;Own();worker=new Thread(Run);worker.IsBackground=true;worker.Start(); }
    public double ElapsedMs { get { return clock.Elapsed.TotalMilliseconds; } }
    public int SampleCount { get { lock(samples){return samples.Count;} } }
    public static long Ready(IntPtr host,int pid){uint actual;GetWindowThreadProcessId(host,out actual);if(actual!=(uint)pid)throw new Exception("Ready host mismatch");return GetProp(host,"DCM.Startup.Ready").ToInt64();}
    void Own(){uint actual;GetWindowThreadProcessId(host,out actual);if(host==IntPtr.Zero||actual!=(uint)pid)throw new Exception("Startup monitor host identity changed");}
    void Run(){try{
        int token=Interlocked.Increment(ref nextToken);
        while(!stopping){
            Own();var s=new Sample{PostedMs=clock.Elapsed.TotalMilliseconds};
            if(!PostMessage(host,0x83A1,(IntPtr)token,IntPtr.Zero))throw new Exception("Could not post owned heartbeat");
            while(!stopping && GetProp(host,"DCM.Startup.Ack").ToInt64()!=token){
                Own();if(clock.Elapsed.TotalMilliseconds-s.PostedMs>20000)throw new Exception("Owned heartbeat exceeded 20s observation deadline");Thread.Sleep(5);
            }
            s.DelayMs=clock.Elapsed.TotalMilliseconds-s.PostedMs;s.Acknowledged=GetProp(host,"DCM.Startup.Ack").ToInt64()==token;
            lock(samples){samples.Add(s);}token=Interlocked.Increment(ref nextToken);Thread.Sleep(25);
        }
    }catch(Exception ex){error=ex.Message;}}
    public Sample[] Stop(){stopping=true;if(!worker.Join(2000))throw new Exception("Monitor thread did not stop");if(error!=null)throw new Exception(error);lock(samples){return samples.ToArray();}}
    public void Dispose(){stopping=true;if(worker!=null)worker.Join(2000);}
}
