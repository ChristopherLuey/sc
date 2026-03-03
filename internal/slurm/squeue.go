package slurm

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ParseSqueueJSON parses the output of `squeue --json`.
func ParseSqueueJSON(data string) ([]JobInfo, error) {
	var resp squeueResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return nil, err
	}

	var jobs []JobInfo
	for _, j := range resp.Jobs {
		gpus := extractJobGPUs(j.TresAllocStr, j.TresReqStr)
		jobs = append(jobs, JobInfo{
			JobID:     fmt.Sprintf("%d", j.JobID),
			Name:      j.Name,
			User:      j.UserName,
			Partition: j.Partition,
			State:     flattenState(j.JobState),
			TimeUsed:  formatDuration(j.RunTime),
			TimeLimit: formatTimeLimit(j.TimeLimit),
			Nodes:     j.NodeList,
			GPUs:      gpus,
			CPUs:      j.CPUs,
			Memory:    formatMemory(j.Memory),
			Reason:    flattenReason(j.StateReason),
		})
	}
	return jobs, nil
}

type squeueResponse struct {
	Jobs []squeueJob `json:"jobs"`
}

type squeueJob struct {
	JobID       int    `json:"job_id"`
	Name        string `json:"name"`
	UserName    string `json:"user_name"`
	Partition   string `json:"partition"`
	JobState    any    `json:"job_state"`
	RunTime     any    `json:"run_time"`
	TimeLimit   any    `json:"time_limit"`
	NodeList    string `json:"node_list"`
	CPUs        int    `json:"cpus"`
	Memory      any    `json:"memory"`
	StateReason any    `json:"state_reason"`
	TresAllocStr string `json:"tres_alloc_str"`
	TresReqStr   string `json:"tres_req_str"`
}

func extractJobGPUs(tresAlloc, tresReq string) int {
	for _, tres := range []string{tresAlloc, tresReq} {
		for _, part := range strings.Split(tres, ",") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 && strings.Contains(kv[0], "gres/gpu") {
				n, _ := strconv.Atoi(kv[1])
				if n > 0 {
					return n
				}
			}
		}
	}
	return 0
}

func formatDuration(v any) string {
	switch d := v.(type) {
	case float64:
		secs := int(d)
		h := secs / 3600
		m := (secs % 3600) / 60
		s := secs % 60
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	case map[string]any:
		// Slurm JSON sometimes returns {set: true, number: N}
		if n, ok := d["number"]; ok {
			if f, ok := n.(float64); ok {
				secs := int(f)
				h := secs / 3600
				m := (secs % 3600) / 60
				s := secs % 60
				return fmt.Sprintf("%d:%02d:%02d", h, m, s)
			}
		}
		return "N/A"
	case string:
		return d
	default:
		return "N/A"
	}
}

func formatTimeLimit(v any) string {
	switch d := v.(type) {
	case float64:
		mins := int(d)
		if mins == 0 {
			return "UNLIMITED"
		}
		h := mins / 60
		m := mins % 60
		return fmt.Sprintf("%d:%02d:00", h, m)
	case map[string]any:
		if n, ok := d["number"]; ok {
			if f, ok := n.(float64); ok {
				mins := int(f)
				h := mins / 60
				m := mins % 60
				return fmt.Sprintf("%d:%02d:00", h, m)
			}
		}
		return "N/A"
	case string:
		return d
	default:
		return "N/A"
	}
}

func formatMemory(v any) string {
	switch m := v.(type) {
	case float64:
		mb := int(m)
		if mb >= 1024 {
			return fmt.Sprintf("%dG", mb/1024)
		}
		return fmt.Sprintf("%dM", mb)
	case map[string]any:
		if n, ok := m["number"]; ok {
			if f, ok := n.(float64); ok {
				mb := int(f)
				if mb >= 1024 {
					return fmt.Sprintf("%dG", mb/1024)
				}
				return fmt.Sprintf("%dM", mb)
			}
		}
		return "N/A"
	case string:
		return m
	default:
		return "N/A"
	}
}

func flattenReason(v any) string {
	switch r := v.(type) {
	case string:
		if r == "None" {
			return ""
		}
		return r
	default:
		return ""
	}
}

// ParseSqueueDelimited parses pipe-delimited squeue output.
// Format: "%i|%j|%u|%P|%T|%M|%l|%N|%b|%C|%m|%R"
func ParseSqueueDelimited(data string) ([]JobInfo, error) {
	var jobs []JobInfo
	for _, line := range strings.Split(strings.TrimSpace(data), "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "|", 12)
		if len(fields) < 12 {
			continue
		}

		gpus := 0
		if fields[8] != "" {
			// %b = tres-per-node, e.g. "gres:gpu:2"
			parts := strings.Split(fields[8], ":")
			if len(parts) >= 3 {
				gpus, _ = strconv.Atoi(parts[2])
			} else if len(parts) >= 2 {
				gpus, _ = strconv.Atoi(parts[1])
			}
		}

		cpus, _ := strconv.Atoi(fields[9])

		reason := fields[11]
		if reason == "(None)" || reason == "None" {
			reason = ""
		}

		jobs = append(jobs, JobInfo{
			JobID:     fields[0],
			Name:      fields[1],
			User:      fields[2],
			Partition: fields[3],
			State:     fields[4],
			TimeUsed:  fields[5],
			TimeLimit: fields[6],
			Nodes:     fields[7],
			GPUs:      gpus,
			CPUs:      cpus,
			Memory:    fields[10],
			Reason:    reason,
		})
	}
	return jobs, nil
}

// SqueueJSONCommand returns the squeue command for JSON output.
func SqueueJSONCommand() string {
	return "squeue --json"
}

// SqueueDelimitedCommand returns the squeue command for pipe-delimited output.
func SqueueDelimitedCommand() string {
	return `squeue -o "%i|%j|%u|%P|%T|%M|%l|%N|%b|%C|%m|%R" --noheader`
}
