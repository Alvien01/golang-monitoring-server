package metrics

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

type ProcessInfo struct {
	PID     int     `json:"pid"`
	User    string  `json:"user"`
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"mem"`
	Command string  `json:"command"`
}

type CompoundMetrics struct {
	Uptime       string
	CPUPercent   float64
	MemUsedMB    float64
	MemTotalMB   float64
	MemPercent   float64
	DiskPercent  float64
	LoadAvg      float64
	TopProcesses []ProcessInfo
}

// CompoundSSHCommand menghasilkan satu baris perintah gabungan untuk dieksekusi via SSH
func CompoundSSHCommand() string {
	return `echo "===UPTIME==="; uptime -p 2>/dev/null || uptime; ` +
		`echo "===CPU==="; top -bn1 | grep "Cpu(s)"; ` +
		`echo "===MEM==="; free -m; ` +
		`echo "===DISK==="; df -h /; ` +
		`echo "===LOAD==="; cat /proc/loadavg; ` +
		`echo "===PROCS==="; ps -eo pid,user,%cpu,%mem,comm --sort=-%cpu | head -n 6`
}

// ParseCompoundOutput memecah output compound SSH command ke dalam struct CompoundMetrics
func ParseCompoundOutput(raw string) CompoundMetrics {
	var res CompoundMetrics
	sections := splitSections(raw)

	if val, ok := sections["UPTIME"]; ok {
		res.Uptime = ParseUptime(val)
	}
	if val, ok := sections["CPU"]; ok {
		res.CPUPercent = ParseCPU(val)
	}
	if val, ok := sections["MEM"]; ok {
		res.MemUsedMB, res.MemTotalMB, res.MemPercent = ParseMem(val)
	}
	if val, ok := sections["DISK"]; ok {
		res.DiskPercent = ParseDisk(val)
	}
	if val, ok := sections["LOAD"]; ok {
		res.LoadAvg = ParseLoadAvg(val)
	}
	if val, ok := sections["PROCS"]; ok {
		res.TopProcesses = ParseTopProcesses(val)
	}

	return res
}

func splitSections(raw string) map[string]string {
	result := make(map[string]string)
	tags := []string{"UPTIME", "CPU", "MEM", "DISK", "LOAD", "PROCS"}

	for i, tag := range tags {
		startTag := "===" + tag + "==="
		startIdx := strings.Index(raw, startTag)
		if startIdx == -1 {
			continue
		}
		contentStart := startIdx + len(startTag)

		endIdx := len(raw)
		if i+1 < len(tags) {
			nextTag := "===" + tags[i+1] + "==="
			if idx := strings.Index(raw[contentStart:], nextTag); idx != -1 {
				endIdx = contentStart + idx
			}
		}

		result[tag] = strings.TrimSpace(raw[contentStart:endIdx])
	}

	return result
}

func ParseUptime(output string) string {
	line := strings.TrimSpace(output)
	line = strings.TrimPrefix(line, "up ")
	if line == "" {
		return "Unknown"
	}
	return line
}

func ParseCPU(output string) float64 {
	re := regexp.MustCompile(`(\d+\.\d+)\s*id`)
	match := re.FindStringSubmatch(output)
	if len(match) < 2 {
		return 0
	}
	idle, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0
	}
	return 100 - idle
}

func ParseMem(output string) (usedMB, totalMB, percent float64) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "Mem:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				totalMB, _ = strconv.ParseFloat(fields[1], 64)
				usedMB, _ = strconv.ParseFloat(fields[2], 64)
				if totalMB > 0 {
					percent = (usedMB / totalMB) * 100
				}
			}
			break
		}
	}
	return
}

func ParseDisk(output string) float64 {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "%") {
			fields := strings.Fields(line)
			for _, f := range fields {
				if strings.HasSuffix(f, "%") {
					val, err := strconv.ParseFloat(strings.TrimSuffix(f, "%"), 64)
					if err == nil {
						return val
					}
				}
			}
		}
	}
	return 0
}

func ParseLoadAvg(output string) float64 {
	fields := strings.Fields(output)
	if len(fields) > 0 {
		val, err := strconv.ParseFloat(fields[0], 64)
		if err == nil {
			return val
		}
	}
	return 0
}

func ParseTopProcesses(output string) []ProcessInfo {
	lines := strings.Split(output, "\n")
	var procs []ProcessInfo
	for idx, line := range lines {
		line = strings.TrimSpace(line)
		if idx == 0 || line == "" {
			// Lewati header (PID USER %CPU %MEM COMMAND)
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		pid, _ := strconv.Atoi(fields[0])
		user := fields[1]
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)
		cmd := strings.Join(fields[4:], " ")

		procs = append(procs, ProcessInfo{
			PID:     pid,
			User:    user,
			CPU:     cpu,
			Memory:  mem,
			Command: cmd,
		})
	}
	return procs
}

// EncodeProcessesToJSON mengubah slice ProcessInfo ke JSON string
func EncodeProcessesToJSON(procs []ProcessInfo) string {
	if len(procs) == 0 {
		return "[]"
	}
	bytes, err := json.Marshal(procs)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}
