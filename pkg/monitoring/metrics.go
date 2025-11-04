package monitoring

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type MetricsCollector struct {
	startTime time.Time
	metrics   map[string]*Metric
	mutex     sync.RWMutex

	// Global counters
	totalOperations int64
	successfulOps   int64
	failedOps       int64
	totalLatency    int64
	operationCount  int64

	// System metrics
	cpuUsage       float64
	memoryUsage    float64
	diskUsage      float64
	networkLatency float64

	// Queue metrics
	queueSizes      map[string]int
	queueThroughput map[string]float64

	// Performance tracking
	enabled           bool
	reportingInterval time.Duration
	lastReport        time.Time
}

type Metric struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Unit      string            `json:"unit"`
	Timestamp time.Time         `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
	History   []MetricPoint     `json:"history"`
	mutex     sync.RWMutex
}

type MetricPoint struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeTiming    MetricType = "timing"
)

type PerformanceReport struct {
	Timestamp        time.Time                `json:"timestamp"`
	Uptime           time.Duration            `json:"uptime"`
	TotalOperations  int64                    `json:"total_operations"`
	SuccessfulOps    int64                    `json:"successful_ops"`
	FailedOps        int64                    `json:"failed_ops"`
	SuccessRate      float64                  `json:"success_rate"`
	OperationsPerSec float64                  `json:"operations_per_sec"`
	AverageLatency   time.Duration            `json:"average_latency"`
	CPUUsage         float64                  `json:"cpu_usage"`
	MemoryUsage      float64                  `json:"memory_usage"`
	DiskUsage        float64                  `json:"disk_usage"`
	NetworkLatency   time.Duration            `json:"network_latency"`
	QueueMetrics     map[string]*QueueMetrics `json:"queue_metrics"`
	SystemLoad       []float64                `json:"system_load"`
	TopMetrics       []*Metric                `json:"top_metrics"`
	Alerts           []Alert                  `json:"alerts"`
}

type QueueMetrics struct {
	Name             string        `json:"name"`
	Size             int           `json:"size"`
	ThroughputPerSec float64       `json:"throughput_per_sec"`
	AverageWaitTime  time.Duration `json:"average_wait_time"`
	ProcessedItems   int64         `json:"processed_items"`
	FailedItems      int64         `json:"failed_items"`
	ActiveWorkers    int           `json:"active_workers"`
	IdleWorkers      int           `json:"idle_workers"`
}

type Alert struct {
	Level        AlertLevel `json:"level"`
	Message      string     `json:"message"`
	Metric       string     `json:"metric"`
	Value        float64    `json:"value"`
	Threshold    float64    `json:"threshold"`
	Timestamp    time.Time  `json:"timestamp"`
	Acknowledged bool       `json:"acknowledged"`
}

type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime:         time.Now(),
		metrics:           make(map[string]*Metric),
		queueSizes:        make(map[string]int),
		queueThroughput:   make(map[string]float64),
		enabled:           true,
		reportingInterval: 30 * time.Second,
		lastReport:        time.Now(),
	}
}

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

func (mc *MetricsCollector) RecordCounter(name string, value float64, tags map[string]string) {
	if !mc.enabled {
		return
	}

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	metric, exists := mc.metrics[name]
	if !exists {
		metric = &Metric{
			Name:    name,
			Type:    MetricTypeCounter,
			Unit:    "count",
			Tags:    tags,
			History: make([]MetricPoint, 0),
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

func (mc *MetricsCollector) RecordGauge(name string, value float64, unit string, tags map[string]string) {
	if !mc.enabled {
		return
	}

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	metric, exists := mc.metrics[name]
	if !exists {
		metric = &Metric{
			Name:    name,
			Type:    MetricTypeGauge,
			Unit:    unit,
			Tags:    tags,
			History: make([]MetricPoint, 0),
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

func (mc *MetricsCollector) RecordTiming(name string, duration time.Duration, tags map[string]string) {
	value := float64(duration.Nanoseconds()) / 1e6 // Convert to milliseconds
	mc.RecordGauge(name, value, "ms", tags)
}

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
