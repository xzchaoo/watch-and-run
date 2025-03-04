//go:build unix

package war

import (
	"os"
	"os/exec"
	"syscall"
)

func enableProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcessGroup(process *os.Process, signal syscall.Signal) error {
	pgid, err := syscall.Getpgid(process.Pid)
	if err != nil {
		return err
	}
	return syscall.Kill(-pgid, signal)
}
