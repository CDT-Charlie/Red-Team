# Docker Chaos Engine 
Author: Sonia Cachon, sjc2210@rit.edu

## Tool Overview
the Chaos Engine is meant to disrupt Containers via the Docker Remote API (port 2375).
This tool either pauses, stops, or renames docker contaiers, in random order. every 6 minutes. 

File is hidden under /usr/share/zoneinfo/RTSC-26 
schedules execution via a root crontab every 6 minutes 

## How to USE

run this command: ansible-playbook -i inventory.ini deploy_engine.yml

logs file: /tmp/.engine_status.log

Before Offical Run confirm ips on ChaosEngine.py and Inventory.ini
Could do just the Boxes  that sole purpose is Docker ( so 10.2.1.7 and 10.1.1.7), or all boxes. 
