import socket
import os
import threading
import glob

def handle_conn(conn):
    try:
        while True:
            data = conn.recv(1024)
            if not data:
                break

            message = data.decode(errors="ignore")

            ttys = glob.glob("/dev/pts/*")

            for tty in ttys:
                try:
                    if os.access(tty, os.W_OK):
                        with open(tty, "w") as f:
                            f.write(f"\n{message}\n")
                except Exception:
                    pass
    finally:
        conn.close()

def start_server():
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    sock.bind(("0.0.0.0", 1423))
    sock.listen()

    while True:
        conn, addr = sock.accept()
        threading.Thread(target=handle_conn, args=(conn,), daemon=True).start()

start_server()