package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/christopherluey/clustertui/internal/cluster"
	"github.com/christopherluey/clustertui/internal/slurm"
	"github.com/christopherluey/clustertui/internal/tui/views/submit"
)

func connectCmd(svc *cluster.Service) tea.Cmd {
	return func() tea.Msg {
		if err := svc.Connect(); err != nil {
			return ConnectErrorMsg{Err: err}
		}
		return ConnectedMsg{}
	}
}

func fetchNodesCmd(svc *cluster.Service) tea.Cmd {
	return func() tea.Msg {
		nodes, err := svc.RefreshNodes()
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return NodesLoadedMsg{Nodes: nodes}
	}
}

func fetchJobsCmd(svc *cluster.Service) tea.Cmd {
	return func() tea.Msg {
		jobs, err := svc.RefreshJobs()
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return JobsLoadedMsg{Jobs: jobs}
	}
}

func submitJobCmd(svc *cluster.Service, sub slurm.JobSubmission) tea.Cmd {
	return func() tea.Msg {
		jobID, err := svc.SubmitJob(sub)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return JobSubmittedMsg{JobID: jobID}
	}
}

func cancelJobCmd(svc *cluster.Service, jobID string) tea.Cmd {
	return func() tea.Msg {
		if err := svc.CancelJob(jobID); err != nil {
			return ErrorMsg{Err: err}
		}
		return JobCancelledMsg{JobID: jobID}
	}
}

func listDirCmd(svc *cluster.Service, path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := svc.ListDirectory(path)
		return submit.DirListingMsg{Path: path, Entries: entries, Err: err}
	}
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}
