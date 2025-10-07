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
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// MetricsCollector collects and aggregates system performance metrics
type MetricsCollector struct {
	startTime time.Time
	metrics   map[string]*Metric
	mutex     sync.RWMutex
	
	// Global counters
	totalOperations    int64
	successfulOps      int64
	failedOps          int64
	totalLatency       int64
	operationCount     int64
	
	// System metrics
	cpuUsage           float64
	memoryUsage        float64
	diskUsage          float64
	networkLatency     float64
	
	// Queue metrics
	queueSizes         map[string]int
	queueThroughput    map[string]float64
	
	// Performance tracking
	enabled            bool
	reportingInterval  time.Duration
	lastReport         time.Time
}

// Metric represents a single performance metric
type Metric struct {
	Name        string        `json:"name"`
	Type        MetricType    `json:"type"`
	Value       float64       `json:"value"`
	Unit        string        `json:"unit"`
	Timestamp   time.Time     `json:"timestamp"`
	Tags        map[string]string `json:"tags"`
	History     []MetricPoint `json:"history"`
	mutex       sync.RWMutex
}

// MetricPoint represents a point in time for a metric
type MetricPoint struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// MetricType defines the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge" 
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeTiming    MetricType = "timing"
)

// PerformanceReport contains a comprehensive performance report
type PerformanceReport struct {
	Timestamp           time.Time                `json:"timestamp"`
	Uptime              time.Duration            `json:"uptime"`
	TotalOperations     int64                    `json:"total_operations"`
	SuccessfulOps       int64                    `json:"successful_ops"`
	FailedOps           int64                    `json:"failed_ops"`
	SuccessRate         float64                  `json:"success_rate"`
	OperationsPerSec    float64                  `json:"operations_per_sec"`
	AverageLatency      time.Duration            `json:"average_latency"`
	CPUUsage            float64                  `json:"cpu_usage"`
	MemoryUsage         float64                  `json:"memory_usage"`
	DiskUsage           float64                  `json:"disk_usage"`
	NetworkLatency      time.Duration            `json:"network_latency"`
	QueueMetrics        map[string]*QueueMetrics `json:"queue_metrics"`
	SystemLoad          []float64                `json:"system_load"`
	TopMetrics          []*Metric                `json:"top_metrics"`
	Alerts              []Alert                  `json:"alerts"`
}

// QueueMetrics contains metrics for a specific queue
type QueueMetrics struct {
	Name               string        `json:"name"`
	Size               int           `json:"size"`
	ThroughputPerSec   float64       `json:"throughput_per_sec"`
	AverageWaitTime    time.Duration `json:"average_wait_time"`
	ProcessedItems     int64         `json:"processed_items"`
	FailedItems        int64         `json:"failed_items"`
	ActiveWorkers      int           `json:"active_workers"`
	IdleWorkers        int           `json:"idle_workers"`
}

// Alert represents a performance alert
type Alert struct {
	Level       AlertLevel `json:"level"`
	Message     string     `json:"message"`
	Metric      string     `json:"metric"`
	Value       float64    `json:"value"`
	Threshold   float64    `json:"threshold"`
	Timestamp   time.Time  `json:"timestamp"`
	Acknowledged bool      `json:"acknowledged"`
}

// AlertLevel defines alert severity levels
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime:          time.Now(),
		metrics:            make(map[string]*Metric),
		queueSizes:         make(map[string]int),
		queueThroughput:    make(map[string]float64),
		enabled:            true,
		reportingInterval:  30 * time.Second,
		lastReport:         time.Now(),
	}
}

// RecordOperation records a completed operation
func (mc *MetricsCollector) RecordOperation(success bool, latency time.Duration) {
	if !mc.enabled {
		return
	}
	
	atomic.AddInt64(&mc.totalOperations, 1)
	atomic.AddInt64(&mc.totalLatency, int64(latency))
	atomic.AddInt64(&mc.operationCount, 1)
	
	if success {
		atomic.AddInt64(&mc.successfulOps, 1)
	} else {
		atomic.AddInt64(&mc.failedOps, 1)
	}
	
	// Record timing metric
	mc.RecordTiming("operation_latency", latency, map[string]string{
		"success": func() string {
			if success {
				return "true"
			}
			return "false"
		}(),
	})
}

// RecordCounter increments a counter metric
func (mc *MetricsCollector) RecordCounter(name string, value float64, tags map[string]string) {
	if !mc.enabled {
		return
	}
	
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	metric, exists := mc.metrics[name]
	if !exists {
		metric = &Metric{
			Name:      name,
			Type:      MetricTypeCounter,
			Unit:      "count",
			Tags:      tags,
			History:   make([]MetricPoint, 0),
		}
		mc.metrics[name] = metric
	}
	
	metric.mutex.Lock()
	metric.Value += value
	metric.Timestamp = time.Now()
	metric.History = append(metric.History, MetricPoint{
		Value:     metric.Value,
		Timestamp: metric.Timestamp,
	})
	
	// Keep only last 100 points
	if len(metric.History) > 100 {
		metric.History = metric.History[1:]
	}
	metric.mutex.Unlock()
}

// RecordGauge sets a gauge metric value
func (mc *MetricsCollector) RecordGauge(name string, value float64, unit string, tags map[string]string) {
	if !mc.enabled {
		return
	}
	
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	metric, exists := mc.metrics[name]
	if !exists {
		metric = &Metric{
			Name:      name,
			Type:      MetricTypeGauge,
			Unit:      unit,
			Tags:      tags,
			History:   make([]MetricPoint, 0),
		}
		mc.metrics[name] = metric
	}
	
	metric.mutex.Lock()
	metric.Value = value
	metric.Timestamp = time.Now()
	metric.History = append(metric.History, MetricPoint{
		Value:     value,
		Timestamp: metric.Timestamp,
	})
	
	// Keep only last 100 points
	if len(metric.History) > 100 {
		metric.History = metric.History[1:]
	}
	metric.mutex.Unlock()
}

