#include <windows.h>
#include <shellapi.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define INTERVAL_MS 5000
/* for 5 minutes instead:
#define INTERVAL_MS 300000
*/

static const char *page_urls[2] = {
    "https://tenor.com/view/leevotee-marioqqqqq-cappy-capybara-good-morning-gif-5263475717538580879",
    "https://tenor.com/view/capybara-chewing-chew-pool-water-gif-3515246996644334671"
};

static char html_paths[2][MAX_PATH];
static char gif_paths[2][MAX_PATH];
static int current_gif = 0;

static void setup_console(void) {
    AllocConsole();
    freopen("CONOUT$", "w", stdout);
    freopen("CONOUT$", "w", stderr);
    freopen("CONIN$", "r", stdin);
}

static int file_exists_and_nonzero(const char *path, LARGE_INTEGER *size_out) {
    WIN32_FILE_ATTRIBUTE_DATA fad;
    LARGE_INTEGER sz;

    if (!GetFileAttributesExA(path, GetFileExInfoStandard, &fad)) {
        return 0;
    }
    if (fad.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY) {
        return 0;
    }

    sz.HighPart = (LONG)fad.nFileSizeHigh;
    sz.LowPart  = fad.nFileSizeLow;

    if (size_out) {
        *size_out = sz;
    }
    return sz.QuadPart > 0;
}

static void build_paths(void) {
    char temp[MAX_PATH] = {0};
    DWORD n = GetTempPathA(MAX_PATH, temp);

    if (n == 0 || n >= MAX_PATH) {
        lstrcpyA(temp, ".\\");
    }

    snprintf(html_paths[0], MAX_PATH, "%scapy_page1.html", temp);
    snprintf(html_paths[1], MAX_PATH, "%scapy_page2.html", temp);
    snprintf(gif_paths[0], MAX_PATH, "%scapy1.gif", temp);
    snprintf(gif_paths[1], MAX_PATH, "%scapy2.gif", temp);

    printf("[*] Temp dir : %s\n", temp);
    printf("[*] HTML #1  : %s\n", html_paths[0]);
    printf("[*] HTML #2  : %s\n", html_paths[1]);
    printf("[*] GIF  #1  : %s\n", gif_paths[0]);
    printf("[*] GIF  #2  : %s\n", gif_paths[1]);
}

static int run_powershell(const char *script) {
    STARTUPINFOA si;
    PROCESS_INFORMATION pi;
    char cmd[8192];
    DWORD exit_code = 0;

    ZeroMemory(&si, sizeof(si));
    ZeroMemory(&pi, sizeof(pi));
    si.cb = sizeof(si);

    snprintf(
        cmd,
        sizeof(cmd),
        "powershell.exe -NoProfile -ExecutionPolicy Bypass -Command \"%s\"",
        script
    );

    printf("[*] Running PowerShell:\n%s\n", cmd);

    if (!CreateProcessA(
            NULL,
            cmd,
            NULL,
            NULL,
            FALSE,
            CREATE_NO_WINDOW,
            NULL,
            NULL,
            &si,
            &pi)) {
        printf("[!] CreateProcessA failed for PowerShell. GetLastError=%lu\n", GetLastError());
        return 0;
    }

    WaitForSingleObject(pi.hProcess, INFINITE);

    if (!GetExitCodeProcess(pi.hProcess, &exit_code)) {
        printf("[!] GetExitCodeProcess failed. GetLastError=%lu\n", GetLastError());
        CloseHandle(pi.hThread);
        CloseHandle(pi.hProcess);
        return 0;
    }

    CloseHandle(pi.hThread);
    CloseHandle(pi.hProcess);

    printf("[*] PowerShell exit code: %lu\n", exit_code);
    return exit_code == 0;
}

static int download_to_file_ps(const char *url, const char *dest) {
    char script[8192];
    LARGE_INTEGER sz = {0};

    DeleteFileA(dest);

    snprintf(
        script,
        sizeof(script),
        "$ProgressPreference='SilentlyContinue'; "
        "Invoke-WebRequest -UseBasicParsing -Uri '%s' -OutFile '%s'",
        url, dest
    );

    printf("[*] Downloading:\n    URL : %s\n    OUT : %s\n", url, dest);

    if (!run_powershell(script)) {
        printf("[!] PowerShell download failed.\n");
        return 0;
    }

    if (!file_exists_and_nonzero(dest, &sz)) {
        printf("[!] File missing or zero bytes after download: %s\n", dest);
        return 0;
    }

    printf("[+] Downloaded OK: %s (%lld bytes)\n", dest, sz.QuadPart);
    return 1;
}

static int read_entire_file(const char *path, char **out_buf, size_t *out_len) {
    FILE *f;
    long len;
    char *buf;

    *out_buf = NULL;
    *out_len = 0;

    f = fopen(path, "rb");
    if (!f) {
        printf("[!] fopen failed: %s\n", path);
        return 0;
    }

    if (fseek(f, 0, SEEK_END) != 0) {
        fclose(f);
        return 0;
    }
    len = ftell(f);
    if (len < 0) {
        fclose(f);
        return 0;
    }
    rewind(f);

    buf = (char *)malloc((size_t)len + 1);
    if (!buf) {
        fclose(f);
        return 0;
    }

    if (fread(buf, 1, (size_t)len, f) != (size_t)len) {
        free(buf);
        fclose(f);
        return 0;
    }
    fclose(f);

    buf[len] = '\0';
    *out_buf = buf;
    *out_len = (size_t)len;
    return 1;
}

