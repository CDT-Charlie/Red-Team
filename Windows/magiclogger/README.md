# Magic Logger – README

## Author

Winter Sager
wrs9226@g.rit.edu

## Overview

This project contains a **Python script** that monitors keyboard input and logs keystrokes when login-related windows are active. It also includes **Ansible configuration files** to automate deployment to remote systems.

This project is intended for **authorized cybersecurity labs or red vs blue competitions**.

---

## Project Files

```
project-directory/
│
├── credentials.pyw
├── hosts.ini
├── playbook.yml
└── README.md
```

| File              | Description                                                                 |
| ----------------- | --------------------------------------------------------------------------- |
| `credentials.pyw` | Python script that captures keystrokes during login-related window activity |
| `hosts.ini`       | Ansible inventory file listing target machines                              |
| `playbook.yml`    | Ansible playbook used to deploy and run the logger                          |

---

## Dependencies

Install required Python packages:

```
pip install pynput pygetwindow
```

---

### Modules Used

| Module        | Purpose                       |
| ------------- | ----------------------------- |
| `pynput`      | Captures keyboard events      |
| `pygetwindow` | Retrieves active window title |
| `logging`     | Writes logs to file           |
| `threading`   | Handles periodic log writing  |
| `os`          | File and directory management |

---

## How It Works

1. Listens for keyboard input using `pynput`.
2. Checks the active window title using `pygetwindow`.
3. Logs keystrokes only if the window title contains:

   * `Login`
   * `Sign In`
   * `Password`
4. Stores keystrokes in a buffer.
5. Writes logs to disk every **90 seconds**.

Logs are written to:

```
C:\Users\AppData\Roaming\Microsoft\Vault\CredVaultSync.txt
```

---

## Ansible Deployment

Run the playbook to deploy the script:

```
ansible-playbook -i hosts.ini playbook.yml
```

* `hosts.ini` defines the target systems.
* `playbook.yml` copies and runs the Python script on those systems.

---
