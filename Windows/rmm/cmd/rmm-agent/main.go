package main

import (
	"flag"
	"log"
	"time"

	"rmm/agent"
	"rmm/shared"
)

func main() {
	interval := flag.Duration("interval", time.Duration(shared.DefaultIntervalSeconds)*time.Second, "Snapshot interval")
	maxSnapshots := flag.Int("max-snapshots", shared.MaxSnapshots(), "Stop after this many snapshots (0 = unlimited)")
	flag.Parse()

	logPath, err := shared.SetupAuditLogger("rmm-agent")
	if err != nil {
		log.Fatalf("[!] failed to configure audit logging: %v", err)
	}
	if err := shared.EnsureDemoMode("rmm-agent"); err != nil {
		log.Fatalf("[!] %v", err)
	}

	stopPath, enabled, err := shared.KillSwitchPresent()
	if err != nil {
		log.Fatalf("[!] failed to inspect stop file: %v", err)
	}
	if enabled {
		log.Fatalf("[!] stop file present at %s", stopPath)
	}

	log.Printf("[AUDIT] rmm agent starting; audit_log=%s snapshot_path=%s stop_file=%s interval=%v max_snapshots=%d",
		logPath, shared.SnapshotPath(), stopPath, *interval, *maxSnapshots)

	runner := agent.NewRunner(*interval, *maxSnapshots)
	if err := runner.Run(); err != nil {
		log.Fatalf("[!] agent error: %v", err)
	}
}
