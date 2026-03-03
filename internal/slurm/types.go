package slurm

type NodeInfo struct {
	Name       string
	State      string
	Partitions []string
	CPUsTotal  int
	CPUsAlloc  int
	MemTotal   int // MB
	MemAlloc   int // MB
	GPUType    string
	GPUsTotal  int
	GPUsAlloc  int
}

type JobInfo struct {
	JobID     string
	Name      string
	User      string
	Partition string
	State     string
	TimeUsed  string
	TimeLimit string
	Nodes     string
	GPUs      int
	CPUs      int
	Memory    string
	Reason    string
}

type JobSubmission struct {
	Account      string
	Partition    string
	JobName      string
	GPUType      string
	GPUs         int
	CPUs         int
	Memory       string
	TimeLimit    string
	OutputPath   string
	WorkDir      string
	NodeList     string
	ExcludeNodes string
	ScriptPath   string
	Command      string
}
