//+build windows

package internal

import (
	"os/exec"
	"syscall"
)

func ExitCode(err error) int {
	if err, ok := err.(*exec.ExitError); ok {
		if status, ok := err.ProcessState.Sys().(syscall.WaitStatus); ok {
			return int(status.ExitCode)
		}
	}
	return 2
}