// RecordTiming records a timing measurement
func (mc *MetricsCollector) RecordTiming(name string, duration time.Duration, tags map[string]string) {
	value := float64(duration.Nanoseconds()) / 1e6 // Convert to milliseconds
	mc.RecordGauge(name, value, "ms", tags)
}

// UpdateSystemMetrics updates system-level metrics
func (mc *MetricsCollector) UpdateSystemMetrics(ctx context.Context) {
	if !mc.enabled {
		return
	}
	
	// Get CPU usage
	cpuUsage := getCPUUsage()
	mc.RecordGauge("cpu_usage", cpuUsage, "percent", nil)
	mc.cpuUsage = cpuUsage
	
	// Get memory usage
	memUsage := getMemoryUsage()
	mc.RecordGauge("memory_usage", memUsage, "percent", nil)
	mc.memoryUsage = memUsage
	
	// Get disk usage
	diskUsage := getDiskUsage()
	mc.RecordGauge("disk_usage", diskUsage, "percent", nil)
	mc.diskUsage = diskUsage
	
	// Get network latency (to external services)
	netLatency := getNetworkLatency()
	mc.RecordGauge("network_latency", netLatency, "ms", nil)
	mc.networkLatency = netLatency
}

// UpdateQueueMetrics updates queue-specific metrics
func (mc *MetricsCollector) UpdateQueueMetrics(queueName string, size int, throughput float64) {
	if !mc.enabled {
		return
	}
	
	mc.mutex.Lock()
	mc.queueSizes[queueName] = size
	mc.queueThroughput[queueName] = throughput
	mc.mutex.Unlock()
	
	mc.RecordGauge("queue_size", float64(size), "items", map[string]string{"queue": queueName})
	mc.RecordGauge("queue_throughput", throughput, "items/sec", map[string]string{"queue": queueName})
}

// GenerateReport generates a comprehensive performance report
func (mc *MetricsCollector) GenerateReport() *PerformanceReport {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	uptime := time.Since(mc.startTime)
	totalOps := atomic.LoadInt64(&mc.totalOperations)
	successOps := atomic.LoadInt64(&mc.successfulOps)
	failedOps := atomic.LoadInt64(&mc.failedOps)
	totalLatency := atomic.LoadInt64(&mc.totalLatency)
	
	var successRate float64
	if totalOps > 0 {
		successRate = float64(successOps) / float64(totalOps) * 100
	}
	
	var opsPerSec float64
	if uptime.Seconds() > 0 {
		opsPerSec = float64(totalOps) / uptime.Seconds()
	}
	
	var avgLatency time.Duration
	if totalOps > 0 {
		avgLatency = time.Duration(totalLatency / totalOps)
	}
	
	// Build queue metrics
	queueMetrics := make(map[string]*QueueMetrics)
	for queueName, size := range mc.queueSizes {
		queueMetrics[queueName] = &QueueMetrics{
			Name:             queueName,
			Size:             size,
			ThroughputPerSec: mc.queueThroughput[queueName],
			// Other metrics would be populated from actual queue implementations
		}
	}
	
	// Get top metrics
	topMetrics := mc.getTopMetrics(10)
	
	// Generate alerts
	alerts := mc.generateAlerts()
	
	return &PerformanceReport{
		Timestamp:        time.Now(),
		Uptime:           uptime,
		TotalOperations:  totalOps,
		SuccessfulOps:    successOps,
		FailedOps:        failedOps,
		SuccessRate:      successRate,
		OperationsPerSec: opsPerSec,
		AverageLatency:   avgLatency,
		CPUUsage:         mc.cpuUsage,
		MemoryUsage:      mc.memoryUsage,
		DiskUsage:        mc.diskUsage,
		NetworkLatency:   time.Duration(mc.networkLatency) * time.Millisecond,
		QueueMetrics:     queueMetrics,
		TopMetrics:       topMetrics,
		Alerts:           alerts,
	}
}

// getTopMetrics returns the most important metrics
func (mc *MetricsCollector) getTopMetrics(limit int) []*Metric {
	metrics := make([]*Metric, 0, len(mc.metrics))
	
	for _, metric := range mc.metrics {
		metrics = append(metrics, metric)
	}
	
	// Sort by importance (simplified)
	// In a real implementation, you'd have more sophisticated sorting
	
	if len(metrics) > limit {
		metrics = metrics[:limit]
	}
	
	return metrics
}

// generateAlerts checks metrics against thresholds and generates alerts
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

// StartPeriodicReporting starts periodic performance reporting
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

// Enable enables metrics collection
func (mc *MetricsCollector) Enable() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.enabled = true
}

// Disable disables metrics collection
func (mc *MetricsCollector) Disable() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.enabled = false
}

// Reset resets all metrics
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

// GetMetric returns a specific metric by name
func (mc *MetricsCollector) GetMetric(name string) (*Metric, bool) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	metric, exists := mc.metrics[name]
	return metric, exists
}

// ListMetrics returns all metric names
func (mc *MetricsCollector) ListMetrics() []string {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	names := make([]string, 0, len(mc.metrics))
	for name := range mc.metrics {
		names = append(names, name)
	}
	
	return names
}

// Export exports metrics in a specific format
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

// Simplified system metric collection functions
// In a real implementation, these would use system-specific APIs

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