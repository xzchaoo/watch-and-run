package war

import (
	"github.com/fatih/color"
	"log"
	"os/exec"
	"syscall"
	"time"
)

func killCmd(hint string, execCmd *exec.Cmd, wait <-chan error, termTimeout time.Duration) error {
	if termTimeout > 0 {
		err := killProcessGroup(execCmd.Process, syscall.SIGTERM)
		log.Println(color.YellowString("%s: send SIGTERM: %v", hint, err))
		select {
		case <-time.NewTimer(termTimeout).C:
			log.Println(color.RedString("%s: send SIGKILL: %v", hint, err))
			return killProcessGroup(execCmd.Process, syscall.SIGKILL)
		case <-wait:
			return nil
		}
	} else {
		return killProcessGroup(execCmd.Process, syscall.SIGKILL)
	}
}
