package util

import (
	"errors"
	"os/exec"
)

func Shell(s string) error {
	cmd := exec.Command("/bin/bash", "-c", s)
	ret, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	return errors.New(s + ": " + string(ret))
}
