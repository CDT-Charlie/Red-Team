Since your boss is an Ansible fan and you're dealing with multiple Windows Server 2022 targets, the most efficient way to deploy your custom Go agent is through a **Role-based architecture**.

In Ansible, a **Role** allows you to package the .exe, the nssm.exe helper, and the configuration logic into a single, reusable unit. This is perfect for deploying your "ArpRmmAgent" to 2, 5, or 50 servers at once.

---

## **1\. The High-Level Architecture (Ansible Deployment)**

Your Linux Admin box will act as the **Ansible Control Node**. It will push the files over WinRM (the standard way Ansible talks to Windows) to install the ARP-based "hidden" service.

---

## **2\. The Recommended Folder Structure**

Based on your boss's hints, here is how you should organize your project files on your Linux box before deployment:

Plaintext  
arp-rmm-deploy/  
├── ansible.cfg          \# Configures WinRM connection settings  
├── inventory.ini        \# Lists your Windows Server 2022 IPs  
├── site.yml             \# The main entry-point playbook  
└── roles/  
    └── arp\_agent/       \# The specific role for our RMM  
        ├── files/  
        │   ├── agent.exe      \# Your compiled Go binary  
        │   └── nssm.exe       \# The service manager  
        ├── tasks/  
        │   └── main.yml       \# The "how-to" install logic  
        └── vars/  
            └── main.yml       \# Variables like the Service Name

---

## **3\. Step-by-Step Build Sequence (The Ansible Way)**

### **Step 1: Define Your Inventory (inventory.ini)**

Group your Windows servers here.

Ini, TOML  
\[windows\_servers\]  
server01.corp.local  
server02.corp.local

\[windows\_servers:vars\]  
ansible\_user=AdminUser  
ansible\_password=SecretPassword  
ansible\_connection=winrm  
ansible\_winrm\_server\_cert\_validation=ignore

### **Step 2: Create the Install Logic (roles/arp\_agent/tasks/main.yml)**

This is the heart of the deployment. It handles the "Life Cycle" of the agent on the remote Windows boxes.

YAML  
\---  
\- name: Create deployment directory  
  win\_file:  
    path: C:\\ProgramData\\WinTelemetry  
    state: directory

\- name: Copy Agent and NSSM binaries  
  win\_copy:  
    src: "{{ item }}"  
    dest: C:\\ProgramData\\WinTelemetry\\  
  loop:  
    \- agent.exe  
    \- nssm.exe

\- name: Check if Npcap is installed (Requirement for ARP)  
  win\_shell: Get-Service \-Name "npcap"  
  register: npcap\_check  
  ignore\_errors: yes

\- name: Install Agent as a Service using NSSM  
  win\_shell: |  
    ./nssm.exe install "{{ service\_name }}" "C:\\ProgramData\\WinTelemetry\\agent.exe"  
    ./nssm.exe set "{{ service\_name }}" DisplayName "{{ display\_name }}"  
    ./nssm.exe set "{{ service\_name }}" Description "{{ description }}"  
    ./nssm.exe set "{{ service\_name }}" Start SERVICE\_AUTO\_START  
    Start-Service "{{ service\_name }}"  
  args:  
    chdir: C:\\ProgramData\\WinTelemetry\\  
  when: npcap\_check.rc \== 0

### **Step 3: Run the Playbook (site.yml)**

This is the file you actually execute to trigger the whole process.

YAML  
\---  
\- name: Deploy ARP RMM Agent to Windows Cluster  
  hosts: windows\_servers  
  roles:  
    \- arp\_agent

---

## **4\. Advanced "Pro" Features for Ansible**

* **Template-Based PSKs:** Use templates/ to create a unique configuration file for each server. You can give every server a different **Pre-Shared Key** so that if one is compromised, the others remain secure.  
* **Handshake Validation:** Add a final task in your playbook that waits for the Linux box to receive an "Initial ARP Check-in" from the server before marking the deployment as "Successful."  
* **WinRM Hardening:** Since you're using WinRM to deploy, your boss might like it if you use Ansible to *disable* WinRM once the ARP agent is confirmed to be working—effectively "cutting the ladder" behind you.

### **Summary of Files vs. Folders**

| Ansible Component | Purpose for your RMM |
| :---- | :---- |
| files/ | Stores the agent.exe you compiled in Go. |
| vars/ | Stores the "Stealth" names like "Print Spooler Extension". |
| tasks/ | The sequential steps to move files and start the service. |
| inventory/ | The master list of all Windows Servers your boss wants monitored. |

To connect to Windows Server 2022 from a Linux control node, your ansible.cfg needs to be tuned for the **WinRM** (Windows Remote Management) transport rather than the default SSH.

On Windows Server 2022, WinRM often requires specific message encryption settings and certificate handling to function reliably over a network.

---

## **1\. The ansible.cfg File**

Place this file in the root of your arp-rmm-deploy/ directory. It tells Ansible how to behave globally for this project.

Ini, TOML

\[defaults\]

\# 1\. Point to your inventory of Windows Servers

inventory \= inventory.ini

\# 2\. Set the default remote user (can be overridden in host\_vars)

remote\_user \= Administrator

\# 3\. Roles path for our 'arp\_agent' role

roles\_path \= ./roles

\# 4\. Disable host key checking for faster initial deployment

\# (Ensure your environment is secure before doing this in production)

host\_key\_checking \= False

\# 5\. Use the 'yaml' callback for more readable output in the terminal

stdout\_callback \= yaml

\[privilege\_escalation\]

\# Windows doesn't use sudo, but this section is good practice for 

\# cross-platform compatibility.

become \= False

\[powershell\]

\# Ensure Ansible uses the correct execution policy for your Go agent scripts

