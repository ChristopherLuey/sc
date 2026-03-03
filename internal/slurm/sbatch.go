package slurm

import (
	"fmt"
	"regexp"
	"strings"
)

// BuildSbatchCommand constructs an sbatch command from a JobSubmission.
func BuildSbatchCommand(sub JobSubmission) string {
	var args []string

	if sub.Account != "" {
		args = append(args, "--account="+sub.Account)
	}
	if sub.Partition != "" {
		args = append(args, "--partition="+sub.Partition)
	}
	if sub.JobName != "" {
		args = append(args, "--job-name="+sub.JobName)
	}
	if sub.GPUs > 0 {
		gres := fmt.Sprintf("--gres=gpu")
		if sub.GPUType != "" && sub.GPUType != "any" {
			gres += fmt.Sprintf(":%s:%d", sub.GPUType, sub.GPUs)
		} else {
			gres += fmt.Sprintf(":%d", sub.GPUs)
		}
		args = append(args, gres)
	}
	if sub.CPUs > 0 {
		args = append(args, fmt.Sprintf("--cpus-per-task=%d", sub.CPUs))
	}
	if sub.Memory != "" {
		args = append(args, "--mem="+sub.Memory)
	}
	if sub.TimeLimit != "" {
		args = append(args, "--time="+sub.TimeLimit)
	}
	if sub.OutputPath != "" {
		args = append(args, "--output="+sub.OutputPath)
	}
	if sub.WorkDir != "" {
		args = append(args, "--chdir="+sub.WorkDir)
	}
	if sub.NodeList != "" {
		args = append(args, "--nodelist="+sub.NodeList)
	}
	if sub.ExcludeNodes != "" {
		args = append(args, "--exclude="+sub.ExcludeNodes)
	}

	cmd := "sbatch " + strings.Join(args, " ")

	if sub.ScriptPath != "" {
		cmd += " " + sub.ScriptPath
	} else if sub.Command != "" {
		cmd += " --wrap=" + shellQuote(sub.Command)
	}

	return cmd
}

var sbatchIDRe = regexp.MustCompile(`Submitted batch job (\d+)`)

// ParseSbatchOutput extracts the job ID from sbatch output.
func ParseSbatchOutput(output string) (string, error) {
	m := sbatchIDRe.FindStringSubmatch(output)
	if len(m) < 2 {
		return "", fmt.Errorf("unexpected sbatch output: %s", strings.TrimSpace(output))
	}
	return m[1], nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
