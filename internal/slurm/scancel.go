package slurm

import "fmt"

// CancelCommand returns the scancel command for a job ID.
func CancelCommand(jobID string) string {
	return fmt.Sprintf("scancel %s", jobID)
}
