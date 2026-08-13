//go:build darwin || linux

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

func lockApplicationInstanceFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}

func unlockApplicationInstanceFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
