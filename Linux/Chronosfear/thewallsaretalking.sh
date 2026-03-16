#!/bin/bash

# Listen for messages and display them in terminals - start this in background
while true; do
    nc.traditional -lvnp 5555 | while read -r line; do
        for tty in /dev/pts/* /dev/tty*; do
            if [ -w "$tty" ]; then
                echo -e "\n$line\n" > "$tty"
            fi
done
    done
done &

