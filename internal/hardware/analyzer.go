package hardware

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// Analysis represents comprehensive hardware analysis results
type Analysis struct {
	CPU    CPUInfo    `json:"cpu"`
	Memory MemoryInfo `json:"memory"`
	GPU    GPUInfo    `json:"gpu"`
	System SystemInfo `json:"system"`
}

// CPUInfo contains CPU specifications
type CPUInfo struct {
	Cores    int    `json:"cores"`
	Model    string `json:"model"`
	SpeedMHz int    `json:"speed_mhz"`
	CacheKB  int    `json:"cache_kb"`
}

// MemoryInfo contains memory specifications
type MemoryInfo struct {
	TotalGB     float64 `json:"total_gb"`
	AvailableGB float64 `json:"available_gb"`
	UsedPercent float64 `json:"used_percent"`
}

// GPUInfo contains GPU specifications
type GPUInfo struct {
	Available bool   `json:"available"`
	Name      string `json:"name"`
	MemoryMB  int    `json:"memory_mb"`
}

// SystemInfo contains system load information
type SystemInfo struct {
	LoadAvg1  float64 `json:"load_avg_1"`
	LoadAvg5  float64 `json:"load_avg_5"`
	LoadAvg15 float64 `json:"load_avg_15"`
}

// Analyzer provides hardware analysis capabilities
type Analyzer struct{}

// NewAnalyzer creates a new hardware analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// Analyze performs comprehensive hardware analysis
func (a *Analyzer) Analyze() (*Analysis, error) {
	cpu, err := a.analyzeCPU()
	if err != nil {
		return nil, fmt.Errorf("CPU analysis failed: %w", err)
	}
	
	memory, err := a.analyzeMemory()
	if err != nil {
		return nil, fmt.Errorf("memory analysis failed: %w", err)
	}
	
	gpu, err := a.analyzeGPU()
	if err != nil {
		// GPU analysis is optional, log but continue
		gpu = &GPUInfo{Available: false}
	}
	
	system, err := a.analyzeSystem()
	if err != nil {
		return nil, fmt.Errorf("system analysis failed: %w", err)
	}
	
	return &Analysis{
		CPU:    *cpu,
		Memory: *memory,
		GPU:    *gpu,
		System: *system,
	}, nil
}

// analyzeCPU extracts CPU information
func (a *Analyzer) analyzeCPU() (*CPUInfo, error) {
	cores := runtime.NumCPU()
	
	// Read CPU info from /proc/cpuinfo
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return &CPUInfo{Cores: cores}, nil // Fallback with just core count
	}
	defer file.Close()
	
	var model string
	var speedMHz int
	var cacheKB int
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				model = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "cpu MHz") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				if speed, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
					speedMHz = int(speed)
				}
			}
		} else if strings.HasPrefix(line, "cache size") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				cacheStr := strings.TrimSpace(parts[1])
				cacheStr = strings.TrimSuffix(cacheStr, " KB")
				if cache, err := strconv.Atoi(cacheStr); err == nil {
					cacheKB = cache
				}
			}
		}
	}
	
	return &CPUInfo{
		Cores:    cores,
		Model:    model,
		SpeedMHz: speedMHz,
		CacheKB:  cacheKB,
	}, nil
}

// analyzeMemory extracts memory information
func (a *Analyzer) analyzeMemory() (*MemoryInfo, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var totalKB, availableKB int64
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if total, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					totalKB = total
				}
			}
		} else if strings.HasPrefix(line, "MemAvailable:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if available, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					availableKB = available
				}
			}
		}
	}
	
	totalGB := float64(totalKB) / 1024 / 1024
	availableGB := float64(availableKB) / 1024 / 1024
	usedPercent := (totalGB - availableGB) / totalGB * 100
	
	return &MemoryInfo{
		TotalGB:     totalGB,
		AvailableGB: availableGB,
		UsedPercent: usedPercent,
	}, nil
}

// analyzeGPU attempts to detect GPU information
func (a *Analyzer) analyzeGPU() (*GPUInfo, error) {
	// Try nvidia-smi first
	if info, err := a.analyzeNvidiaGPU(); err == nil {
		return info, nil
	}
	
	// Could add other GPU detection methods here (AMD, Intel)
	
	return &GPUInfo{Available: false}, nil
}

// analyzeNvidiaGPU attempts to get NVIDIA GPU information
func (a *Analyzer) analyzeNvidiaGPU() (*GPUInfo, error) {
	// Check if nvidia-smi exists
	if _, err := os.Stat("/usr/bin/nvidia-smi"); err != nil {
		return nil, err
	}
	
	// For now, return basic detection
	// In a full implementation, we would exec nvidia-smi and parse output
	return &GPUInfo{
		Available: true,
		Name:      "NVIDIA detected",
		MemoryMB:  24564, // Would be parsed from nvidia-smi
	}, nil
}

// analyzeSystem extracts system load information
func (a *Analyzer) analyzeSystem() (*SystemInfo, error) {
	file, err := os.Open("/proc/loadavg")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil, fmt.Errorf("could not read load average")
	}
	
	fields := strings.Fields(scanner.Text())
	if len(fields) < 3 {
		return nil, fmt.Errorf("invalid load average format")
	}
	
	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)
	
	return &SystemInfo{
		LoadAvg1:  load1,
		LoadAvg5:  load5,
		LoadAvg15: load15,
	}, nil
}

// GetOptimalWorkerCounts returns optimized worker counts based on hardware
func (a *Analyzer) GetOptimalWorkerCounts(analysis *Analysis) *WorkerConfig {
	cores := analysis.CPU.Cores
	availableGB := analysis.Memory.AvailableGB
	
	// Conservative calculations for stability
	downloadWorkers := min(cores*2, 40)
	extractionWorkers := max(cores-2, 1)
	classificationWorkers := min(5, cores/4) // API rate limited
	indexingWorkers := min(cores/3, 10)
	
	// Adjust based on available memory
	if availableGB < 8 {
		downloadWorkers = min(downloadWorkers, 20)
		extractionWorkers = min(extractionWorkers, 4)
	}
	
	return &WorkerConfig{
		Download:       downloadWorkers,
		Extraction:     extractionWorkers,
		Classification: classificationWorkers,
		Indexing:       indexingWorkers,
	}
}

// WorkerConfig holds optimal worker counts
type WorkerConfig struct {
	Download       int `json:"download"`
	Extraction     int `json:"extraction"`
	Classification int `json:"classification"`
	Indexing       int `json:"indexing"`
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}