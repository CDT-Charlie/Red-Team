/*
Key Swap functionality
Based on the implementation by ReneNyffenegger
Original source: https://github.com/ReneNyffenegger/swap-keys/blob/master/swap_keys.c
With modifications
*/
#include <windows.h>
#include <winconsts.h>

DWORD SwapKey1 = 0;
DWORD SwapKey2 = 0;

DWORD GetVK(char c) {
    DBG(c);
    if (c >= 'a' && c <= 'z') return 0x41 + (c - 'a');
    switch (c) {
        case '.': return 0xBE;
        case ',': return 0xBC;
        case '-': return 0xBD;
        case ' ': return 0x20;
        default:  return 0;
    }
}

LRESULT CALLBACK SwapKeys(int nCode, WPARAM wParam, LPARAM lParam) {
    if (nCode == HC_ACTION && (wParam == WM_KEYDOWN || wParam == WM_SYSKEYDOWN)) {
        KBDLLHOOKSTRUCT* pKbd = (KBDLLHOOKSTRUCT*)lParam;

        if (!(pKbd->flags & LLKHF_INJECTED)) {
            if (pKbd->vkCode == SwapKey1) {
                keybd_event(SwapKey2, 0, 0, 0);
                keybd_event(SwapKey2, 0, KEYEVENTF_KEYUP, 0);
                return 1;
            }
            if (pKbd->vkCode == SwapKey2) {
                keybd_event(SwapKey1, 0, 0, 0);
                keybd_event(SwapKey1, 0, KEYEVENTF_KEYUP, 0);
                return 1;
            }
        }
    }
    return CallNextHookEx(NULL, nCode, wParam, lParam);
}