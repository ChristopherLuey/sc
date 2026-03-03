package tui

import "github.com/christopherluey/sc/internal/slurm"

type ConnectedMsg struct{}

type ConnectErrorMsg struct{ Err error }

type NodesLoadedMsg struct{ Nodes []slurm.NodeInfo }

type JobsLoadedMsg struct{ Jobs []slurm.JobInfo }

type JobSubmittedMsg struct{ JobID string }

type JobCancelledMsg struct{ JobID string }

type ErrorMsg struct{ Err error }

type TickMsg struct{}

type JSONProbeMsg struct{ Supported bool }
