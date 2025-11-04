package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

func (mc *MetricsCollector) generateAlerts() []Alert {
	var alerts []Alert

	// CPU usage alert
	if mc.cpuUsage > 90 {
		alerts = append(alerts, Alert{
			Level:     AlertLevelCritical,
			Message:   "High CPU usage detected",
			Metric:    "cpu_usage",
			Value:     mc.cpuUsage,
			Threshold: 90.0,
			Timestamp: time.Now(),
		})
	} else if mc.cpuUsage > 80 {
		alerts = append(alerts, Alert{
			Level:     AlertLevelWarning,
			Message:   "Elevated CPU usage",
			Metric:    "cpu_usage",
			Value:     mc.cpuUsage,
			Threshold: 80.0,
			Timestamp: time.Now(),
		})
	}

	// Memory usage alert
	if mc.memoryUsage > 95 {
		alerts = append(alerts, Alert{
			Level:     AlertLevelCritical,
			Message:   "Critical memory usage",
			Metric:    "memory_usage",
			Value:     mc.memoryUsage,
			Threshold: 95.0,
			Timestamp: time.Now(),
		})
	} else if mc.memoryUsage > 85 {
		alerts = append(alerts, Alert{
			Level:     AlertLevelWarning,
			Message:   "High memory usage",
			Metric:    "memory_usage",
			Value:     mc.memoryUsage,
			Threshold: 85.0,
			Timestamp: time.Now(),
		})
	}

	// Network latency alert
	if mc.networkLatency > 1000 { // 1 second
		alerts = append(alerts, Alert{
			Level:     AlertLevelWarning,
			Message:   "High network latency detected",
			Metric:    "network_latency",
			Value:     mc.networkLatency,
			Threshold: 1000.0,
			Timestamp: time.Now(),
		})
	}

	// Queue size alerts
	for _, size := range mc.queueSizes {
		if size > 1000 {
			alerts = append(alerts, Alert{
				Level:     AlertLevelWarning,
				Message:   "Large queue size detected",
				Metric:    "queue_size",
				Value:     float64(size),
				Threshold: 1000.0,
				Timestamp: time.Now(),
			})
		}
	}

	return alerts
}

func (mc *MetricsCollector) StartPeriodicReporting(ctx context.Context, interval time.Duration, callback func(*PerformanceReport)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Update system metrics
			mc.UpdateSystemMetrics(ctx)

			// Generate and send report
			report := mc.GenerateReport()
			if callback != nil {
				callback(report)
			}
		}
	}
}

func (mc *MetricsCollector) Enable() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.enabled = true
}

func (mc *MetricsCollector) Disable() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.enabled = false
}

func (mc *MetricsCollector) Reset() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.metrics = make(map[string]*Metric)
	atomic.StoreInt64(&mc.totalOperations, 0)
	atomic.StoreInt64(&mc.successfulOps, 0)
	atomic.StoreInt64(&mc.failedOps, 0)
	atomic.StoreInt64(&mc.totalLatency, 0)
	atomic.StoreInt64(&mc.operationCount, 0)
	mc.startTime = time.Now()
}

func (mc *MetricsCollector) GetMetric(name string) (*Metric, bool) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	metric, exists := mc.metrics[name]
	return metric, exists
}

func (mc *MetricsCollector) ListMetrics() []string {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	names := make([]string, 0, len(mc.metrics))
	for name := range mc.metrics {
		names = append(names, name)
	}

	return names
}

func (mc *MetricsCollector) Export(format string) ([]byte, error) {
	report := mc.GenerateReport()

	switch format {
	case "json":
		return exportJSON(report)
	case "prometheus":
		return exportPrometheus(mc.metrics)
	default:
		return exportJSON(report)
	}
}

func getCPUUsage() float64 {
	// Read CPU usage from /proc/stat
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0.0
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return 0.0
	}

	// Parse first CPU line: cpu  user nice system idle iowait irq softirq steal guest guest_nice
	fields := strings.Fields(lines[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0.0
	}

	var idle, total int64
	for i := 1; i < len(fields); i++ {
		val, err := strconv.ParseInt(fields[i], 10, 64)
		if err != nil {
			continue
		}
		total += val
		if i == 4 { // idle is the 4th field
			idle = val
		}
	}

	if total == 0 {
		return 0.0
	}

	usage := float64(total-idle) / float64(total) * 100
	return math.Max(0, math.Min(100, usage))
}

func getMemoryUsage() float64 {
	// Read memory usage from /proc/meminfo
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0.0
	}

	var memTotal, memAvailable int64
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			if val, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				memTotal = val
			}
		case "MemAvailable:":
			if val, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				memAvailable = val
			}
		}

		if memTotal > 0 && memAvailable > 0 {
			break
		}
	}

	if memTotal == 0 {
		return 0.0
	}

	usage := float64(memTotal-memAvailable) / float64(memTotal) * 100
	return math.Max(0, math.Min(100, usage))
}

func getDiskUsage() float64 {
	// Get disk usage for root filesystem
	var stat syscall.Statfs_t
	err := syscall.Statfs("/", &stat)
	if err != nil {
		return 0.0
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	if total == 0 {
		return 0.0
	}

	usage := float64(used) / float64(total) * 100
	return math.Max(0, math.Min(100, usage))
}

func getNetworkLatency() float64 {
	// Ping Google DNS to measure network latency
	start := time.Now()

	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", timeout)
	if err != nil {
		return 0.0
	}
	defer conn.Close()

	latency := time.Since(start)
	return float64(latency.Nanoseconds()) / 1e6 // Convert to milliseconds
}

func exportJSON(report *PerformanceReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

func exportPrometheus(metrics map[string]*Metric) ([]byte, error) {
	var result strings.Builder

	for _, metric := range metrics {
		// Write metric help
		result.WriteString(fmt.Sprintf("# HELP %s %s\n", metric.Name, metric.Name))
		result.WriteString(fmt.Sprintf("# TYPE %s %s\n", metric.Name, strings.ToLower(string(metric.Type))))

		// Write metric value with tags
		if len(metric.Tags) > 0 {
			var tags []string
			for k, v := range metric.Tags {
				tags = append(tags, fmt.Sprintf("%s=\"%s\"", k, v))
			}
			result.WriteString(fmt.Sprintf("%s{%s} %.2f\n",
				metric.Name, strings.Join(tags, ","), metric.Value))
		} else {
			result.WriteString(fmt.Sprintf("%s %.2f\n", metric.Name, metric.Value))
		}
	}

	return []byte(result.String()), nil
}
