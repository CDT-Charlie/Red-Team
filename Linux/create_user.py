import os
import subprocess
import argparse
import sys
import hashlib
import random

def generate_name(length=32):
    alpha = "abcdefghijklmnopqrstuvwxyz1234567890"
    return ''.join(random.choices(alpha, k=length))

def create_user(name, password="password123!", home=True):
    home_flag = "-m" if home else "-M"
    try:
        subprocess.run(
            ['useradd', home_flag, '-s', '/bin/bash', name], 
            check=True, 
            capture_output=True, 
            text=True
        )
        subprocess.run(
            ['chpasswd'], 
            input=f"{name}:{password}", 
            text=True, 
            check=True
        )
    except subprocess.CalledProcessError as e:
        print("Username either exists, bad password, or it is too long/short.")
        print(e)
        pass
    # I was going to do something with hashing, but i forgot what
    name_hash = hashlib.sha256(name.encode('utf-8')).hexdigest()
    return (name, name_hash)

def main():
    if os.geteuid() != 0:
        print("This script must be run as root (sudo).")
        sys.exit(1)

    parser = argparse.ArgumentParser(description="User creation with specific length requirements.")
    parser.add_argument("password", help="The default password for all created users")
    parser.add_argument("-m", action="store_true", help="Create home directory")
    parser.add_argument("--length", type=int, default=32, help="Total length of the username")
    parser.add_argument("--num", type=int, default=1, help="Number of users to create")
    parser.add_argument("--name", type=str, default="-", help="Define an username")

    args = parser.parse_args()

    print(f"Creating {args.num} users (Length: {args.length}, Home: {args.m})...")

    for i in range(args.num):
        create_user(f"{generate_name(args.length) if args.name == "-" else args.name}", args.password, home=args.m)

if __name__ == "__main__":
    main()