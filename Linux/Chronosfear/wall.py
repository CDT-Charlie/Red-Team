import socket
import os
import threading
import glob

def handle_conn(conn):
    ttys = glob.glob("/dev/pts/*") + glob.glob("/dev/tty*")
    while True: 
        message = conn.recv(1024).decode()
        for tty in ttys:
            try:
                if os.access(tty, os.W_OK):
                    with open(tty, "w") as f:
                        f.write(f"\n{message}\n")
            except Exception:
                conn.close()

def start_server():
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.bind(("0.0.0.0", 1423))
    sock.listen()
    while True:
        conn, addr = sock.accept()
        threading.Thread(target=handle_conn, args=conn, daemon=True).start()

start_server()