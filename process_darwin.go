package watcher

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// start the process. Using a special start process for macOS to be able to set
// a process group ID for all child process, this will help to kill correctly
// the command.
func startProcess(osEnv []string, args ...string) *exec.Cmd {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	cmd.Env = osEnv
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	return cmd
}

// Kill the process and all children. Using a special kill for macOS as the
// original approach using pgrep cause issues.
func kill(proc *os.Process) error {
	if proc == nil {
		return errors.New("nil process")
	}
	return syscall.Kill(-proc.Pid, syscall.SIGKILL)
}
