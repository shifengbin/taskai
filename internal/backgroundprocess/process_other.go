//go:build !windows

package backgroundprocess

import "os/exec"

func Configure(*exec.Cmd) {}
