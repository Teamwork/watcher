// +build !darwin

package watcher

import (
	"errors"
	"os"
	"os/exec"

	"github.com/shirou/gopsutil/process"
)

// start the process
func startProcess(osEnv []string, args ...string) *exec.Cmd {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = osEnv
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start() // nolint: errcheck
	return cmd
}

// Kill the process and all children
func kill(proc *os.Process) error {
	if proc == nil {
		return errors.New("nil process")
	}
	var kfn func(p *process.Process) error
	kfn = func(p *process.Process) error {
		// this uses pgrep :/
		children, err := p.Children()
		if err != process.ErrorNoChildren && err != nil {
			return err
		}
		for _, c := range children {
			if err := kfn(c); err != nil {
				return err
			}
		}
		return p.Kill()
	}

	p, err := process.NewProcess(int32(proc.Pid))
	if err != nil {
		return err
	}
	return kfn(p)
}
