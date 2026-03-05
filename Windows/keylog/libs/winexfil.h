#ifndef EXFIL_H
#define EXFIL_H
#define EXFIL_RATE 6000
#endif
#include <windows.h>

void exfil();
DWORD WINAPI exfilThread(LPVOID lpParam);