execution\_policy \= ByPass

\[ssh\_connection\]

\# This section is ignored for Windows, as we use WinRM below.

pipelining \= True

---

## **2\. The High-Level Architecture (The WinRM Pipe)**

While your RMM tool uses **ARP**, Ansible needs **WinRM** (running on ports 5985/5986) to perform the initial installation. Once the agent.exe is installed, you can technically turn WinRM off.

---

## **3\. Step-by-Step Build Sequence (The Connection)**

### **Step 1: Install Dependencies**

For Ansible to speak WinRM, you must install the pywinrm library on your **Linux** box:

Bash

pip install "pywinrm\>=0.3.0"

### **Step 2: Define Connection Variables**

While ansible.cfg handles global settings, you should define the specific Windows connection details in your inventory.ini or group\_vars/windows.yml.

**In your inventory.ini:**

Ini, TOML

\[windows:vars\]

\# Use WinRM instead of SSH

ansible\_connection=winrm

\# Use NTLM or Kerberos (NTLM is easiest for lab environments)

ansible\_winrm\_transport=ntlm

\# Use HTTPS (5986) if possible; use HTTP (5985) for basic challenges

ansible\_winrm\_server\_cert\_validation=ignore

ansible\_port=5985

### **Step 3: Test the Connection**

Before running your full RMM deployment, run a simple "ping" to verify the Linux-to-Windows link:

Bash

ansible windows \-m win\_ping

---

## **4\. Advanced "Pro" Features for Windows 2022**

* **Async Tasks:** If your Go agent installation involves a long-running process (like installing Npcap drivers), use the async keyword in your tasks to prevent Ansible from timing out.  
* **WinRM Hardening:** For Windows Server 2022, you should ideally use **Kerberos** authentication. This requires joining your Linux box to the same domain as the Windows servers, providing much higher security than NTLM.  
* **Custom Filter Plugins:** Your boss mentioned the filter\_plugins/ folder. You could write a small Python filter that takes a server's MAC address and automatically generates the **Initial Sequence ID** for the ARP hopping, so the Ansible playbook and the Go agent are perfectly in sync from second one.

### **Summary Table for the IT Admin**

| Setting | Recommended Value | Reason |
| :---- | :---- | :---- |
| ansible\_connection | winrm | Windows does not natively support SSH for Ansible. |
| ansible\_winrm\_transport | ntlm or credssp | Required for authentication on modern Windows Server. |
| stdout\_callback | yaml | Makes it easier to read error logs when PowerShell fails. |
| host\_key\_checking | False | Prevents the "Accept Fingerprint" prompt during mass deployment. |

To tie everything together, your site.yml acts as the "Master Controller." It maps your group of Windows servers (defined in your inventory.ini) to the specific arp\_agent role you've built.

### **1\. The site.yml Playbook**

Place this in your root arp-rmm-deploy/ directory.

YAML

\---

\- name: Deploy Custom ARP-Based RMM Agent

  hosts: windows\_servers

  gather\_facts: yes


  \# Global variables for the role (can be overridden in host\_vars)

  vars:

    service\_name: "WinNetExtension"

    display\_name: "Windows Network Extension Service"

    description: "Handles low-level hardware resolution and legacy network mapping."

    deploy\_path: "C:\\\\ProgramData\\\\WinNetExt"

    psk\_secret: "S3cur3\_Adm1n\_K3y" \# Match this to your Go code\!

  roles:

    \- role: arp\_agent

      become: no \# We use LocalSystem via NSSM instead of 'become'

---

### **2\. The Execution Command**

On your Linux Admin box, you would trigger the entire deployment with a single command from the root folder:

Bash

ansible-playbook \-i inventory.ini site.yml

---

### **3\. Step-by-Step Deployment Flow (What Happens Internally)**

1. **Connection:** Ansible reads ansible.cfg, sees the winrm transport, and establishes a secure PowerShell session to every server in your inventory.ini.  
2. **Fact Gathering:** Ansible runs Gathers Facts to ensure the target is actually Windows and to check architecture (x64).  
3. **Role Execution:**  
   * **Directory Setup:** Creates C:\\ProgramData\\WinNetExt.  
   * **Binary Push:** Uploads your compiled agent.exe and nssm.exe from your Linux files/ folder.  
   * **Service Registration:** Runs the nssm commands to lock the agent into the Windows Service Control Manager.  
4. **Verification:** The playbook attempts to start the service and checks the exit code.

---

### **4\. Advanced "Pro" Features for the Final Submission**

* **Idempotency:** The win\_copy and win\_file modules are idempotent by nature. If you run the playbook a second time, it won't re-upload the files unless they have changed.  
* **Dynamic PSKs:** In a real-world "Challenge," you shouldn't use the same PSK for every server. You can use an Ansible **Jinja2 filter** to generate a PSK based on the server's BIOS UUID:  
  psk\_secret: "{{ ansible\_system\_vendor | hash('sha256') }}"

**Firewall Management:** You can add a task to ensure the Windows Firewall allows the Npcap driver to bypass standard filtering for ARP:  
YAML  
\- name: Ensure Npcap driver is allowed

  win\_shell: Enable-NetAdapterBinding \-Name "\*" \-ComponentID "ms\_tcpip" \# Example binding logic

* 

---

### **Summary of the Final Package**

| File | Role |
| :---- | :---- |
| **ansible.cfg** | Sets the "Rules of Engagement" (WinRM, Timeout). |
| **inventory.ini** | The "Target List" (Windows Server IPs/Names). |
| **site.yml** | The "Mission Brief" (Assigns the Agent role to the Servers). |
| **roles/arp\_agent/** | The "Toolbox" (Contains the actual code and install steps). |

