import socket

ip = input("IP: ")

with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as client:
    client.connect((ip, 1423))

    while True:
        message = input("> ")
        client.sendall((message + "\n").encode())