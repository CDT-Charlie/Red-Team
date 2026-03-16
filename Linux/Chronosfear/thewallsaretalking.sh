#!/bin/bash

MESSAGE="$*"

for tty in /dev/pts/* /dev/tty*; do
    if [ -w "$tty" ]; then
        echo -e "\n[System Notice] $MESSAGE\n" > "$tty"
    fi
done