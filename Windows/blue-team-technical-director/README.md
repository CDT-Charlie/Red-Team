# **PROJECT: ARP-Based RMM**

### **Internal Engineering Challenge: Phase 1 (MVP)**

## **1\. Objective**

To build a functional **Remote Monitoring and Management (RMM)** tool that operates exclusively at **Layer 2 (Data Link Layer)** using the Address Resolution Protocol (ARP).

This challenge is designed to force our Linux administrators to master low-level networking in **Go**, bypass traditional Layer 3 (IP) filtering, and manage Windows Server 2022 endpoints without relying on standard TCP/IP sockets or WinRM for long-term management.

**The Goal:** Prove that you can execute a command on a Windows Server and receive the output on a Linux box using *only* Ethernet frames.

---

## **2\. The Architecture (The "How")**

Traditional RMMs use HTTPS or SSH (Layer 3/4). This tool moves data inside the metadata fields of ARP packets—specifically the **Sender Hardware Address (SHA)** and **Sender Protocol Address (SPA)**.

### **MVP Communication Flow:**

1. **Command (Linux \-\> Windows):** The Admin box broadcasts an ARP Request. The "Target IP" field in the packet is actually a 4-byte encoded command string (e.g., `INIT`).  
2. **Execution (Windows):** The Go-based Agent sniffs the raw Ethernet frame, decodes the "IP" back into a string, and triggers a local PowerShell script.  
3. **Response (Windows \-\> Linux):** The Agent broadcasts an ARP Reply. The "Sender IP" field contains the first 4 bytes of the PowerShell output.

---

## **3\. Technology Stack**

* **Control Node:** Linux (Ubuntu/Debian)  
* **Managed Endpoints:** Windows Server 2022 (Standard/Datacenter)  
* **Core Language:** Go (Golang) — chosen for its `gopacket` library and static cross-compilation.  
* **Driver Requirement:** Npcap (WinPcap compatible) must be installed on Windows targets for raw packet access.  
* **Deployment:** Ansible (via WinRM) for the initial "bootstrapping" of the service.

---

## **4\. Phase 1: The MVP Requirements**

To pass the first phase of this challenge, your team must demonstrate the **"ARP Shell"**:

* \[ \] **Handshake:** Linux box sends a "HELO" ARP packet; Windows responds with "READY".  
* \[ \] **Unidirectional Command:** Successfully trigger `Restart-Service W3SVC` via an ARP frame.  
* \[ \] **Bidirectional Data:** Execute `hostname` and see the result printed on the Linux terminal.  
* \[ \] **Persistence:** The Windows Agent must run as a background service (using NSSM) and survive a reboot.

---

## **5\. Deployment Instructions (Ansible)**

We use Ansible to push the RMM agent.

Bash

\# From the Linux Admin Box

ansible-playbook \-i inventory.ini site.yml

This will:

1. Create `C:\ProgramData\WinNetExt`  
2. Deploy `agent.exe` and `nssm.exe`  
3. Register the **"Windows Network Extension"** service.

---

## **6\. Phase 2: Future Roadmap (v2.0)**

Once the MVP is stabilized, we will move to the "Hardened" version of the tool:

* **MAC Hopping:** Randomize the Source MAC for every fragment to look like network noise.  
* **Fragmentation Engine:** Support for large data transfers across hundreds of ARP packets.  
* **Encryption:** Implement XXTEA or AES-GCM on the 4-byte payloads.  
* **Self-Destruct:** A "Kill-Packet" sequence that wipes the agent from the Windows disk.

---

### **A Note from the Lead Architect:**

"Standard IT admins rely on the OS to handle networking for them. By stripping away the IP layer, you are forced to understand how the wire actually works. If you can control a server through ARP, you can control it through anything."

## **7\. Troubleshooting & Debugging (Wireshark)**

Since we are bypassing the standard TCP/IP stack, traditional tools like `ping` or `netstat` will not show our RMM traffic. To debug the **L2-SHADOW** protocol, you must use **Wireshark** or `tcpdump` to inspect the raw Ethernet frames.

### **A. Wireshark Display Filters**

The network will be full of legitimate ARP traffic (gateways, broadcasts). Use these filters to isolate the challenge traffic:

* **Filter by Protocol:** `arp`  
  * *Shows all ARP traffic on the segment.*  
* **Filter by MAC Hopping (V2):** `eth.src[0] == 0x02`  
  * *Locates packets where the "Locally Administered" bit is set, which our Hopping algorithm uses.*  
* **Filter by Specific Field Content:** `arp.src.proto_ipv4 == 1.2.3.4`  
  * *If you are testing the MVP with a static "fake" IP, this isolates those specific packets.*

### **B. Analyzing the Payload**

In Wireshark, look at the **Address Resolution Protocol** header in the packet details pane.

1. **Sender MAC Address (SHA):** In v2, this will change every packet. Ensure the OUI (first 3 bytes) matches your generator's logic.  
2. **Sender IP Address (SPA):** This is where our **4-byte payload** lives.  
   * Right-click this field \-\> **Apply as Column**.  
   * You can now see your command strings (e.g., `INIT`, `HELO`) scrolling in the main packet list.  
3. **Target IP Address (TPA):** In the Linux-to-Windows direction, this field carries the 4-byte command fragment.

### **C. Common Failure Points**

* **Packet Drops:** If your Reassembly logic hangs, check Wireshark for missing **Sequence IDs**. If Seq 4 follows Seq 2, the switch dropped Seq 3\.  
* **Npcap Binding:** If you see packets leaving the Linux box but the Windows Agent isn't "seeing" them, verify that Npcap is bound to the correct network adapter.  
* **Endianness:** If your 4-byte string `HELO` appears as `OLEH` in Wireshark, your Go code is swapping the Byte Order. Ensure you are using `binary.BigEndian` when packing the IP fields.

---

### **Final Submission Checklist**

1. **Compile:** Ensure `agent.exe` is statically linked.  
2. **Deploy:** Run the Ansible playbook and verify the "Print Spooler" service is `Running`.  
3. **Validate:** Open a terminal on Linux and run the `arpshell` binary.  
4. **Proof:** Execute `whoami` and capture the Wireshark trace of the response fragments for the boss.

