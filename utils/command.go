package utils

import (
	"os/exec"
)

func Run(cmd string) ([]byte, error) {
	return exec.Command("/bin/bash", "-c", cmd).CombinedOutput()
}

func RunFile(path string) ([]byte, error) {
	return exec.Command("/bin/bash", path).CombinedOutput()
}
