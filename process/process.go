package process

import (
	"syscall"
	"time"

	"github.com/shirou/gopsutil/process"
)

type Process struct {
	PID       int        // Process ID
	PPID      int        // Parent Process ID
	Command   string     // Command used to start the process
	User      string     // User who started the process
	Priority  int        // Scheduling Priority of the process
	Nice      int        // Nice value of the process
	VirtualM  uint64     // Virtual memory
	SharedM   uint64     // Shared memory
	ResidentM uint64     // Resident memory
	State     rune       // R (running),S (sleeping), Z (zombie)
	CPU       float32    // Percent CPU used
	Memory    float32    // Percent Memory used
	Time      time.Time  //
	Children  []*Process // Children of the process
	Depth     int        // Depth of tree
	Expanded  bool       // Process children should be visible or not
}

func BuildTree() ([]*Process, error) {
	var processes []*Process           // Array to store the processes
	lookup := make(map[int32]*Process) // map for fast lookup of pid->process
	procs, err := process.Processes()  // get processes
	if err != nil {
		return nil, err
	}
	for _, proc := range procs {
		pid := proc.Pid
		ppid, _ := proc.Ppid()
		name, _ := proc.Name()
		nice, _ := proc.Nice()
		user, _ := proc.Username()
		cpu, _ := proc.CPUPercent()
		memory, _ := proc.MemoryPercent()
		node := &Process{
			PID:      int(pid),
			PPID:     int(ppid),
			Command:  name,
			Nice:     int(nice),
			User:     user,
			CPU:      float32(cpu),
			Memory:   memory,
			Children: []*Process{},
			Expanded: false,
		}
		lookup[pid] = node
		processes = append(processes, node)
	}
	var roots []*Process
	for _, proc := range processes {
		if parent, exists := lookup[int32(proc.PPID)]; exists {
			parent.Children = append(parent.Children, proc)
		} else {
			roots = append(roots, proc) // process with pid 1(root process)
		}
	}
	setDepth(roots, 0)

	return roots, nil
}
func setDepth(nodes []*Process, depth int) {
	for _, node := range nodes {
		node.Depth = depth
		if len(node.Children) > 0 {
			setDepth(node.Children, depth+1)
		}
	}
}

func FlattenVisible(nodes []*Process) []*Process {
	var result []*Process
	for _, node := range nodes {
		result = append(result, node)
		if node.Expanded && len(node.Children) > 0 {
			result = append(result, FlattenVisible(node.Children)...)
		}
	}
	return result
}

func CascadingGracefulKill(proc *Process) error {
	for _, child := range proc.Children {
		if err := CascadingGracefulKill(child); err != nil {
			return err
		}
	}
	if err := syscall.Kill(proc.PID, syscall.SIGTERM); err != nil {
		return err
	}
	return nil
}
func CascadingForcefulKill(proc *Process) error {
	for _, child := range proc.Children {
		if err := CascadingForcefulKill(child); err != nil {
			return err
		}
	}
	if err := syscall.Kill(proc.PID, syscall.SIGKILL); err != nil {
		return err
	}
	return nil
}
