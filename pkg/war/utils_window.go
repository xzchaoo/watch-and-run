//go:build windows

package war

import (
	"os"
	"os/exec"
	"syscall"
)

func enableProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

func killProcessGroup(process *os.Process, signal os.Signal) error {
	// https://github.com/loov/watchrun/tree/master
	return process.Signal(signal)
}
