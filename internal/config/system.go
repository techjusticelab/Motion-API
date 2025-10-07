package config

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// getSystemMemoryInfo reads memory information from /proc/meminfo
func getSystemMemoryInfo() (totalGB float64, availableGB float64, err error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open /proc/meminfo: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var memTotal, memAvailable int64

	for scanner.Scan() {
		line := scanner.Text()
		
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if val, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
					memTotal = val // Value in KB
				}
			}
		} else if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if val, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
					memAvailable = val // Value in KB
				}
			}
		}
		
		// Break early if we have both values
		if memTotal > 0 && memAvailable > 0 {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("error reading /proc/meminfo: %w", err)
	}

	// Convert KB to GB
	totalGB = float64(memTotal) / (1024 * 1024)
	availableGB = float64(memAvailable) / (1024 * 1024)

	return totalGB, availableGB, nil
}

// detectNVIDIAGPU checks for NVIDIA GPU presence and memory
func detectNVIDIAGPU() (hasGPU bool, memoryMB int) {
	// Try nvidia-smi command first
	if gpuInfo := detectGPUViaNVIDIASMI(); gpuInfo != nil {
		return true, gpuInfo.memoryMB
	}
	
	// Try alternative detection methods
	if hasNVIDIADevice() {
		return true, 0 // GPU present but couldn't determine memory
	}
	
	return false, 0
}

// GPUInfo holds NVIDIA GPU information
type GPUInfo struct {
	name     string
	memoryMB int
}

// detectGPUViaNVIDIASMI uses nvidia-smi to detect GPU information
func detectGPUViaNVIDIASMI() *GPUInfo {
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return nil // nvidia-smi not available or no GPU
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return nil
	}
	
	// Parse first GPU line
	line := strings.TrimSpace(lines[0])
	parts := strings.Split(line, ",")
	if len(parts) < 2 {
		return nil
	}
	
	name := strings.TrimSpace(parts[0])
	memoryStr := strings.TrimSpace(parts[1])
	
	memory, err := strconv.Atoi(memoryStr)
	if err != nil {
		return nil
	}
	
	return &GPUInfo{
		name:     name,
		memoryMB: memory,
	}
}

// hasNVIDIADevice checks for NVIDIA devices in /proc/driver/nvidia
func hasNVIDIADevice() bool {
	// Check if nvidia driver is loaded
	if _, err := os.Stat("/proc/driver/nvidia"); err == nil {
		return true
	}
	
	// Check lspci for NVIDIA devices
	cmd := exec.Command("lspci")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	
	// Look for NVIDIA in the output
	return strings.Contains(strings.ToLower(string(output)), "nvidia")
}

// GetDetailedSystemInfo returns comprehensive system information
func GetDetailedSystemInfo() (*DetailedSystemInfo, error) {
	info := &DetailedSystemInfo{}
	
	// Get hardware info
	hardware, err := DetectHardware()
	if err != nil {
		return nil, fmt.Errorf("failed to detect hardware: %w", err)
	}
	info.Hardware = hardware
	
	// Get CPU information
	info.CPU = getCPUInfo()
	
	// Get disk space information
	info.Disk = getDiskSpaceInfo()
	
	// Get load average
	info.LoadAverage = getLoadAverage()
	
	return info, nil
}

// DetailedSystemInfo contains comprehensive system information
type DetailedSystemInfo struct {
	Hardware    *HardwareInfo `json:"hardware"`
	CPU         *CPUInfo      `json:"cpu"`
	Disk        *DiskInfo     `json:"disk"`
	LoadAverage []float64     `json:"load_average"`
}

// CPUInfo contains detailed CPU information
type CPUInfo struct {
	ModelName     string  `json:"model_name"`
	MHz           float64 `json:"mhz"`
	CacheSize     string  `json:"cache_size"`
	PhysicalCores int     `json:"physical_cores"`
	LogicalCores  int     `json:"logical_cores"`
}

// DiskInfo contains disk space information
type DiskInfo struct {
	TotalGB      float64 `json:"total_gb"`
	AvailableGB  float64 `json:"available_gb"`
	UsedGB       float64 `json:"used_gb"`
	UsagePercent float64 `json:"usage_percent"`
}

