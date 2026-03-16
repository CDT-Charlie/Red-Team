# CHRONOSFEAR

Author: Caroline Richards
Email: cpr9499@rit.edu

## Description
Chronosfear is a Command and Control server meant to run on remote Linux clients and communicate over NTP using the extension fields in the headers. The server can handle communication with multiple clients and send commands to one client at a time or to all connected clients. 

The server uses Textual to create a simple UI within the terminal, and maintain a visual representation of connected clients and returned command results. 

*Note: The server currently supports piping and redirects but cannot currently cannot open text editors like vim and nano. 

## Usage
1. Start server `python3 chronosfear_server.py`
2. Populate `inventory.ini` file with clients and credentials to login over SSH
3. Run ansible playbook with `ansible-playbook chrono_client.yml -i inventory.ini`
4. When the playbook finishes the server should begin to receive connections from clients. 