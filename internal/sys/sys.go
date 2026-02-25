package sys

import (
	"os"
	"runtime"
	"runtime/debug"
	"time"
)

// Tuning sets the GOMAXPROCS and GC settings for the application.
func Tuning() {
	// Only set GOMAXPROCS if it hasn't been set via env var
	if os.Getenv("GOMAXPROCS") == "" {
		// GOMAXPROCS=2 → limits OS thread count. A reverse proxy is I/O-bound;
		// goroutines park on network I/O and need almost no CPU. Each OS thread
		// has a ~1 MB stack that is NEVER freed, so fewer threads = much less RSS.
		runtime.GOMAXPROCS(2)
	}

	// Only set GOGC if it hasn't been set via env var
	if os.Getenv("GOGC") == "" {
		// GOGC=20  → GC triggers at 20% heap growth (default 100)
		debug.SetGCPercent(20)
	}

	// Only set GOMEMLIMIT if it hasn't been set via env var
	if os.Getenv("GOMEMLIMIT") == "" {
		// GOMEMLIMIT → soft ceiling the runtime tries to stay under
		debug.SetMemoryLimit(8 << 20) // 8 MiB
	}
}

// PeriodicGC forces a GC + scavenge every 30 s to return pages to the OS
// when traffic is idle, keeping RSS as low as possible.
func PeriodicGC() {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for range tick.C {
		runtime.GC()
		debug.FreeOSMemory()
	}
}