static void unescape_jsonish_url(char *s) {
    char *src = s;
    char *dst = s;

    while (*src) {
        if (src[0] == '\\' && src[1] == '/') {
            *dst++ = '/';
            src += 2;
        } else if (strncmp(src, "\\u0026", 6) == 0) {
            *dst++ = '&';
            src += 6;
        } else if (src[0] == '\\' && src[1] == '\\') {
            *dst++ = '\\';
            src += 2;
        } else {
            *dst++ = *src++;
        }
    }
    *dst = '\0';
}

static int extract_tenor_gif_url(const char *html_path, char *out_url, size_t out_cap) {
    char *buf = NULL;
    size_t len = 0;
    char *p = NULL;
    char *end = NULL;
    size_t n = 0;

    if (!read_entire_file(html_path, &buf, &len)) {
        return 0;
    }

    p = strstr(buf, "https:\\/\\/media.tenor.com\\/");
    if (!p) {
        p = strstr(buf, "https://media.tenor.com/");
    }
    if (!p) {
        p = strstr(buf, "http:\\/\\/media.tenor.com\\/");
    }
    if (!p) {
        printf("[!] Could not find media.tenor.com URL in HTML.\n");
        free(buf);
        return 0;
    }

    end = strstr(p, ".gif");
    if (!end) {
        printf("[!] Could not find .gif terminator in HTML.\n");
        free(buf);
        return 0;
    }
    end += 4;

    n = (size_t)(end - p);
    if (n + 1 > out_cap) {
        printf("[!] Extracted URL too long.\n");
        free(buf);
        return 0;
    }

    memcpy(out_url, p, n);
    out_url[n] = '\0';
    unescape_jsonish_url(out_url);

    printf("[+] Extracted GIF asset URL:\n    %s\n", out_url);

    free(buf);
    return 1;
}

static int download_gif_from_tenor_page(const char *page_url, const char *html_path, const char *gif_path) {
    char asset_url[4096];

    if (!download_to_file_ps(page_url, html_path)) {
        printf("[!] Failed to download page HTML.\n");
        return 0;
    }

    if (!extract_tenor_gif_url(html_path, asset_url, sizeof(asset_url))) {
        printf("[!] Failed to extract GIF asset URL from page.\n");
        return 0;
    }

    if (!download_to_file_ps(asset_url, gif_path)) {
        printf("[!] Failed to download final GIF asset.\n");
        return 0;
    }

    return 1;
}

static int ensure_all_downloaded(void) {
    int i;
    for (i = 0; i < 2; ++i) {
        printf("\n=== Download pipeline for GIF %d ===\n", i + 1);
        if (!download_gif_from_tenor_page(page_urls[i], html_paths[i], gif_paths[i])) {
            printf("[!] GIF %d failed (Tenor pipeline).\n", i + 1);
            /* Do not fail hard; we will fall back to browser pop-ups. */
        }
    }
    return 1;
}

static void open_current(void) {
    HINSTANCE rc;
    LARGE_INTEGER sz = {0};

    printf("\n[*] Opening GIF index %d\n", current_gif);
    printf("    PATH: %s\n", gif_paths[current_gif]);

    if (!file_exists_and_nonzero(gif_paths[current_gif], &sz)) {
        printf("[!] GIF file missing before ShellExecute.\n");
        return;
    }

    printf("    SIZE: %lld bytes\n", sz.QuadPart);

    rc = ShellExecuteA(
        NULL,
        "open",
        gif_paths[current_gif],
        NULL,
        NULL,
        SW_SHOWNORMAL
    );

    printf("    ShellExecuteA returned: %lld\n", (long long)(INT_PTR)rc);

    if ((INT_PTR)rc <= 32) {
        printf("[!] ShellExecute failed.\n");
    } else {
        printf("[+] Open request sent.\n");
    }

    current_gif = (current_gif + 1) % 2;
}

static void show_popup(void) {
    const char *messages[] = {
        "It is capybara o'clock.",
        "You have been visited by the capybara.",
        "Reminder: take a break and look at capybaras.",
        "Capybara says hi.",
        "Capybara wants your attention."
    };
    int idx;

    srand((unsigned int)GetTickCount());
    idx = rand() % (sizeof(messages) / sizeof(messages[0]));

    MessageBoxA(
        NULL,
        messages[idx],
        "Capybara Time",
        MB_OK | MB_ICONINFORMATION | MB_TOPMOST
    );
}

static void open_page_in_browser(void) {
    HINSTANCE rc;

    printf("\n[*] Opening capybara page index %d\n", current_gif);
    printf("    URL: %s\n", page_urls[current_gif]);

    rc = ShellExecuteA(
        NULL,
        "open",
        page_urls[current_gif],
        NULL,
        NULL,
        SW_SHOWNORMAL
    );

    printf("    ShellExecuteA (URL) returned: %lld\n", (long long)(INT_PTR)rc);

    if ((INT_PTR)rc <= 32) {
        printf("[!] ShellExecute failed for URL.\n");
    } else {
        printf("[+] Browser open request sent.\n");
    }

    current_gif = (current_gif + 1) % 2;
}

int main(void) {
    setup_console();

    printf("=== Capybara annoyance pop-up ===\n");
    printf("[*] Interval: %d ms\n", INTERVAL_MS);

    printf("[*] Press Ctrl+C to stop.\n");

    while (1) {
        show_popup();
        open_page_in_browser();
        Sleep(INTERVAL_MS);
    }

    return 0;
}