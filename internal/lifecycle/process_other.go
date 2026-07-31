//go:build !windows

package lifecycle

import "os/exec"

func configureBackgroundProcess(*exec.Cmd) {}
