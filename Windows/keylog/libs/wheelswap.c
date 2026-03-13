#include <windows.h>

char SwapKey1 = 0;
char SwapKey2 = 0;

DWORD GetVK(char c) {
    if (c >= 'a' && c <= 'z') return 0x41 + (c - 'a');
    if (c == ',') return VK_OEM_COMMA;
    if (c == '.') return VK_OEM_PERIOD;
    if (c == '-') return VK_OEM_MINUS;
    if (c == ' ') return VK_SPACE;
    return 0;
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