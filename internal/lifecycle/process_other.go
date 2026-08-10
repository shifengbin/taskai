//go:build !windows

package lifecycle

import "os/exec"

func ConfigureBackgroundProcess(*exec.Cmd) {}