// getCPUInfo reads CPU information from /proc/cpuinfo
func getCPUInfo() *CPUInfo {
	info := &CPUInfo{
		LogicalCores: 0,
	}
	
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return info
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	physicalIDs := make(map[string]bool)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		if strings.HasPrefix(line, "processor") {
			info.LogicalCores++
		} else if strings.HasPrefix(line, "model name") {
			if info.ModelName == "" {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info.ModelName = strings.TrimSpace(parts[1])
				}
			}
		} else if strings.HasPrefix(line, "cpu MHz") {
			if info.MHz == 0 {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					if val, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
						info.MHz = val
					}
				}
			}
		} else if strings.HasPrefix(line, "cache size") {
			if info.CacheSize == "" {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info.CacheSize = strings.TrimSpace(parts[1])
				}
			}
		} else if strings.HasPrefix(line, "physical id") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				physicalIDs[strings.TrimSpace(parts[1])] = true
			}
		}
	}
	
	info.PhysicalCores = len(physicalIDs)
	if info.PhysicalCores == 0 {
		info.PhysicalCores = info.LogicalCores // Fallback
	}
	
	return info
}

// getDiskSpaceInfo gets disk space information for current directory
func getDiskSpaceInfo() *DiskInfo {
	cmd := exec.Command("df", "-BG", ".")
	output, err := cmd.Output()
	if err != nil {
		return &DiskInfo{}
	}
	
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return &DiskInfo{}
	}
	
	// Parse df output (skip header)
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return &DiskInfo{}
	}
	
	info := &DiskInfo{}
	
	// Parse sizes (remove 'G' suffix)
	if total, err := strconv.ParseFloat(strings.TrimSuffix(fields[1], "G"), 64); err == nil {
		info.TotalGB = total
	}
	
	if used, err := strconv.ParseFloat(strings.TrimSuffix(fields[2], "G"), 64); err == nil {
		info.UsedGB = used
	}
	
	if avail, err := strconv.ParseFloat(strings.TrimSuffix(fields[3], "G"), 64); err == nil {
		info.AvailableGB = avail
	}
	
	if info.TotalGB > 0 {
		info.UsagePercent = (info.UsedGB / info.TotalGB) * 100
	}
	
	return info
}

// getLoadAverage reads system load average
func getLoadAverage() []float64 {
	file, err := os.Open("/proc/loadavg")
	if err != nil {
		return []float64{0, 0, 0}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return []float64{0, 0, 0}
	}
	
	fields := strings.Fields(scanner.Text())
	if len(fields) < 3 {
		return []float64{0, 0, 0}
	}
	
	var loads []float64
	for i := 0; i < 3; i++ {
		if val, err := strconv.ParseFloat(fields[i], 64); err == nil {
			loads = append(loads, val)
		} else {
			loads = append(loads, 0)
		}
	}
	
	return loads
}

// CheckNVIDIADriverVersion returns NVIDIA driver version if available
func CheckNVIDIADriverVersion() string {
	cmd := exec.Command("nvidia-smi", "--query-gpu=driver_version", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	
	return strings.TrimSpace(string(output))
}

// GetGPUUtilization returns current GPU utilization if available
func GetGPUUtilization() map[string]interface{} {
	result := make(map[string]interface{})
	
	cmd := exec.Command("nvidia-smi", "--query-gpu=utilization.gpu,utilization.memory,temperature.gpu", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		result["available"] = false
		return result
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		result["available"] = false
		return result
	}
	
	// Parse first GPU
	parts := strings.Split(strings.TrimSpace(lines[0]), ",")
	if len(parts) >= 3 {
		result["available"] = true
		
		if gpuUtil, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err == nil {
			result["gpu_utilization_percent"] = gpuUtil
		}
		
		if memUtil, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
			result["memory_utilization_percent"] = memUtil
		}
		
		if temp, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64); err == nil {
			result["temperature_celsius"] = temp
		}
	}
	
	return result
}

// EstimateOptimalConcurrency provides recommendations for different workload types
func EstimateOptimalConcurrency(hardware *HardwareInfo) map[string]int {
	cores := hardware.CPUCores
	memGB := int(hardware.MemoryAvailableGB)
	
	recommendations := make(map[string]int)
	
	// CPU-bound tasks (text extraction, processing)
	recommendations["cpu_bound"] = cores - 1
	if recommendations["cpu_bound"] < 1 {
		recommendations["cpu_bound"] = 1
	}
	
	// I/O-bound tasks (downloads, uploads)
	recommendations["io_bound"] = cores * 2
	if recommendations["io_bound"] > 50 {
		recommendations["io_bound"] = 50
	}
	
	// Memory-bound tasks (large document processing)
	recommendations["memory_bound"] = memGB / 4  // Assume 4GB per task
	if recommendations["memory_bound"] < 2 {
		recommendations["memory_bound"] = 2
	}
	if recommendations["memory_bound"] > cores {
		recommendations["memory_bound"] = cores
	}
	
	// API rate-limited tasks (OpenAI, external services)
	recommendations["rate_limited"] = 5  // Conservative for most APIs
	
	// Mixed workload (balanced)
	recommendations["mixed"] = cores / 2
	if recommendations["mixed"] < 4 {
		recommendations["mixed"] = 4
	}
	
	return recommendations
}