package cluster

import (
	"fmt"
	"strings"

	"github.com/christopherluey/sc/internal/config"
	"github.com/christopherluey/sc/internal/slurm"
	"github.com/christopherluey/sc/internal/ssh"
)

type Service struct {
	cfg          *config.Config
	ssh          *ssh.Client
	supportsJSON bool
}

func New(cfg *config.Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) Connect() error {
	c, err := ssh.Dial(
		s.cfg.SSH.Host,
		s.cfg.SSH.Port,
		s.cfg.SSH.User,
		s.cfg.SSH.UseAgent,
		s.cfg.SSH.IdentityFile,
	)
	if err != nil {
		return err
	}
	s.ssh = c

	// Probe JSON support
	stdout, _, err := s.ssh.RunCommand("sinfo --json 2>/dev/null | head -c 1")
	if err == nil && strings.TrimSpace(stdout) == "{" {
		s.supportsJSON = true
	}

	return nil
}

func (s *Service) IsConnected() bool {
	return s.ssh != nil && s.ssh.IsConnected()
}

func (s *Service) User() string {
	return s.cfg.SSH.User
}

func (s *Service) RefreshNodes() ([]slurm.NodeInfo, error) {
	if s.ssh == nil {
		return nil, fmt.Errorf("not connected")
	}

	if s.supportsJSON {
		stdout, _, err := s.ssh.RunCommand(slurm.SinfoJSONCommand())
		if err == nil {
			nodes, parseErr := slurm.ParseSinfoJSON(stdout)
			if parseErr == nil {
				return nodes, nil
			}
		}
	}

	// Fallback to delimited
	stdout, _, err := s.ssh.RunCommand(slurm.SinfoDelimitedCommand())
	if err != nil {
		return nil, fmt.Errorf("sinfo: %w", err)
	}
	return slurm.ParseSinfoDelimited(stdout)
}

func (s *Service) RefreshJobs() ([]slurm.JobInfo, error) {
	if s.ssh == nil {
		return nil, fmt.Errorf("not connected")
	}

	if s.supportsJSON {
		stdout, _, err := s.ssh.RunCommand(slurm.SqueueJSONCommand())
		if err == nil {
			jobs, parseErr := slurm.ParseSqueueJSON(stdout)
			if parseErr == nil {
				return jobs, nil
			}
		}
	}

	// Fallback to delimited
	stdout, _, err := s.ssh.RunCommand(slurm.SqueueDelimitedCommand())
	if err != nil {
		return nil, fmt.Errorf("squeue: %w", err)
	}
	return slurm.ParseSqueueDelimited(stdout)
}

func (s *Service) SubmitJob(sub slurm.JobSubmission) (string, error) {
	if s.ssh == nil {
		return "", fmt.Errorf("not connected")
	}

	cmd := slurm.BuildSbatchCommand(sub)
	stdout, stderr, err := s.ssh.RunCommand(cmd)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("sbatch: %s", msg)
	}
	return slurm.ParseSbatchOutput(stdout)
}

func (s *Service) CancelJob(jobID string) error {
	if s.ssh == nil {
		return fmt.Errorf("not connected")
	}

	cmd := slurm.CancelCommand(jobID)
	_, stderr, err := s.ssh.RunCommand(cmd)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("scancel: %s", msg)
	}
	return nil
}

type DirEntry struct {
	Name  string
	IsDir bool
}

func (s *Service) ListDirectory(path string) ([]DirEntry, error) {
	if s.ssh == nil {
		return nil, fmt.Errorf("not connected")
	}

	cmd := "ls -1pA " + path
	stdout, stderr, err := s.ssh.RunCommand(cmd)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return ParseLsOutput(stdout), nil
}

func ParseLsOutput(output string) []DirEntry {
	var entries []DirEntry
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		isDir := strings.HasSuffix(line, "/")
		// Trim type suffixes: / * @ = | >
		name := strings.TrimRight(line, "/*@=|>")
		if name == "" {
			continue
		}
		entries = append(entries, DirEntry{Name: name, IsDir: isDir})
	}
	return entries
}

func (s *Service) Close() error {
	if s.ssh != nil {
		return s.ssh.Close()
	}
	return nil
}
