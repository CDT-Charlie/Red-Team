#include <windows.h>
#include <wincrypt.h>

char* encodeForPowerShell(const char* command) {
    int wideSize = MultiByteToWideChar(CP_ACP, 0, command, -1, NULL, 0);
    wchar_t* wideCmd = (wchar_t*)malloc(wideSize * sizeof(wchar_t));
    MultiByteToWideChar(CP_ACP, 0, command, -1, wideCmd, wideSize);
    DWORD b64Size = 0;
    int byteLen = (wideSize - 1) * sizeof(wchar_t); 
    CryptBinaryToStringA((BYTE*)wideCmd, byteLen, CRYPT_STRING_BASE64 | CRYPT_STRING_NOCRLF, NULL, &b64Size);
    char* b64Result = (char*)malloc(b64Size);
    CryptBinaryToStringA((BYTE*)wideCmd, byteLen, CRYPT_STRING_BASE64 | CRYPT_STRING_NOCRLF, b64Result, &b64Size);
    free(wideCmd);
    return b64Result;
}