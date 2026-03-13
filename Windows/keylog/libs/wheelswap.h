#include <windows.h>
#ifndef WHEEL_SWAP
#define WHEEL_SWAP
#endif

extern char SwapKey1;
extern char SwapKey2;

DWORD GetVK(char c);
LRESULT CALLBACK SwapKeys(int nCode, WPARAM wParam, LPARAM lParam);