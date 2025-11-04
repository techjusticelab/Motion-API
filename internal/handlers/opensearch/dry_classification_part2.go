package opensearch

import (
	"fmt"
	"runtime"
)

func (h *DryClassificationHandler) logMemoryUsage(stage string, fileName string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := float64(m.Alloc) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024
	gcCount := m.NumGC

	// Log detailed memory info for first 10 files, then sample every 10th
	static := map[string]int{}
	count := static["logCount"]
	static["logCount"] = count + 1

	if count < 10 || count%10 == 0 {
		fmt.Printf("[MEMORY] %s - File: %s | Alloc: %.1fMB | Sys: %.1fMB | GCs: %d | Goroutines: %d\n",
			stage, fileName, allocMB, sysMB, gcCount, runtime.NumGoroutine())
	}

	// Trigger additional GC if memory usage is high (>500MB allocated)
	if allocMB > 500 {
		fmt.Printf("[MEMORY-WARNING] High memory usage detected: %.1fMB - triggering additional GC\n", allocMB)
		runtime.GC()
		runtime.ReadMemStats(&m)
		newAllocMB := float64(m.Alloc) / 1024 / 1024
		fmt.Printf("[MEMORY-WARNING] After additional GC: %.1fMB (freed %.1fMB)\n", newAllocMB, allocMB-newAllocMB)
	}
}
