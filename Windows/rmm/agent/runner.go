package agent

import (
	"context"
	"log"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"rmm/shared"
	"rmm/telemetry"
)

type Runner struct {
	Collector    *telemetry.Collector
	Interval     time.Duration
	MaxSnapshots int
}

func NewRunner(interval time.Duration, maxSnapshots int) *Runner {
	return &Runner{
		Collector:    telemetry.NewCollector(),
		Interval:     interval,
		MaxSnapshots: maxSnapshots,
	}
}

func (r *Runner) Run() error {
	ctx, cancelSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelSignals()

	ticker := time.NewTicker(r.Interval)
	defer ticker.Stop()

	count := 0
	for {
		if r.MaxSnapshots > 0 && count >= r.MaxSnapshots {
			log.Printf("[AUDIT] snapshot budget reached: %d", r.MaxSnapshots)
			return nil
		}
		if stopPath, enabled, err := shared.KillSwitchPresent(); err != nil {
			log.Printf("[AUDIT] failed to inspect stop file: %v", err)
		} else if enabled {
			log.Printf("[AUDIT] stop file detected at %s; exiting", stopPath)
			return nil
		}

		snapshotCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		snapshot, err := r.Collector.CollectAndPersist(snapshotCtx)
		cancel()
		if err != nil {
			log.Printf("[AUDIT] collection failed: %v", err)
		} else {
			log.Printf("[AUDIT] wrote snapshot %d to %s: host=%s processes=%d services=%d interfaces=%d arp=%d anomalies=%d",
				count+1,
				r.Collector.SnapshotPath,
				snapshot.Hostname,
				len(snapshot.Processes),
				len(snapshot.Services),
				len(snapshot.Network.Interfaces),
				len(snapshot.ARP.Entries),
				len(snapshot.ARP.Anomalies),
			)
			if len(snapshot.ARP.Anomalies) > 0 {
				log.Printf("[AUDIT] arp anomalies: %s", shared.SummarizeForAudit(strings.Join(snapshot.ARP.Anomalies, "; "), shared.AuditPreviewLimit))
			}
		}
		count++

		select {
		case <-ctx.Done():
			log.Printf("[AUDIT] shutdown requested")
			return nil
		case <-ticker.C:
		}
	}
}
