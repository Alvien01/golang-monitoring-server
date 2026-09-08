package metrics

import (
	"regexp"
	"strconv"
	"strings"
)

type Snapshot struct {
	CPUPercent   float64
	MemPercent   float64
	MemUsedMB    float64
	MemTotalMB   float64
	DiskPercent  float64
	LoadAvg1Min  float64
	UptimeRaw    string
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
