#!/bin/bash

MESSAGE="$*"

for tty in /dev/pts/* /dev/tty*; do
    if [ -w "$tty" ]; then
        echo -e "\n$MESSAGE\n" > "$tty"
    fi
done