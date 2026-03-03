package slurm

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// ParseSinfoJSON parses the output of `sinfo --json`.
func ParseSinfoJSON(data string) ([]NodeInfo, error) {
	var resp sinfoResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return nil, err
	}

	var nodes []NodeInfo
	for _, n := range resp.Nodes {
		gpuType, gpuTotal, gpuAlloc := extractGRES(n.Gres, n.GresUsed)
		nodes = append(nodes, NodeInfo{
			Name:       n.Name,
			State:      flattenState(n.State),
			Partitions: n.Partitions,
			CPUsTotal:  n.CPUs,
			CPUsAlloc:  n.AllocCPUs,
			MemTotal:   n.RealMemory,
			MemAlloc:   n.AllocMemory,
			GPUType:    gpuType,
			GPUsTotal:  gpuTotal,
			GPUsAlloc:  gpuAlloc,
		})
	}
	return nodes, nil
}

type sinfoResponse struct {
	Nodes []sinfoNode `json:"nodes"`
}

type sinfoNode struct {
	Name        string   `json:"name"`
	State       any      `json:"state"`
	Partitions  []string `json:"partitions"`
	CPUs        int      `json:"cpus"`
	AllocCPUs   int      `json:"alloc_cpus"`
	RealMemory  int      `json:"real_memory"`
	AllocMemory int      `json:"alloc_memory"`
	Gres        string   `json:"gres"`
	GresUsed    string   `json:"gres_used"`
}

func flattenState(v any) string {
	switch s := v.(type) {
	case string:
		return strings.ToUpper(s)
	case []any:
		var parts []string
		for _, e := range s {
			if str, ok := e.(string); ok {
				parts = append(parts, str)
			}
		}
		return strings.ToUpper(strings.Join(parts, "+"))
	default:
		return "UNKNOWN"
	}
}

var gresRe = regexp.MustCompile(`gpu:([^:]+):(\d+)`)

func extractGRES(gres, gresUsed string) (gpuType string, total, alloc int) {
	m := gresRe.FindStringSubmatch(gres)
	if len(m) == 3 {
		gpuType = m[1]
		total, _ = strconv.Atoi(m[2])
	} else if strings.Contains(gres, "gpu:") {
		// gpu:N format without type
		parts := strings.Split(gres, ":")
		if len(parts) >= 2 {
			total, _ = strconv.Atoi(parts[len(parts)-1])
		}
	}

	mu := gresRe.FindStringSubmatch(gresUsed)
	if len(mu) == 3 {
		alloc, _ = strconv.Atoi(mu[2])
	} else if strings.Contains(gresUsed, "gpu:") {
		parts := strings.Split(gresUsed, ":")
		if len(parts) >= 2 {
			alloc, _ = strconv.Atoi(parts[len(parts)-1])
		}
	}

	return gpuType, total, alloc
}

// ParseSinfoDelimited parses pipe-delimited sinfo output.
// Format: "%N|%P|%T|%c|%C|%m|%e|%G"
func ParseSinfoDelimited(data string) ([]NodeInfo, error) {
	var nodes []NodeInfo
	for _, line := range strings.Split(strings.TrimSpace(data), "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "|", 8)
		if len(fields) < 8 {
			continue
		}

		cpusTotal, _ := strconv.Atoi(fields[3])
		// %C = alloc/idle/other/total
		cpuParts := strings.Split(fields[4], "/")
		cpusAlloc := 0
		if len(cpuParts) >= 1 {
			cpusAlloc, _ = strconv.Atoi(cpuParts[0])
		}

		memTotal, _ := strconv.Atoi(fields[5])
		memFree, _ := strconv.Atoi(fields[6])
		memAlloc := memTotal - memFree

		gpuType, gpuTotal, gpuAlloc := extractGRES(fields[7], "")

		nodes = append(nodes, NodeInfo{
			Name:       fields[0],
			State:      strings.ToUpper(fields[2]),
			Partitions: strings.Split(fields[1], ","),
			CPUsTotal:  cpusTotal,
			CPUsAlloc:  cpusAlloc,
			MemTotal:   memTotal,
			MemAlloc:   memAlloc,
			GPUType:    gpuType,
			GPUsTotal:  gpuTotal,
			GPUsAlloc:  gpuAlloc,
		})
	}
	return nodes, nil
}

// SinfoJSONCommand returns the sinfo command for JSON output.
func SinfoJSONCommand() string {
	return "sinfo --json"
}

// SinfoDelimitedCommand returns the sinfo command for pipe-delimited output.
func SinfoDelimitedCommand() string {
	return `sinfo -N -o "%N|%P|%T|%c|%C|%m|%e|%G" --noheader`
}
