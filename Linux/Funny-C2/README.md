# Funny C2
## Author: Swapnil Acharjee
## Deploying Agents
Create inventory file with targets listed under \[targets\].  
List the hostname of the command server in `command_server` under \[targets:vars\]
Run `ansible-playbook -i inventory.ini playbook deploy-c2.yml`

## Running Control Server
### Dependencies
* Textual
* Flask
* A Fully Qualified Domain Name
* Signed Certificate (LetsEncrypt works)
    * cert.pem
    * privkey.pem
### Deployment
Run `python main.py`. You may need to run with privileges to properly host on port 443.  
You need the endpoint to be hosted publicly and advertized over DNS.
You must have your cert files within the same directory as main.py